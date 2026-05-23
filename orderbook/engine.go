/*
FILE: orderbook/engine.go

DESCRIPTION:
Local L2 order book engine for the Bybit V5 orderbook.{depth}.{symbol}
topic. Detailed package overview lives in doc.go.

ENTITIES:
  - Engine     — one instance per symbol.
  - Snapshot   — full book state (resets the engine).
  - Delta      — incremental update (validated by "u" + 1).
  - ApplyResult — outcome of an apply call: Gap classifier + post-apply
                  metrics so callers can log/observe without re-locking.
  - GapKind    — gap categories (None / Sequence / Initial / ServiceRestart).

ALGORITHM:
  1. ApplySnapshot drops the local state, copies bids/asks, sorts them,
     and stamps lastU / lastSeq / lastTsMs.
  2. ApplyDelta:
       a. If engine has never received a snapshot → Gap=Initial, dirty=true,
          state unchanged.
       b. If d.UpdateID == 1 → Gap=ServiceRestart, dirty=true. Bybit
          publishes u=1 immediately after a service restart; the engine
          MUST be reseeded with a fresh snapshot before applying further
          deltas.
       c. If d.UpdateID != lastU + 1 → Gap=Sequence, dirty=true,
          state unchanged.
       d. Otherwise the delta is applied: levels with size==0 are removed,
          others are inserted/updated; trailing entries past maxDepth are
          dropped.

NUMERICS:
All prices and sizes are stored as shopspring/decimal.Decimal; the engine
itself never converts to float64.
*/

package orderbook

import (
	"sort"
	"sync"

	"github.com/shopspring/decimal"
)

// Level — one order book level inside the engine. Profile packages
// (linears, spot, ...) own their own public OrderBookLevel struct and
// convert at the package boundary; keeping the engine's level type
// independent from any profile package avoids the import cycle that
// would otherwise prevent the spot profile from sharing the engine.
type Level struct {
	Price decimal.Decimal
	Size  decimal.Decimal
}

// SideKind identifies the order book side in internal helpers.
type SideKind uint8

const (
	// SideBid — bids (sorted descending by price).
	SideBid SideKind = iota
	// SideAsk — asks (sorted ascending by price).
	SideAsk
)

// GapKind categorises gap detections returned by ApplyDelta.
type GapKind uint8

const (
	// GapNone — the delta was applied cleanly.
	GapNone GapKind = iota
	// GapInitial — engine has not received a snapshot yet; the delta was
	// dropped.
	GapInitial
	// GapSequence — d.UpdateID != lastU + 1.
	GapSequence
	// GapServiceRestart — delta carries u==1, indicating Bybit's matcher
	// service restarted; engine must be re-seeded with a snapshot.
	GapServiceRestart
)

// String returns a short human-readable label for logs/metrics.
func (g GapKind) String() string {
	switch g {
	case GapInitial:
		return "initial"
	case GapSequence:
		return "sequence"
	case GapServiceRestart:
		return "service_restart"
	default:
		return "none"
	}
}

// Snapshot — a full order book state. Created by the dispatcher from the
// "snapshot"-typed WS push or from a REST GetOrderBook call.
type Snapshot struct {
	Symbol   string
	Bids     []Level
	Asks     []Level
	UpdateID int64
	SeqID    int64
	TsMs     int64
}

// Delta — incremental order book update from a "delta"-typed WS push.
type Delta struct {
	Symbol   string
	Bids     []Level
	Asks     []Level
	UpdateID int64
	SeqID    int64
	TsMs     int64
}

// ApplyResult describes the outcome of applying a snapshot/delta.
type ApplyResult struct {
	Gap      GapKind
	UpdateID int64
	SeqID    int64
	TsMs     int64
	BidsLen  int
	AsksLen  int
}

// Engine — local order book engine for one symbol.
type Engine struct {
	symbol   string
	maxDepth int

	mu       sync.RWMutex
	bids     []Level
	asks     []Level
	lastU    int64
	lastSeq  int64
	lastTsMs int64
	primed   bool
	dirty    bool
}

// NewEngine creates an empty engine. maxDepth caps the local book depth
// (default 200, matching the typical Bybit linear orderbook topic).
func NewEngine(symbol string, maxDepth int) *Engine {
	if maxDepth <= 0 {
		maxDepth = 200
	}
	return &Engine{
		symbol:   symbol,
		maxDepth: maxDepth,
		bids:     make([]Level, 0, maxDepth),
		asks:     make([]Level, 0, maxDepth),
	}
}

// Symbol returns the symbol associated with the engine.
func (e *Engine) Symbol() string { return e.symbol }

// IsDirty reports whether the engine needs a resync (after Gap*).
func (e *Engine) IsDirty() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.dirty
}

// LastUpdateID returns the last applied "u" value (0 before first snapshot).
func (e *Engine) LastUpdateID() int64 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.lastU
}

// LastSeqID returns the last applied "seq" value.
func (e *Engine) LastSeqID() int64 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.lastSeq
}

// ApplySnapshot replaces the local state with the snapshot. Always
// returns Gap=GapNone — a snapshot is the canonical way to clear a
// dirty engine.
func (e *Engine) ApplySnapshot(s Snapshot) ApplyResult {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.bids = copyLevelsSortedDesc(s.Bids, e.maxDepth)
	e.asks = copyLevelsSortedAsc(s.Asks, e.maxDepth)
	e.lastU = s.UpdateID
	e.lastSeq = s.SeqID
	e.lastTsMs = s.TsMs
	e.primed = true
	e.dirty = false

	return ApplyResult{
		Gap:      GapNone,
		UpdateID: s.UpdateID,
		SeqID:    s.SeqID,
		TsMs:     s.TsMs,
		BidsLen:  len(e.bids),
		AsksLen:  len(e.asks),
	}
}

// ApplyDelta applies an incremental update. See doc.go for the exact
// gap-detection rules.
func (e *Engine) ApplyDelta(d Delta) ApplyResult {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.primed {
		e.dirty = true
		return ApplyResult{
			Gap:      GapInitial,
			UpdateID: e.lastU,
			SeqID:    e.lastSeq,
			TsMs:     e.lastTsMs,
			BidsLen:  len(e.bids),
			AsksLen:  len(e.asks),
		}
	}
	if d.UpdateID == 1 {
		e.dirty = true
		return ApplyResult{
			Gap:      GapServiceRestart,
			UpdateID: e.lastU,
			SeqID:    e.lastSeq,
			TsMs:     e.lastTsMs,
			BidsLen:  len(e.bids),
			AsksLen:  len(e.asks),
		}
	}
	if d.UpdateID != e.lastU+1 {
		e.dirty = true
		return ApplyResult{
			Gap:      GapSequence,
			UpdateID: e.lastU,
			SeqID:    e.lastSeq,
			TsMs:     e.lastTsMs,
			BidsLen:  len(e.bids),
			AsksLen:  len(e.asks),
		}
	}

	var i int
	for i = 0; i < len(d.Bids); i++ {
		e.applyLevelLocked(SideBid, d.Bids[i])
	}
	for i = 0; i < len(d.Asks); i++ {
		e.applyLevelLocked(SideAsk, d.Asks[i])
	}
	e.trimLocked()
	e.lastU = d.UpdateID
	e.lastSeq = d.SeqID
	e.lastTsMs = d.TsMs

	return ApplyResult{
		Gap:      GapNone,
		UpdateID: d.UpdateID,
		SeqID:    d.SeqID,
		TsMs:     d.TsMs,
		BidsLen:  len(e.bids),
		AsksLen:  len(e.asks),
	}
}

// MarkResynced clears the dirty flag and stamps the engine with the new
// (u, seq, ts). Called by the dispatcher after manually fetching a fresh
// REST snapshot — the caller is responsible for also pushing the
// snapshot via ApplySnapshot if the local state needs to be replaced.
// Use ApplySnapshot when you have the full state in hand.
func (e *Engine) MarkResynced(updateID, seqID, tsMs int64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.lastU = updateID
	e.lastSeq = seqID
	e.lastTsMs = tsMs
	e.primed = true
	e.dirty = false
}

// TopLevels returns a copy of the top-n bid/ask levels. n ≤ 0 returns
// the full local state.
func (e *Engine) TopLevels(n int) (bids, asks []Level) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	var nb int = len(e.bids)
	var na int = len(e.asks)
	if n > 0 {
		if n < nb {
			nb = n
		}
		if n < na {
			na = n
		}
	}
	bids = make([]Level, nb)
	asks = make([]Level, na)
	copy(bids, e.bids[:nb])
	copy(asks, e.asks[:na])
	return bids, asks
}

// BestBidAsk returns the best (top-of-book) bid/ask price + size pairs.
// All four values are decimal.Zero when the corresponding side is empty.
func (e *Engine) BestBidAsk() (bidPx, bidSz, askPx, askSz decimal.Decimal) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if len(e.bids) > 0 {
		bidPx = e.bids[0].Price
		bidSz = e.bids[0].Size
	}
	if len(e.asks) > 0 {
		askPx = e.asks[0].Price
		askSz = e.asks[0].Size
	}
	return bidPx, bidSz, askPx, askSz
}

// applyLevelLocked applies one level. Size 0 → remove. e.mu must be held.
func (e *Engine) applyLevelLocked(side SideKind, lvl Level) {
	var slice *[]Level
	var less func(a, b decimal.Decimal) bool
	if side == SideBid {
		slice = &e.bids
		less = func(a, b decimal.Decimal) bool { return a.GreaterThan(b) }
	} else {
		slice = &e.asks
		less = func(a, b decimal.Decimal) bool { return a.LessThan(b) }
	}
	var arr []Level = *slice
	var idx int = sort.Search(len(arr), func(i int) bool {
		return !less(arr[i].Price, lvl.Price)
	})
	if idx < len(arr) && arr[idx].Price.Equal(lvl.Price) {
		if lvl.Size.IsZero() {
			arr = append(arr[:idx], arr[idx+1:]...)
		} else {
			arr[idx].Size = lvl.Size
		}
	} else if !lvl.Size.IsZero() {
		arr = append(arr, Level{})
		copy(arr[idx+1:], arr[idx:])
		arr[idx] = lvl
	}
	*slice = arr
}

// trimLocked clamps both sides to maxDepth.
func (e *Engine) trimLocked() {
	if len(e.bids) > e.maxDepth {
		e.bids = e.bids[:e.maxDepth]
	}
	if len(e.asks) > e.maxDepth {
		e.asks = e.asks[:e.maxDepth]
	}
}

// copyLevelsSortedDesc copies and sorts by price descending (bids).
// Truncates to max levels.
func copyLevelsSortedDesc(src []Level, max int) []Level {
	var out []Level = make([]Level, 0, len(src))
	var i int
	for i = 0; i < len(src); i++ {
		if src[i].Size.IsZero() {
			continue
		}
		out = append(out, src[i])
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Price.GreaterThan(out[j].Price)
	})
	if len(out) > max {
		out = out[:max]
	}
	return out
}

// copyLevelsSortedAsc copies and sorts by price ascending (asks).
// Truncates to max levels.
func copyLevelsSortedAsc(src []Level, max int) []Level {
	var out []Level = make([]Level, 0, len(src))
	var i int
	for i = 0; i < len(src); i++ {
		if src[i].Size.IsZero() {
			continue
		}
		out = append(out, src[i])
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Price.LessThan(out[j].Price)
	})
	if len(out) > max {
		out = out[:max]
	}
	return out
}
