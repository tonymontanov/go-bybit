/*
FILE: orderbook/engine_test.go

DESCRIPTION:
Unit tests for the Bybit V5 order book engine. Verify:
  - Snapshot replaces the book and stamps lastU/lastSeq/lastTsMs.
  - Delta gap detection (initial / sequence / service-restart).
  - Delta application (insert / update / delete).
  - TopLevels / BestBidAsk are consistent with the applied state.
  - Concurrent readers + a single writer pass `-race`.
*/

package orderbook

import (
	"sync"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/tonymontanov/go-bybit/linears/types"
)

func dq(s string) decimal.Decimal {
	var d decimal.Decimal
	var err error
	d, err = decimal.NewFromString(s)
	if err != nil {
		panic(err)
	}
	return d
}

func lvl(p, s string) types.OrderBookLevel {
	return types.OrderBookLevel{Price: dq(p), Size: dq(s)}
}

func TestEngine_ApplySnapshot_StampsState(t *testing.T) {
	t.Parallel()
	var e *Engine = NewEngine("BTCUSDT", 200)
	var res ApplyResult = e.ApplySnapshot(Snapshot{
		Symbol:   "BTCUSDT",
		Bids:     []types.OrderBookLevel{lvl("100", "1"), lvl("101", "0.5")},
		Asks:     []types.OrderBookLevel{lvl("102", "2"), lvl("103", "1.5")},
		UpdateID: 100,
		SeqID:    9001,
		TsMs:     1700000000000,
	})
	if res.Gap != GapNone {
		t.Fatalf("Gap: got %v", res.Gap)
	}
	if res.BidsLen != 2 || res.AsksLen != 2 {
		t.Fatalf("len: bids=%d asks=%d", res.BidsLen, res.AsksLen)
	}
	if e.LastUpdateID() != 100 || e.LastSeqID() != 9001 {
		t.Fatalf("stamps: u=%d seq=%d", e.LastUpdateID(), e.LastSeqID())
	}
	var bids, asks = e.TopLevels(0)
	// Bids must be sorted descending — best bid is 101.
	if !bids[0].Price.Equal(dq("101")) {
		t.Fatalf("best bid: got %v", bids[0].Price)
	}
	// Asks must be sorted ascending — best ask is 102.
	if !asks[0].Price.Equal(dq("102")) {
		t.Fatalf("best ask: got %v", asks[0].Price)
	}
}

func TestEngine_ApplyDelta_InitialBeforeSnapshot(t *testing.T) {
	t.Parallel()
	var e *Engine = NewEngine("BTCUSDT", 200)
	var res = e.ApplyDelta(Delta{Symbol: "BTCUSDT", UpdateID: 5})
	if res.Gap != GapInitial {
		t.Fatalf("Gap: got %v", res.Gap)
	}
	if !e.IsDirty() {
		t.Fatalf("engine must be dirty after Initial gap")
	}
}

func TestEngine_ApplyDelta_SequenceGap(t *testing.T) {
	t.Parallel()
	var e *Engine = NewEngine("BTCUSDT", 200)
	e.ApplySnapshot(Snapshot{Symbol: "BTCUSDT", UpdateID: 100})
	// expected next u = 101; deliver 103 → sequence gap.
	var res = e.ApplyDelta(Delta{UpdateID: 103})
	if res.Gap != GapSequence {
		t.Fatalf("Gap: got %v", res.Gap)
	}
	if !e.IsDirty() {
		t.Fatalf("engine must be dirty")
	}
}

func TestEngine_ApplyDelta_ServiceRestart(t *testing.T) {
	t.Parallel()
	var e *Engine = NewEngine("BTCUSDT", 200)
	e.ApplySnapshot(Snapshot{Symbol: "BTCUSDT", UpdateID: 999})
	var res = e.ApplyDelta(Delta{UpdateID: 1})
	if res.Gap != GapServiceRestart {
		t.Fatalf("Gap: got %v", res.Gap)
	}
	if !e.IsDirty() {
		t.Fatalf("engine must be dirty after ServiceRestart")
	}
}

func TestEngine_ApplyDelta_HappyPath(t *testing.T) {
	t.Parallel()
	var e *Engine = NewEngine("BTCUSDT", 200)
	e.ApplySnapshot(Snapshot{
		Symbol:   "BTCUSDT",
		Bids:     []types.OrderBookLevel{lvl("100", "1"), lvl("99", "0.5")},
		Asks:     []types.OrderBookLevel{lvl("101", "1"), lvl("102", "0.5")},
		UpdateID: 100,
		SeqID:    9001,
		TsMs:     1700000000000,
	})
	// Insert a new bid level 99.5 (between 100 and 99), update 101 to 1.7,
	// and remove ask 102 (size=0).
	var res = e.ApplyDelta(Delta{
		UpdateID: 101,
		SeqID:    9002,
		TsMs:     1700000001000,
		Bids:     []types.OrderBookLevel{lvl("99.5", "0.3")},
		Asks: []types.OrderBookLevel{
			lvl("101", "1.7"),
			lvl("102", "0"),
		},
	})
	if res.Gap != GapNone {
		t.Fatalf("Gap: got %v", res.Gap)
	}
	if e.IsDirty() {
		t.Fatalf("engine must NOT be dirty after a clean delta")
	}
	var bids, asks = e.TopLevels(0)
	// Bids sorted desc: 100, 99.5, 99
	if len(bids) != 3 || !bids[0].Price.Equal(dq("100")) || !bids[1].Price.Equal(dq("99.5")) || !bids[2].Price.Equal(dq("99")) {
		t.Fatalf("bids: got %v", bids)
	}
	// Asks: 101 updated to 1.7, 102 removed.
	if len(asks) != 1 || !asks[0].Price.Equal(dq("101")) || !asks[0].Size.Equal(dq("1.7")) {
		t.Fatalf("asks: got %v", asks)
	}
}

func TestEngine_BestBidAsk(t *testing.T) {
	t.Parallel()
	var e *Engine = NewEngine("BTCUSDT", 200)
	var bidPx, bidSz, askPx, askSz = e.BestBidAsk()
	if !bidPx.IsZero() || !bidSz.IsZero() || !askPx.IsZero() || !askSz.IsZero() {
		t.Fatalf("empty engine must return zero best bid/ask, got %v %v %v %v", bidPx, bidSz, askPx, askSz)
	}
	e.ApplySnapshot(Snapshot{
		Symbol:   "BTCUSDT",
		Bids:     []types.OrderBookLevel{lvl("100", "1.5")},
		Asks:     []types.OrderBookLevel{lvl("101", "0.5")},
		UpdateID: 100,
	})
	bidPx, bidSz, askPx, askSz = e.BestBidAsk()
	if !bidPx.Equal(dq("100")) || !bidSz.Equal(dq("1.5")) || !askPx.Equal(dq("101")) || !askSz.Equal(dq("0.5")) {
		t.Fatalf("best: got %v %v %v %v", bidPx, bidSz, askPx, askSz)
	}
}

func TestEngine_MarkResynced_ClearsDirty(t *testing.T) {
	t.Parallel()
	var e *Engine = NewEngine("BTCUSDT", 200)
	e.ApplySnapshot(Snapshot{Symbol: "BTCUSDT", UpdateID: 100})
	e.ApplyDelta(Delta{UpdateID: 999}) // gap
	if !e.IsDirty() {
		t.Fatalf("setup: must be dirty")
	}
	e.MarkResynced(2000, 50000, 1700000005000)
	if e.IsDirty() {
		t.Fatalf("must not be dirty after MarkResynced")
	}
	if e.LastUpdateID() != 2000 {
		t.Fatalf("LastUpdateID: got %d", e.LastUpdateID())
	}
}

func TestEngine_TopLevels_ClampsN(t *testing.T) {
	t.Parallel()
	var e *Engine = NewEngine("BTCUSDT", 200)
	e.ApplySnapshot(Snapshot{
		Symbol: "BTCUSDT",
		Bids: []types.OrderBookLevel{
			lvl("100", "1"), lvl("99", "1"), lvl("98", "1"),
		},
		Asks: []types.OrderBookLevel{
			lvl("101", "1"), lvl("102", "1"),
		},
		UpdateID: 100,
	})
	var bids, asks = e.TopLevels(2)
	if len(bids) != 2 || len(asks) != 2 {
		t.Fatalf("len: bids=%d asks=%d", len(bids), len(asks))
	}
	if !bids[0].Price.Equal(dq("100")) || !bids[1].Price.Equal(dq("99")) {
		t.Fatalf("bids[0..1]: %v", bids)
	}
}

// Concurrency smoke test: a single writer applying deltas while many
// readers query TopLevels / BestBidAsk. The race detector should stay
// silent.
func TestEngine_RaceFree(t *testing.T) {
	t.Parallel()
	var e *Engine = NewEngine("BTCUSDT", 200)
	e.ApplySnapshot(Snapshot{
		Symbol:   "BTCUSDT",
		Bids:     []types.OrderBookLevel{lvl("100", "1")},
		Asks:     []types.OrderBookLevel{lvl("101", "1")},
		UpdateID: 100,
	})

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		var i int
		for i = 0; i < 1000; i++ {
			e.ApplyDelta(Delta{
				UpdateID: int64(101 + i),
				Bids:     []types.OrderBookLevel{lvl("100", decimal.NewFromInt(int64(i)).String())},
			})
		}
	}()
	go func() {
		defer wg.Done()
		var i int
		for i = 0; i < 1000; i++ {
			_, _, _, _ = e.BestBidAsk()
			_, _ = e.TopLevels(5)
		}
	}()
	wg.Wait()
}
