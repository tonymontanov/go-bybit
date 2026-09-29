/*
FILE: spot/stream-ts_test.go

DESCRIPTION:
TsMs of WS tickers and orderbook snapshots comes from the push envelope
"ts" (the data of these topics carries no timestamp of its own); a frame
without "ts" gets the local receive time, never zero. Mirrors
linears/stream-ts_test.go through a minimal fake Bybit public WS endpoint.
*/

package spot

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	bybit "github.com/tonymontanov/go-bybit/v2"
	bybitspottypes "github.com/tonymontanov/go-bybit/v2/spot/types"
)

// spotFakeWS — minimal public Bybit V5 WS endpoint: waits for the
// subscribe op, acks it and writes the given frames.
type spotFakeWS struct {
	srv *httptest.Server
	mu  sync.Mutex
	c   *websocket.Conn
}

func newSpotFakeWS(t *testing.T, frames []map[string]any) *spotFakeWS {
	t.Helper()
	var upgrader = websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	var s *spotFakeWS = &spotFakeWS{}
	s.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var c, err = upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("upgrade: %v", err)
			return
		}
		s.mu.Lock()
		s.c = c
		s.mu.Unlock()
		if !waitSubscribe(c, 2*time.Second) {
			t.Errorf("did not see subscribe op")
			return
		}
		var out = append([]map[string]any{{"op": "subscribe", "success": true}}, frames...)
		for _, f := range out {
			var raw, _ = json.Marshal(f)
			s.mu.Lock()
			_ = c.WriteMessage(websocket.TextMessage, raw)
			s.mu.Unlock()
		}
	}))
	t.Cleanup(func() {
		s.mu.Lock()
		if s.c != nil {
			_ = s.c.Close()
		}
		s.mu.Unlock()
		s.srv.Close()
	})
	return s
}

// waitSubscribe reads inbound frames until {"op":"subscribe"} or timeout.
func waitSubscribe(c *websocket.Conn, timeout time.Duration) bool {
	var deadline time.Time = time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		_ = c.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
		var _, raw, err = c.ReadMessage()
		if err != nil {
			continue
		}
		var probe map[string]any
		if json.Unmarshal(raw, &probe) == nil && probe["op"] == "subscribe" {
			return true
		}
	}
	return false
}

func (s *spotFakeWS) wsURL() string { return "ws" + strings.TrimPrefix(s.srv.URL, "http") }

// newSpotStreamClient builds a spot client whose public WS points at fake.
func newSpotStreamClient(t *testing.T, fake *spotFakeWS) *Client {
	t.Helper()
	var cfg bybit.Config = bybit.DefaultConfig()
	cfg.WS.PublicSpotURL = fake.wsURL()
	cfg.WS.HandshakeTimeout = time.Second
	cfg.WS.ReadTimeout = 3 * time.Second
	cfg.WS.WriteTimeout = time.Second
	cfg.WS.PingInterval = 0
	cfg.WS.ReconnectInitialBackoff = 5 * time.Millisecond
	cfg.WS.ReconnectMaxBackoff = 50 * time.Millisecond
	cfg.WS.ReconnectJitter = 0

	var bc, err = bybit.NewClient(cfg)
	if err != nil {
		t.Fatalf("bybit.NewClient: %v", err)
	}
	t.Cleanup(func() { _ = bc.Close() })

	var sc = bc.Spot().(*Client)
	t.Cleanup(func() { _ = sc.Stream().Close() })
	return sc
}

func TestStream_WatchTicker_TsMsFromEnvelope(t *testing.T) {
	const publishTs int64 = 1700000000123

	var fake *spotFakeWS = newSpotFakeWS(t, []map[string]any{
		{
			"topic": "tickers.BTCUSDT", "type": "snapshot", "ts": publishTs,
			"data": map[string]any{"symbol": "BTCUSDT", "lastPrice": "60000"},
		},
		// No "ts" in the envelope — receive time is expected.
		{
			"topic": "tickers.BTCUSDT", "type": "snapshot",
			"data": map[string]any{"symbol": "BTCUSDT", "lastPrice": "60100"},
		},
	})
	var sc *Client = newSpotStreamClient(t, fake)

	var ticks = make(chan bybitspottypes.TickerUpdate, 4)
	var ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	var before int64 = time.Now().UnixMilli()
	if err := sc.Stream().WatchTicker(ctx, "BTCUSDT",
		func(tk bybitspottypes.TickerUpdate) { ticks <- tk },
		func(err error) { t.Errorf("errHandler: %v", err) },
	); err != nil {
		t.Fatalf("WatchTicker: %v", err)
	}

	select {
	case tk := <-ticks:
		if tk.TsMs != publishTs {
			t.Fatalf("frame with ts: TsMs=%d want %d", tk.TsMs, publishTs)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("first ticker not delivered")
	}

	select {
	case tk := <-ticks:
		var after int64 = time.Now().UnixMilli()
		if tk.TsMs < before || tk.TsMs > after {
			t.Fatalf("frame without ts: TsMs=%d want receive time in [%d, %d]", tk.TsMs, before, after)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("second ticker not delivered")
	}
}

func TestStream_WatchOrderBook_TsMsFromEnvelope(t *testing.T) {
	const publishTs int64 = 1700000000456

	var fake *spotFakeWS = newSpotFakeWS(t, []map[string]any{
		{
			"topic": "orderbook.1.BTCUSDT", "type": "snapshot", "ts": publishTs,
			"data": map[string]any{
				"s": "BTCUSDT", "u": 100, "seq": 1,
				"b": [][]string{{"60000", "1.0"}},
				"a": [][]string{{"60001", "0.7"}},
			},
		},
	})
	var sc *Client = newSpotStreamClient(t, fake)

	var snaps = make(chan bybitspottypes.OrderBookSnapshot, 2)
	var ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	if err := sc.Stream().WatchOrderBook(ctx, "BTCUSDT", 1, 1,
		func(ob bybitspottypes.OrderBookSnapshot) { snaps <- ob },
		func(err error) { t.Errorf("errHandler: %v", err) },
	); err != nil {
		t.Fatalf("WatchOrderBook: %v", err)
	}

	select {
	case ob := <-snaps:
		if ob.TsMs != publishTs {
			t.Fatalf("OrderBookSnapshot.TsMs=%d want %d", ob.TsMs, publishTs)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("snapshot not delivered")
	}
}
