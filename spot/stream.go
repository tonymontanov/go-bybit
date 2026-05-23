/*
FILE: spot/stream.go

DESCRIPTION:
StreamClient — domain WebSocket subscription sub-client for the Bybit
V5 spot category. Public Watch* methods live here; private Watch*
methods live in stream-private.go.

GENERAL PATTERN FOR EACH WATCH*:
  1. Validate the input arguments (symbol non-empty, depth allowed, ...).
  2. Lazily start the spot public ws.Conn under ctx.
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

STREAM TOPIC NAMES (spot):
  - orderbook.{depth}.{symbol}    — depth = 1, 50, 200.
  - publicTrade.{symbol}          — taker trades.
  - tickers.{symbol}              — top-of-book + 24h stats.
  - kline.{interval}.{symbol}     — kline events.
  - order                         — own orders (private, UTA only).
  - execution                     — own fills (private, UTA only).
  - wallet                        — own wallet state (private).
*/

package spot

import (
	"context"
	"strconv"

	bybit "github.com/tonymontanov/go-bybit"
	"github.com/tonymontanov/go-bybit/internal/codec"
	"github.com/tonymontanov/go-bybit/internal/ws"
	"github.com/tonymontanov/go-bybit/orderbook"
	bybitspottypes "github.com/tonymontanov/go-bybit/spot/types"
)

// StreamClient — WebSocket subscription sub-client for the spot profile.
type StreamClient struct {
	c *Client
}

func newStreamClient(c *Client) *StreamClient {
	return &StreamClient{c: c}
}

// Close gracefully tears down the public and private WS connections
// owned by this StreamClient. Safe to call when neither connection has
// been opened.
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

// WatchOrderBook subscribes to orderbook.{depth}.{symbol} and maintains
// a local orderbook.Engine. After every successful apply the handler is
// called with the top-`displayDepth` levels and the engine's
// LastUpdateID as snapshot.UpdateID. depth is the wire-level depth
// (clamped to 1/50/200 for spot); displayDepth controls how many levels
// the SDK forwards (≤0 → all available).
//
// On a service-restart or sequence gap the engine is marked dirty,
// errHandler is called once, and further deltas are ignored until the
// next snapshot push (Bybit ships one automatically on a fresh
// subscription / reconnect).
func (s *StreamClient) WatchOrderBook(
	ctx context.Context,
	symbol string,
	depth int,
	displayDepth int,
	handler func(bybitspottypes.OrderBookSnapshot),
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
// into the engine. Snapshot/delta routing matches the linears profile.
func (s *StreamClient) applyOrderbookFrame(
	eng *orderbook.Engine,
	pushType string,
	payload []byte,
	handler func(bybitspottypes.OrderBookSnapshot),
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
	handler(bybitspottypes.OrderBookSnapshot{
		Symbol:   eng.Symbol(),
		Bids:     toSpotLevels(topBids),
		Asks:     toSpotLevels(topAsks),
		UpdateID: eng.LastUpdateID(),
		SeqID:    eng.LastSeqID(),
	})
}

// parseLevelsForEngine turns Bybit's wire-level rows into the orderbook
// engine's level type. The engine's level type is shared via the public
// `orderbook` package, which uses its own struct (not spot/types.Level).
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

// toSpotLevels converts orderbook.Level slices into the spot-typed
// OrderBookLevel slice the user handler expects.
func toSpotLevels(in []orderbook.Level) []bybitspottypes.OrderBookLevel {
	if len(in) == 0 {
		return nil
	}
	var out []bybitspottypes.OrderBookLevel = make([]bybitspottypes.OrderBookLevel, len(in))
	var i int
	for i = 0; i < len(in); i++ {
		out[i] = bybitspottypes.OrderBookLevel{Price: in[i].Price, Size: in[i].Size}
	}
	return out
}

// =====================================================================
// PUBLIC: tickers.
// =====================================================================

// rawTickerPush — fields exposed from tickers.{symbol} for spot. Spot
// has no funding / open-interest / mark-price, so derivative-only
// fields are absent. usdIndexPrice is spot-only (USD reference for
// non-USDT quote pairs).
type rawTickerPush struct {
	Symbol        string `json:"symbol"`
	LastPrice     string `json:"lastPrice"`
	Bid1Price     string `json:"bid1Price"`
	Bid1Size      string `json:"bid1Size"`
	Ask1Price     string `json:"ask1Price"`
	Ask1Size      string `json:"ask1Size"`
	PrevPrice24h  string `json:"prevPrice24h"`
	HighPrice24h  string `json:"highPrice24h"`
	LowPrice24h   string `json:"lowPrice24h"`
	Volume24h     string `json:"volume24h"`
	Turnover24h   string `json:"turnover24h"`
	UsdIndexPrice string `json:"usdIndexPrice"`
}

// WatchTicker subscribes to tickers.{symbol}.
func (s *StreamClient) WatchTicker(
	ctx context.Context,
	symbol string,
	handler func(bybitspottypes.TickerUpdate),
	errHandler func(error),
) error {
	if symbol == "" {
		return bybit.NewError(bybit.ErrorKindInvalidRequest, "", "stream.WatchTicker: symbol is empty", nil)
	}
	var topic string = "tickers." + symbol
	var merged bybitspottypes.TickerUpdate = bybitspottypes.TickerUpdate{Symbol: symbol}

	var sub *ws.Subscription = &ws.Subscription{
		Topic: topic,
		Reset: func() {
			merged = bybitspottypes.TickerUpdate{Symbol: symbol}
		},
		Handler: func(_, pushType string, payload []byte) {
			var push rawTickerPush
			if err := codec.Unmarshal(payload, &push); err != nil {
				s.c.logger().Warn("stream.WatchTicker: parse", bybit.Str("symbol", symbol), bybit.Err(err))
				return
			}
			if pushType == "snapshot" {
				merged = bybitspottypes.TickerUpdate{Symbol: symbol}
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
// previous snapshot.
func mergeTickerUpdate(dst *bybitspottypes.TickerUpdate, src rawTickerPush) {
	if src.LastPrice != "" {
		dst.LastPrice = dec(src.LastPrice)
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
	if src.UsdIndexPrice != "" {
		dst.UsdIndexPrice = dec(src.UsdIndexPrice)
	}
}

// =====================================================================
// PUBLIC: trades.
// =====================================================================

// rawTradePush — one element of publicTrade.{symbol}. Spot uses the
// same wire schema as linears.
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

// WatchTrades subscribes to publicTrade.{symbol}.
func (s *StreamClient) WatchTrades(
	ctx context.Context,
	symbol string,
	handler func(bybitspottypes.TradeUpdate),
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
				handler(bybitspottypes.TradeUpdate{
					Symbol:     symbol,
					Price:      dec(trades[i].Price),
					Size:       dec(trades[i].Volume),
					Side:       bybitspottypes.SideType(trades[i].Side),
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
	tf bybitspottypes.Timeframe,
	handler func(bybitspottypes.KlineUpdate),
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
				handler(bybitspottypes.KlineUpdate{
					Symbol:    symbol,
					Interval:  bybitspottypes.Timeframe(klines[i].Interval),
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
