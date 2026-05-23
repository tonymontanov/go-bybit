/*
FILE: linears/stream_test.go

DESCRIPTION:
Mock-WebSocket tests for the linears StreamClient. Each test wires a tiny
fake Bybit V5 WS server (httptest + gorilla/websocket) into a real
linears.Client via Config.WS.PublicLinearURL / PrivateURL, calls the
matching Watch* method and verifies that the user handler receives the
expected, fully-decoded domain struct.

WHY MOCK-WS RATHER THAN UNIT-TEST EVERY PARSER:
The Watch* methods are thin glue between the wire JSON and the
linears/types structs; the cheapest way to gain confidence is to feed
real-shaped frames through the entire stack — ws.Conn, dispatcher,
applyOrderbookFrame, mergeTickerUpdate. The mock server speaks just
enough of the Bybit V5 WS protocol for that:

  - /public/linear : ack subscribe, push topic frames.
  - /private       : ack auth, ack subscribe, push topic frames.

Tests use very short timeouts (HandshakeTimeout=1s, etc.) so a hung mock
fails the test in seconds rather than the package default of 35s.

NOTES:
  - The tests use Config.WS.PingInterval = 0 to disable client-side pings.
  - Reconnect/jitter is disabled the same way (ReconnectInitialBackoff is
    set very small and the mock keeps the connection open for the
    duration of the test).
*/

package linears

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shopspring/decimal"

	bybit "github.com/tonymontanov/go-bybit/v2"
	"github.com/tonymontanov/go-bybit/v2/linears/types"
)

// ---------------------------------------------------------------------
// Mock WS server.
// ---------------------------------------------------------------------

var streamUpgrader = websocket.Upgrader{
	CheckOrigin: func(*http.Request) bool { return true },
}

// fakeWS is a minimal Bybit V5 WS endpoint. The handler closure controls
// the conversation: it can read inbound frames (subscribe / auth / ping)
// and push outbound frames in any order.
type fakeWS struct {
	t   *testing.T
	srv *httptest.Server
	mu  sync.Mutex
	c   *websocket.Conn
}

func newFakeWS(t *testing.T, handler func(s *fakeWS)) *fakeWS {
	t.Helper()
	var s *fakeWS = &fakeWS{t: t}
	s.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var c *websocket.Conn
		var err error
		c, err = streamUpgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("upgrade: %v", err)
			return
		}
		s.mu.Lock()
		s.c = c
		s.mu.Unlock()
		if handler != nil {
			handler(s)
		}
	}))
	t.Cleanup(s.close)
	return s
}

func (s *fakeWS) wsURL() string { return "ws" + strings.TrimPrefix(s.srv.URL, "http") }

func (s *fakeWS) close() {
	s.mu.Lock()
	if s.c != nil {
		_ = s.c.Close()
	}
	s.mu.Unlock()
	s.srv.Close()
}

func (s *fakeWS) writeJSON(v any) error {
	var raw []byte
	var err error
	raw, err = json.Marshal(v)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.c == nil {
		return nil
	}
	return s.c.WriteMessage(websocket.TextMessage, raw)
}

// readUntilOp blocks (with a hard deadline) until the server sees a
// frame with top-level "op"==want. Returns the decoded probe map.
func (s *fakeWS) readUntilOp(want string, timeout time.Duration) (map[string]any, bool) {
	var deadline time.Time = time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		s.mu.Lock()
		var conn *websocket.Conn = s.c
		s.mu.Unlock()
		if conn == nil {
			time.Sleep(5 * time.Millisecond)
			continue
		}
		_ = conn.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
		var typ int
		var raw []byte
		var err error
		typ, raw, err = conn.ReadMessage()
		if err != nil {
			continue
		}
		if typ != websocket.TextMessage {
			continue
		}
		var probe map[string]any
		if json.Unmarshal(raw, &probe) == nil {
			if op, _ := probe["op"].(string); op == want {
				return probe, true
			}
		}
	}
	return nil, false
}

// ---------------------------------------------------------------------
// Client wiring helpers.
// ---------------------------------------------------------------------

// newStreamTestClient builds a *linears.Client whose public/private WS
// URLs point at the supplied fake servers. Either pointer may be nil if
// the test only exercises one side.
func newStreamTestClient(t *testing.T, public, private *fakeWS) *Client {
	t.Helper()
	var cfg bybit.Config = bybit.DefaultConfig()
	cfg.APIKey = "TESTKEY1234567890"
	cfg.SecretKey = "testsecret"
	if public != nil {
		cfg.WS.PublicLinearURL = public.wsURL()
	}
	if private != nil {
		cfg.WS.PrivateURL = private.wsURL()
	}
	cfg.WS.HandshakeTimeout = time.Second
	cfg.WS.ReadTimeout = 3 * time.Second
	cfg.WS.WriteTimeout = time.Second
	cfg.WS.PingInterval = 0
	cfg.WS.AuthExpiresWindow = time.Second
	cfg.WS.AuthTimeout = 2 * time.Second
	cfg.WS.ReconnectInitialBackoff = 5 * time.Millisecond
	cfg.WS.ReconnectMaxBackoff = 50 * time.Millisecond
	cfg.WS.ReconnectJitter = 0

	var bc *bybit.Client
	var err error
	bc, err = bybit.NewClient(cfg)
	if err != nil {
		t.Fatalf("bybit.NewClient: %v", err)
	}
	t.Cleanup(func() { _ = bc.Close() })

	var lc = bc.Linears().(*Client)
	t.Cleanup(func() { _ = lc.Stream().Close() })
	return lc
}

// ---------------------------------------------------------------------
// PUBLIC: orderbook.
// ---------------------------------------------------------------------

func TestStream_WatchOrderBook_SnapshotThenDelta(t *testing.T) {
	var public *fakeWS = newFakeWS(t, func(s *fakeWS) {
		var _, ok = s.readUntilOp("subscribe", 2*time.Second)
		if !ok {
			t.Errorf("did not see subscribe op")
			return
		}
		_ = s.writeJSON(map[string]any{"op": "subscribe", "success": true, "conn_id": "C"})

		_ = s.writeJSON(map[string]any{
			"topic": "orderbook.50.BTCUSDT",
			"type":  "snapshot",
			"ts":    time.Now().UnixMilli(),
			"data": map[string]any{
				"s":   "BTCUSDT",
				"b":   [][]string{{"60000", "1.0"}, {"59999", "0.5"}},
				"a":   [][]string{{"60001", "0.7"}, {"60002", "0.2"}},
				"u":   100,
				"seq": 1,
			},
		})
		_ = s.writeJSON(map[string]any{
			"topic": "orderbook.50.BTCUSDT",
			"type":  "delta",
			"ts":    time.Now().UnixMilli(),
			"data": map[string]any{
				"s":   "BTCUSDT",
				"b":   [][]string{{"60000", "0.8"}},
				"a":   [][]string{{"60001", "0"}},
				"u":   101,
				"seq": 2,
			},
		})
	})

	var lc *Client = newStreamTestClient(t, public, nil)

	var snaps = make(chan types.OrderBookSnapshot, 4)
	var ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	if err := lc.Stream().WatchOrderBook(ctx, "BTCUSDT", 50, 5,
		func(ob types.OrderBookSnapshot) { snaps <- ob },
		func(err error) { t.Errorf("errHandler: %v", err) },
	); err != nil {
		t.Fatalf("WatchOrderBook: %v", err)
	}

	// First push — snapshot fully populated.
	select {
	case ob := <-snaps:
		if ob.Symbol != "BTCUSDT" {
			t.Fatalf("Symbol=%q want BTCUSDT", ob.Symbol)
		}
		if ob.UpdateID != 100 {
			t.Fatalf("snapshot UpdateID=%d want 100", ob.UpdateID)
		}
		if !ob.Bids[0].Price.Equal(decimal.RequireFromString("60000")) {
			t.Fatalf("snapshot top bid price=%s want 60000", ob.Bids[0].Price)
		}
		if !ob.Asks[0].Price.Equal(decimal.RequireFromString("60001")) {
			t.Fatalf("snapshot top ask price=%s want 60001", ob.Asks[0].Price)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("snapshot not delivered")
	}

	// Second push — delta. Top bid size moves to 0.8, top ask removed →
	// next ask is 60002.
	select {
	case ob := <-snaps:
		if ob.UpdateID != 101 {
			t.Fatalf("delta UpdateID=%d want 101", ob.UpdateID)
		}
		if !ob.Bids[0].Size.Equal(decimal.RequireFromString("0.8")) {
			t.Fatalf("delta top bid size=%s want 0.8", ob.Bids[0].Size)
		}
		if !ob.Asks[0].Price.Equal(decimal.RequireFromString("60002")) {
			t.Fatalf("delta top ask price=%s want 60002 (60001 removed)", ob.Asks[0].Price)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("delta not delivered")
	}
}

func TestStream_WatchOrderBook_GapTriggersErrHandler(t *testing.T) {
	var public *fakeWS = newFakeWS(t, func(s *fakeWS) {
		var _, ok = s.readUntilOp("subscribe", 2*time.Second)
		if !ok {
			return
		}
		_ = s.writeJSON(map[string]any{"op": "subscribe", "success": true})

		_ = s.writeJSON(map[string]any{
			"topic": "orderbook.50.ETHUSDT", "type": "snapshot",
			"data": map[string]any{
				"s": "ETHUSDT",
				"b": [][]string{{"3000", "1"}}, "a": [][]string{{"3001", "1"}},
				"u": 50, "seq": 1,
			},
		})
		// Gap: u jumps from 50 to 60 (lastU+1==51 expected).
		_ = s.writeJSON(map[string]any{
			"topic": "orderbook.50.ETHUSDT", "type": "delta",
			"data": map[string]any{
				"s": "ETHUSDT",
				"b": [][]string{{"3000", "2"}}, "a": [][]string{},
				"u": 60, "seq": 2,
			},
		})
	})

	var lc *Client = newStreamTestClient(t, public, nil)

	var errs = make(chan error, 4)
	var ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	if err := lc.Stream().WatchOrderBook(ctx, "ETHUSDT", 50, 5,
		func(types.OrderBookSnapshot) {},
		func(err error) { errs <- err },
	); err != nil {
		t.Fatalf("WatchOrderBook: %v", err)
	}

	select {
	case e := <-errs:
		if e == nil {
			t.Fatalf("nil err")
		}
		if !strings.Contains(e.Error(), "orderbook gap") {
			t.Fatalf("err=%v want orderbook gap", e)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("gap not signalled")
	}
}

// ---------------------------------------------------------------------
// PUBLIC: tickers.
// ---------------------------------------------------------------------

func TestStream_WatchTicker_MergesDeltas(t *testing.T) {
	var public *fakeWS = newFakeWS(t, func(s *fakeWS) {
		var _, ok = s.readUntilOp("subscribe", 2*time.Second)
		if !ok {
			return
		}
		_ = s.writeJSON(map[string]any{"op": "subscribe", "success": true})

		// Snapshot — full payload.
		_ = s.writeJSON(map[string]any{
			"topic": "tickers.BTCUSDT", "type": "snapshot",
			"data": map[string]any{
				"symbol":     "BTCUSDT",
				"lastPrice":  "60000",
				"markPrice":  "59999.5",
				"indexPrice": "60001",
				"bid1Price":  "59995", "bid1Size": "1",
				"ask1Price":  "60005", "ask1Size": "1.2",
				"volume24h": "120",
			},
		})
		// Delta — only lastPrice moves.
		_ = s.writeJSON(map[string]any{
			"topic": "tickers.BTCUSDT", "type": "delta",
			"data": map[string]any{"symbol": "BTCUSDT", "lastPrice": "60100"},
		})
	})

	var lc *Client = newStreamTestClient(t, public, nil)

	var ticks = make(chan types.TickerUpdate, 4)
	var ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	if err := lc.Stream().WatchTicker(ctx, "BTCUSDT",
		func(t types.TickerUpdate) { ticks <- t },
		func(err error) { t.Errorf("errHandler: %v", err) },
	); err != nil {
		t.Fatalf("WatchTicker: %v", err)
	}

	// Snapshot delivery.
	select {
	case tk := <-ticks:
		if !tk.LastPrice.Equal(decimal.RequireFromString("60000")) {
			t.Fatalf("snapshot LastPrice=%s want 60000", tk.LastPrice)
		}
		if !tk.BestBid.Equal(decimal.RequireFromString("59995")) {
			t.Fatalf("snapshot BestBid=%s want 59995", tk.BestBid)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("snapshot not delivered")
	}

	// Delta delivery — LastPrice changes, other fields preserved.
	select {
	case tk := <-ticks:
		if !tk.LastPrice.Equal(decimal.RequireFromString("60100")) {
			t.Fatalf("delta LastPrice=%s want 60100", tk.LastPrice)
		}
		if !tk.BestBid.Equal(decimal.RequireFromString("59995")) {
			t.Fatalf("delta lost BestBid; got %s", tk.BestBid)
		}
		if !tk.MarkPrice.Equal(decimal.RequireFromString("59999.5")) {
			t.Fatalf("delta lost MarkPrice; got %s", tk.MarkPrice)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("delta not delivered")
	}
}

// ---------------------------------------------------------------------
// PUBLIC: publicTrade.
// ---------------------------------------------------------------------

func TestStream_WatchTrades_FansOutBatch(t *testing.T) {
	var public *fakeWS = newFakeWS(t, func(s *fakeWS) {
		var _, ok = s.readUntilOp("subscribe", 2*time.Second)
		if !ok {
			return
		}
		_ = s.writeJSON(map[string]any{"op": "subscribe", "success": true})

		_ = s.writeJSON(map[string]any{
			"topic": "publicTrade.BTCUSDT", "type": "snapshot",
			"data": []map[string]any{
				{"i": "T1", "s": "BTCUSDT", "S": "Buy", "v": "0.1", "p": "60000", "T": int64(1700000000000)},
				{"i": "T2", "s": "BTCUSDT", "S": "Sell", "v": "0.2", "p": "60001", "T": int64(1700000000100)},
			},
		})
	})

	var lc *Client = newStreamTestClient(t, public, nil)

	var trades = make(chan types.TradeUpdate, 8)
	var ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	if err := lc.Stream().WatchTrades(ctx, "BTCUSDT",
		func(t types.TradeUpdate) { trades <- t },
		func(err error) { t.Errorf("errHandler: %v", err) },
	); err != nil {
		t.Fatalf("WatchTrades: %v", err)
	}

	var got []types.TradeUpdate
	var deadline = time.After(2 * time.Second)
loop:
	for len(got) < 2 {
		select {
		case t := <-trades:
			got = append(got, t)
		case <-deadline:
			break loop
		}
	}
	if len(got) != 2 {
		t.Fatalf("got %d trades, want 2", len(got))
	}
	if got[0].TradeID != "T1" || got[1].TradeID != "T2" {
		t.Fatalf("trade order: %s %s", got[0].TradeID, got[1].TradeID)
	}
		if got[0].Side != types.SideTypeBuy || got[1].Side != types.SideTypeSell {
		t.Fatalf("trade sides: %s %s", got[0].Side, got[1].Side)
	}
}

// ---------------------------------------------------------------------
// PUBLIC: kline.
// ---------------------------------------------------------------------

func TestStream_WatchKline_DecodesPayload(t *testing.T) {
	var public *fakeWS = newFakeWS(t, func(s *fakeWS) {
		var _, ok = s.readUntilOp("subscribe", 2*time.Second)
		if !ok {
			return
		}
		_ = s.writeJSON(map[string]any{"op": "subscribe", "success": true})

		_ = s.writeJSON(map[string]any{
			"topic": "kline.1.BTCUSDT", "type": "snapshot",
			"data": []map[string]any{{
				"start":    int64(1700000000000),
				"end":      int64(1700000060000),
				"interval": "1",
				"open":     "60000",
				"close":    "60100",
				"high":     "60150",
				"low":      "59990",
				"volume":   "10",
				"turnover": "600500",
				"confirm":  true,
			}},
		})
	})

	var lc *Client = newStreamTestClient(t, public, nil)

	var klines = make(chan types.KlineUpdate, 2)
	var ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	if err := lc.Stream().WatchKline(ctx, "BTCUSDT", types.Timeframe1m,
		func(k types.KlineUpdate) { klines <- k },
		func(err error) { t.Errorf("errHandler: %v", err) },
	); err != nil {
		t.Fatalf("WatchKline: %v", err)
	}

	select {
	case k := <-klines:
		if !k.Confirmed {
			t.Fatalf("Confirmed=false want true")
		}
		if !k.Close.Equal(decimal.RequireFromString("60100")) {
			t.Fatalf("Close=%s want 60100", k.Close)
		}
		if k.Symbol != "BTCUSDT" {
			t.Fatalf("Symbol=%q", k.Symbol)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("kline not delivered")
	}
}

// ---------------------------------------------------------------------
// PRIVATE: orders / positions / executions / wallet.
// ---------------------------------------------------------------------

// privateAuthAndSubscribe wires the standard auth handshake → subscribe ack
// for the private endpoint and then invokes payload(). Splitting it keeps
// the per-test code focused on the topic-specific pushes.
func privateAuthAndSubscribe(t *testing.T, payload func(s *fakeWS)) *fakeWS {
	t.Helper()
	return newFakeWS(t, func(s *fakeWS) {
		if _, ok := s.readUntilOp("auth", 2*time.Second); !ok {
			t.Errorf("did not see auth op")
			return
		}
		_ = s.writeJSON(map[string]any{"op": "auth", "success": true, "conn_id": "C"})

		if _, ok := s.readUntilOp("subscribe", 2*time.Second); !ok {
			t.Errorf("did not see subscribe op")
			return
		}
		_ = s.writeJSON(map[string]any{"op": "subscribe", "success": true})

		payload(s)
	})
}

func TestStream_WatchOrders_FiltersByCategory(t *testing.T) {
	var private *fakeWS = privateAuthAndSubscribe(t, func(s *fakeWS) {
		_ = s.writeJSON(map[string]any{
			"topic": "order",
			"data": []map[string]any{
				{"category": "spot", "orderId": "S1", "symbol": "BTCUSDT", "orderStatus": "New"},
				{"category": "linear", "orderId": "L1", "symbol": "BTCUSDT", "orderStatus": "New",
					"side": "Buy", "orderType": "Limit", "qty": "0.01", "price": "60000",
					"leavesQty": "0.01", "cumExecQty": "0", "createdTime": "1700000000000",
					"updatedTime": "1700000000100"},
			},
		})
	})

	var lc *Client = newStreamTestClient(t, nil, private)

	var orders = make(chan types.OrderInfo, 4)
	var ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	if err := lc.Stream().WatchOrders(ctx,
		func(o types.OrderInfo) { orders <- o },
		func(err error) { t.Errorf("errHandler: %v", err) },
	); err != nil {
		t.Fatalf("WatchOrders: %v", err)
	}

	select {
	case o := <-orders:
		if o.OrderID != "L1" {
			t.Fatalf("OrderID=%q want L1 (spot row should be filtered)", o.OrderID)
		}
		if o.Side != types.SideTypeBuy {
			t.Fatalf("Side=%s want Buy", o.Side)
		}
		if !o.Price.Equal(decimal.RequireFromString("60000")) {
			t.Fatalf("Price=%s want 60000", o.Price)
		}
		if o.CreatedAtMs != 1700000000000 {
			t.Fatalf("CreatedAtMs=%d", o.CreatedAtMs)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("order not delivered")
	}

	// Make sure we did NOT receive the spot row (drain a tiny grace
	// window).
	select {
	case extra := <-orders:
		t.Fatalf("unexpected extra order: %+v", extra)
	case <-time.After(150 * time.Millisecond):
	}
}

func TestStream_WatchOrders_RequiresCredentials(t *testing.T) {
	var cfg bybit.Config = bybit.DefaultConfig()
	cfg.WS.PrivateURL = "ws://127.0.0.1:1" // never reached
	var bc *bybit.Client
	var err error
	bc, err = bybit.NewClient(cfg)
	if err != nil {
		t.Fatalf("bybit.NewClient: %v", err)
	}
	defer bc.Close()
	var lc = bc.Linears().(*Client)

	var werr = lc.Stream().WatchOrders(context.Background(),
		func(types.OrderInfo) {}, func(error) {},
	)
	if werr == nil {
		t.Fatalf("expected error when APIKey is missing")
	}
	var be *bybit.Error
	if !errors.As(werr, &be) || be.Kind != bybit.ErrorKindAuth {
		t.Fatalf("err=%v want ErrorKindAuth", werr)
	}
}

func TestStream_WatchPositions_DecodesLinearOnly(t *testing.T) {
	var private *fakeWS = privateAuthAndSubscribe(t, func(s *fakeWS) {
		_ = s.writeJSON(map[string]any{
			"topic": "position",
			"data": []map[string]any{
				{"category": "linear", "symbol": "BTCUSDT", "side": "Buy",
					"size": "0.5", "entryPrice": "60000", "markPrice": "60050",
					"liqPrice": "30000", "leverage": "10",
					"unrealisedPnl": "25", "cumRealisedPnl": "100",
					"positionValue": "30025", "updatedTime": "1700000000200",
					"positionIdx": 0},
			},
		})
	})

	var lc *Client = newStreamTestClient(t, nil, private)

	var pos = make(chan types.PositionInfo, 2)
	var ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	if err := lc.Stream().WatchPositions(ctx,
		func(p types.PositionInfo) { pos <- p },
		func(err error) { t.Errorf("errHandler: %v", err) },
	); err != nil {
		t.Fatalf("WatchPositions: %v", err)
	}

	select {
	case p := <-pos:
		if p.Symbol != "BTCUSDT" {
			t.Fatalf("Symbol=%q", p.Symbol)
		}
		if p.Side != types.SideTypeBuy {
			t.Fatalf("Side=%s", p.Side)
		}
		if !p.Quantity.Equal(decimal.RequireFromString("0.5")) {
			t.Fatalf("Quantity=%s", p.Quantity)
		}
		if !p.UnrealizedPnL.Equal(decimal.RequireFromString("25")) {
			t.Fatalf("UnrealizedPnL=%s", p.UnrealizedPnL)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("position not delivered")
	}
}

func TestStream_WatchExecutions_DecodesLinearFill(t *testing.T) {
	var private *fakeWS = privateAuthAndSubscribe(t, func(s *fakeWS) {
		_ = s.writeJSON(map[string]any{
			"topic": "execution",
			"data": []map[string]any{
				{"category": "linear", "symbol": "BTCUSDT",
					"orderId": "L1", "orderLinkId": "CL1", "execId": "E1",
					"side": "Buy", "execPrice": "60000", "execQty": "0.01",
					"execValue": "600", "execFee": "0.36", "feeCurrency": "USDT",
					"isMaker": false, "positionIdx": 0,
					"execTime": "1700000000300"},
			},
		})
	})

	var lc *Client = newStreamTestClient(t, nil, private)

	var fills = make(chan types.ExecutionInfo, 2)
	var ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	if err := lc.Stream().WatchExecutions(ctx,
		func(e types.ExecutionInfo) { fills <- e },
		func(err error) { t.Errorf("errHandler: %v", err) },
	); err != nil {
		t.Fatalf("WatchExecutions: %v", err)
	}

	select {
	case e := <-fills:
		if e.ExecID != "E1" {
			t.Fatalf("ExecID=%q", e.ExecID)
		}
		if !e.ExecFee.Equal(decimal.RequireFromString("0.36")) {
			t.Fatalf("ExecFee=%s", e.ExecFee)
		}
		if e.IsMaker {
			t.Fatalf("IsMaker=true want false")
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("fill not delivered")
	}
}

func TestStream_WatchWallet_ReusesRESTConverter(t *testing.T) {
	var private *fakeWS = privateAuthAndSubscribe(t, func(s *fakeWS) {
		_ = s.writeJSON(map[string]any{
			"topic": "wallet",
			"data": []map[string]any{{
				"accountType":           "UNIFIED",
				"totalEquity":           "1000",
				"totalAvailableBalance": "950",
				"totalMarginBalance":    "1000",
				"totalInitialMargin":    "50",
				"totalMaintenanceMargin": "20",
				"coin": []map[string]any{
					{"coin": "USDT", "equity": "1000", "walletBalance": "1000",
						"availableToWithdraw": "950", "free": "950",
						"locked": "50", "unrealisedPnl": "0", "cumRealisedPnl": "0"},
				},
			}},
		})
	})

	var lc *Client = newStreamTestClient(t, nil, private)

	var balances = make(chan types.Balance, 2)
	var ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	if err := lc.Stream().WatchWallet(ctx,
		func(b types.Balance) { balances <- b },
		func(err error) { t.Errorf("errHandler: %v", err) },
	); err != nil {
		t.Fatalf("WatchWallet: %v", err)
	}

	select {
	case b := <-balances:
		if b.AccountType != "UNIFIED" {
			t.Fatalf("AccountType=%q", b.AccountType)
		}
		if !b.TotalEquity.Equal(decimal.RequireFromString("1000")) {
			t.Fatalf("TotalEquity=%s", b.TotalEquity)
		}
		if len(b.Coins) != 1 || b.Coins[0].Coin != "USDT" {
			t.Fatalf("Coins=%+v", b.Coins)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("wallet not delivered")
	}
}
