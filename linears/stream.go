/*
FILE: linears/stream.go

DESCRIPTION:
StreamClient — domain WebSocket subscription sub-client for the Bybit V5
linear category. Public Watch* methods live in this file; private Watch*
methods live in stream-private.go.

GENERAL PATTERN FOR EACH WATCH*:
  1. Validate the input arguments (symbol non-empty, depth allowed, ...).
  2. Lazily start the corresponding ws.Conn under ctx (publicConn /
     privateConn on the parent linears.Client).
  3. Build a ws.Subscription with Topic + Handler + (optional) Reset.
  4. Subscribe; the connection performs (re)connect / auth / resubscribe
     transparently.

ERROR HANDLING:
  - Local parse errors are logged and the bad frame is dropped — calling
    errHandler for every malformed push would be noisy and is rarely
    actionable.
  - Critical errors (Subscribe failure, auth-required-but-missing) call
    errHandler synchronously and return *bberr.Error from Watch*.
  - Reconnect / resubscribe is invisible to the caller; the orderbook
    Watch* uses Subscription.Reset to drop local engine state ahead of
    the next snapshot.

CANCELLATION:
  - When ctx is cancelled the supervisor terminates and the socket
    closes. Watch* does NOT call Unsubscribe — there is no point on a
    closed socket.

STREAM TOPIC NAMES (linear):
  - orderbook.{depth}.{symbol}    — depth = 1, 50, 200, 500.
  - publicTrade.{symbol}          — taker trades.
  - tickers.{symbol}              — top-of-book + 24h stats + funding.
  - kline.{interval}.{symbol}     — kline events.
  - order                         — own orders (private).
  - position                      — own positions (private).
  - execution                     — own fills (private).
  - wallet                        — own wallet state (private).
*/

package linears

import (
	"context"
	"strconv"

	bybit "github.com/tonymontanov/go-bybit/v2"
	"github.com/tonymontanov/go-bybit/v2/internal/codec"
	"github.com/tonymontanov/go-bybit/v2/internal/ws"
	"github.com/tonymontanov/go-bybit/v2/linears/types"
	"github.com/tonymontanov/go-bybit/v2/orderbook"
)

// StreamClient — WebSocket subscription sub-client for the linear profile.
type StreamClient struct {
	c *Client
}

func newStreamClient(c *Client) *StreamClient {
	return &StreamClient{c: c}
}

// Close gracefully tears down the public and private WS connections owned
// by this StreamClient. It is safe to call Close even if the WS were
// never opened.
func (s *StreamClient) Close() error {
	if s.c.publicWs != nil {
		_ = s.c.publicWs.Close()
	}
	if s.c.privateWs != nil {
		_ = s.c.privateWs.Close()
	}
	return nil
}

// =====================================================================
// PUBLIC: orderbook (with local engine).
// =====================================================================

// rawOrderbookPush — payload of one orderbook.{depth}.{symbol} push.
type rawOrderbookPush struct {
	Symbol string     `json:"s"`
	B      [][]string `json:"b"`
	A      [][]string `json:"a"`
	U      int64      `json:"u"`
	Seq    int64      `json:"seq"`
}

// WatchOrderBook subscribes to orderbook.{depth}.{symbol} and maintains a
// local orderbook.Engine. After every successful apply the handler is
// called with the top-`displayDepth` levels and the engine's
// LastUpdateID as snapshot.UpdateID. depth is the wire-level depth
// (clamped to 1/50/200/500); displayDepth controls how many levels the
// SDK forwards to the caller (≤0 → all available).
//
// On a service-restart or a sequence gap the engine is marked dirty,
// errHandler is called once with the *bberr.Error describing the gap
// kind, and further deltas are ignored until the next snapshot push
// (which Bybit ships automatically on a fresh subscription / new
// connection).
func (s *StreamClient) WatchOrderBook(
	ctx context.Context,
	symbol string,
	depth int,
	displayDepth int,
	handler func(types.OrderBookSnapshot),
	errHandler func(error),
) error {
	if symbol == "" {
		return bybit.NewError(bybit.ErrorKindInvalidRequest, "", "stream.WatchOrderBook: symbol is empty", nil)
	}
	depth = clampOrderbookDepth(depth)

	var topic string = "orderbook." + strconv.Itoa(depth) + "." + symbol
	var maxLocal int = s.c.config().Orderbook.MaxDepth
	var eng *orderbook.Engine = orderbook.NewEngine(symbol, maxLocal)

	var sub *ws.Subscription = &ws.Subscription{
		Topic: topic,
		Reset: func() { eng.MarkResynced(0, 0, 0) },
		Handler: func(_, pushType string, payload []byte) {
			// pushType is the envelope's "type" field — "snapshot" /
			// "delta" / "" (rare). The orderbook routing branches on it
			// directly; helpers below absorb the parse-and-apply work.
			s.applyOrderbookFrame(eng, pushType, payload, handler, errHandler, displayDepth)
		},
	}
	s.c.publicConn().Start(ctx)
	if err := s.c.publicConn().Subscribe(sub); err != nil {
		if errHandler != nil {
			errHandler(err)
		}
		return err
	}
	return nil
}

// applyOrderbookFrame decodes one Bybit V5 orderbook push and routes it
// into the engine. The pushType argument is the envelope's "type" field
// — "snapshot" picks ApplySnapshot, "delta" picks ApplyDelta. Anything
// else (including the empty string) is treated as a snapshot, which is
// the safe fallback: snapshots always reset the local state cleanly.
func (s *StreamClient) applyOrderbookFrame(
	eng *orderbook.Engine,
	pushType string,
	payload []byte,
	handler func(types.OrderBookSnapshot),
	errHandler func(error),
	displayDepth int,
) {
	var push rawOrderbookPush
	if err := codec.Unmarshal(payload, &push); err != nil {
		s.c.logger().Warn("stream.WatchOrderBook: parse data", bybit.Str("symbol", eng.Symbol()), bybit.Err(err))
		return
	}

	var bids = parseLevelsForEngine(push.B)
	var asks = parseLevelsForEngine(push.A)

	if pushType == "delta" {
		var res = eng.ApplyDelta(orderbook.Delta{
			Symbol:   eng.Symbol(),
			Bids:     bids,
			Asks:     asks,
			UpdateID: push.U,
			SeqID:    push.Seq,
		})
		if res.Gap != orderbook.GapNone {
			if errHandler != nil {
				errHandler(bybit.NewError(bybit.ErrorKindInvalidRequest, "", "stream.WatchOrderBook: orderbook gap "+res.Gap.String(), nil))
			}
			return
		}
	} else {
		// "snapshot" or unknown — replace local state.
		eng.ApplySnapshot(orderbook.Snapshot{
			Symbol:   eng.Symbol(),
			Bids:     bids,
			Asks:     asks,
			UpdateID: push.U,
			SeqID:    push.Seq,
		})
	}
	if eng.IsDirty() {
		return
	}
	var topBids, topAsks = eng.TopLevels(displayDepth)
	handler(types.OrderBookSnapshot{
		Symbol:   eng.Symbol(),
		Bids:     engineLevelsToTypes(topBids),
		Asks:     engineLevelsToTypes(topAsks),
		UpdateID: eng.LastUpdateID(),
		SeqID:    eng.LastSeqID(),
	})
}

// parseLevelsForEngine converts Bybit's wire-level rows into the
// orderbook engine's level type. The engine type is independent of any
// profile package, so we convert at the boundary; the linears handler
// later flips engine levels back into types.OrderBookLevel via
// engineLevelsToTypes.
func parseLevelsForEngine(rows [][]string) []orderbook.Level {
	if len(rows) == 0 {
		return nil
	}
	var out []orderbook.Level = make([]orderbook.Level, 0, len(rows))
	var i int
	for i = 0; i < len(rows); i++ {
		if len(rows[i]) < 2 {
			continue
		}
		out = append(out, orderbook.Level{
			Price: dec(rows[i][0]),
			Size:  dec(rows[i][1]),
		})
	}
	return out
}

// engineLevelsToTypes converts engine-side levels to the public linears
// OrderBookLevel slice the handler expects.
func engineLevelsToTypes(in []orderbook.Level) []types.OrderBookLevel {
	if len(in) == 0 {
		return nil
	}
	var out []types.OrderBookLevel = make([]types.OrderBookLevel, len(in))
	var i int
	for i = 0; i < len(in); i++ {
		out[i] = types.OrderBookLevel{Price: in[i].Price, Size: in[i].Size}
	}
	return out
}

// =====================================================================
// PUBLIC: tickers.
// =====================================================================

// rawTickerPush — fields the SDK exposes from tickers.{symbol}. Every
// numeric field is a wire-string; absent fields stay empty (Bybit only
// ships changed values in deltas).
type rawTickerPush struct {
	Symbol            string `json:"symbol"`
	LastPrice         string `json:"lastPrice"`
	IndexPrice        string `json:"indexPrice"`
	MarkPrice         string `json:"markPrice"`
	Bid1Price         string `json:"bid1Price"`
	Bid1Size          string `json:"bid1Size"`
	Ask1Price         string `json:"ask1Price"`
	Ask1Size          string `json:"ask1Size"`
	PrevPrice24h      string `json:"prevPrice24h"`
	HighPrice24h      string `json:"highPrice24h"`
	LowPrice24h       string `json:"lowPrice24h"`
	Volume24h         string `json:"volume24h"`
	Turnover24h       string `json:"turnover24h"`
	FundingRate       string `json:"fundingRate"`
	NextFundingTime   string `json:"nextFundingTime"`
	OpenInterest      string `json:"openInterest"`
	OpenInterestValue string `json:"openInterestValue"`
}

// WatchTicker subscribes to tickers.{symbol} and merges Bybit's deltas
// into a running TickerUpdate snapshot before invoking the handler.
func (s *StreamClient) WatchTicker(
	ctx context.Context,
	symbol string,
	handler func(types.TickerUpdate),
	errHandler func(error),
) error {
	if symbol == "" {
		return bybit.NewError(bybit.ErrorKindInvalidRequest, "", "stream.WatchTicker: symbol is empty", nil)
	}
	var topic string = "tickers." + symbol
	var merged types.TickerUpdate = types.TickerUpdate{Symbol: symbol}

	var sub *ws.Subscription = &ws.Subscription{
		Topic: topic,
		Reset: func() {
			merged = types.TickerUpdate{Symbol: symbol}
		},
		Handler: func(_, pushType string, payload []byte) {
			var push rawTickerPush
			if err := codec.Unmarshal(payload, &push); err != nil {
				s.c.logger().Warn("stream.WatchTicker: parse", bybit.Str("symbol", symbol), bybit.Err(err))
				return
			}
			// "snapshot" replaces, "delta" merges. Bybit sends a single
			// snapshot at subscribe time and deltas after that.
			if pushType == "snapshot" {
				merged = types.TickerUpdate{Symbol: symbol}
			}
			mergeTickerUpdate(&merged, push)
			handler(merged)
		},
	}
	s.c.publicConn().Start(ctx)
	if err := s.c.publicConn().Subscribe(sub); err != nil {
		if errHandler != nil {
			errHandler(err)
		}
		return err
	}
	return nil
}

// mergeTickerUpdate merges the changed fields from a delta into the
// previous snapshot. Empty wire fields are treated as "unchanged" — Bybit
// omits unchanged fields from delta pushes.
func mergeTickerUpdate(dst *types.TickerUpdate, src rawTickerPush) {
	if src.LastPrice != "" {
		dst.LastPrice = dec(src.LastPrice)
	}
	if src.IndexPrice != "" {
		dst.IndexPrice = dec(src.IndexPrice)
	}
	if src.MarkPrice != "" {
		dst.MarkPrice = dec(src.MarkPrice)
	}
	if src.Bid1Price != "" {
		dst.BestBid = dec(src.Bid1Price)
	}
	if src.Bid1Size != "" {
		dst.BestBidSize = dec(src.Bid1Size)
	}
	if src.Ask1Price != "" {
		dst.BestAsk = dec(src.Ask1Price)
	}
	if src.Ask1Size != "" {
		dst.BestAskSize = dec(src.Ask1Size)
	}
	if src.PrevPrice24h != "" {
		dst.PrevPrice24h = dec(src.PrevPrice24h)
	}
	if src.HighPrice24h != "" {
		dst.HighPrice24h = dec(src.HighPrice24h)
	}
	if src.LowPrice24h != "" {
		dst.LowPrice24h = dec(src.LowPrice24h)
	}
	if src.Volume24h != "" {
		dst.Volume24h = dec(src.Volume24h)
	}
	if src.Turnover24h != "" {
		dst.Turnover24h = dec(src.Turnover24h)
	}
	if src.FundingRate != "" {
		dst.FundingRate = dec(src.FundingRate)
	}
	if src.NextFundingTime != "" {
		dst.NextFundingTimeMs = ms(src.NextFundingTime)
	}
	if src.OpenInterest != "" {
		dst.OpenInterest = dec(src.OpenInterest)
	}
	if src.OpenInterestValue != "" {
		dst.OpenInterestValue = dec(src.OpenInterestValue)
	}
}

// =====================================================================
// PUBLIC: trades.
// =====================================================================

// rawTradePush — one element of publicTrade.{symbol}. Bybit ships an
// array of these per push.
type rawTradePush struct {
	TradeID    string `json:"i"`
	Symbol     string `json:"s"`
	Side       string `json:"S"`
	Volume     string `json:"v"`
	Price      string `json:"p"`
	Tick       string `json:"L"`
	BlockTrade bool   `json:"BT"`
	Ts         int64  `json:"T"`
}

// WatchTrades subscribes to publicTrade.{symbol} and fans the resulting
// array out — the handler is called once per trade.
func (s *StreamClient) WatchTrades(
	ctx context.Context,
	symbol string,
	handler func(types.TradeUpdate),
	errHandler func(error),
) error {
	if symbol == "" {
		return bybit.NewError(bybit.ErrorKindInvalidRequest, "", "stream.WatchTrades: symbol is empty", nil)
	}
	var topic string = "publicTrade." + symbol

	var sub *ws.Subscription = &ws.Subscription{
		Topic: topic,
		Handler: func(_, _ string, payload []byte) {
			var trades []rawTradePush
			if err := codec.Unmarshal(payload, &trades); err != nil {
				s.c.logger().Warn("stream.WatchTrades: parse", bybit.Str("symbol", symbol), bybit.Err(err))
				return
			}
			var i int
			for i = 0; i < len(trades); i++ {
				handler(types.TradeUpdate{
					Symbol:     symbol,
					Price:      dec(trades[i].Price),
					Size:       dec(trades[i].Volume),
					Side:       types.SideType(trades[i].Side),
					TradeID:    trades[i].TradeID,
					TsMs:       trades[i].Ts,
					BlockTrade: trades[i].BlockTrade,
				})
			}
		},
	}
	s.c.publicConn().Start(ctx)
	if err := s.c.publicConn().Subscribe(sub); err != nil {
		if errHandler != nil {
			errHandler(err)
		}
		return err
	}
	return nil
}

// =====================================================================
// PUBLIC: kline.
// =====================================================================

// rawKlinePush — one element of kline.{interval}.{symbol}.
type rawKlinePush struct {
	Start    int64  `json:"start"`
	End      int64  `json:"end"`
	Interval string `json:"interval"`
	Open     string `json:"open"`
	Close    string `json:"close"`
	High     string `json:"high"`
	Low      string `json:"low"`
	Volume   string `json:"volume"`
	Turnover string `json:"turnover"`
	Confirm  bool   `json:"confirm"`
}

// WatchKline subscribes to kline.{interval}.{symbol}.
func (s *StreamClient) WatchKline(
	ctx context.Context,
	symbol string,
	tf types.Timeframe,
	handler func(types.KlineUpdate),
	errHandler func(error),
) error {
	if symbol == "" {
		return bybit.NewError(bybit.ErrorKindInvalidRequest, "", "stream.WatchKline: symbol is empty", nil)
	}
	if tf == "" {
		return bybit.NewError(bybit.ErrorKindInvalidRequest, "", "stream.WatchKline: timeframe is empty", nil)
	}
	var topic string = "kline." + tf.Wire() + "." + symbol

	var sub *ws.Subscription = &ws.Subscription{
		Topic: topic,
		Handler: func(_, _ string, payload []byte) {
			var klines []rawKlinePush
			if err := codec.Unmarshal(payload, &klines); err != nil {
				s.c.logger().Warn("stream.WatchKline: parse", bybit.Str("symbol", symbol), bybit.Err(err))
				return
			}
			var i int
			for i = 0; i < len(klines); i++ {
				handler(types.KlineUpdate{
					Symbol:    symbol,
					Interval:  types.Timeframe(klines[i].Interval),
					StartMs:   klines[i].Start,
					EndMs:     klines[i].End,
					Open:      dec(klines[i].Open),
					High:      dec(klines[i].High),
					Low:       dec(klines[i].Low),
					Close:     dec(klines[i].Close),
					Volume:    dec(klines[i].Volume),
					Turnover:  dec(klines[i].Turnover),
					Confirmed: klines[i].Confirm,
				})
			}
		},
	}
	s.c.publicConn().Start(ctx)
	if err := s.c.publicConn().Subscribe(sub); err != nil {
		if errHandler != nil {
			errHandler(err)
		}
		return err
	}
	return nil
}
