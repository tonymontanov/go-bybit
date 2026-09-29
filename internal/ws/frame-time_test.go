/*
FILE: internal/ws/frame-time_test.go

DESCRIPTION:
Subscription.TimedHandler receives the frame time: the envelope "ts" when
Bybit sends it, the local receive time otherwise — never zero. Handler-only
subscriptions keep working unchanged, and a subscription with neither
handler is rejected.
*/

package ws

import (
	"context"
	"testing"
	"time"

	"github.com/tonymontanov/go-bybit/v2/internal/auth"
)

func TestConn_TimedHandlerGetsFrameTime(t *testing.T) {
	const publishTs int64 = 1700000000123

	var srv *fakeBybitServer = newFakeServer(t, false, func(s *fakeBybitServer) {
		var _, ok = s.readUntilOp("subscribe", 2*time.Second)
		if !ok {
			t.Errorf("did not see subscribe op")
			return
		}
		_ = s.writeJSON(map[string]any{"op": "subscribe", "success": true, "ret_msg": "", "conn_id": "C1"})
		// Frame with the envelope "ts".
		_ = s.writeJSON(map[string]any{
			"topic": "tickers.BTCUSDT",
			"type":  "snapshot",
			"ts":    publishTs,
			"data":  map[string]any{"symbol": "BTCUSDT", "lastPrice": "60000"},
		})
		// Frame without "ts" — the receive time must be used instead.
		_ = s.writeJSON(map[string]any{
			"topic": "tickers.BTCUSDT",
			"type":  "delta",
			"data":  map[string]any{"symbol": "BTCUSDT", "lastPrice": "60100"},
		})
	})
	defer srv.close()

	var c *Conn = NewConn(Config{
		URL:                     srv.wsURL(),
		HandshakeTimeout:        time.Second,
		ReadTimeout:             3 * time.Second,
		WriteTimeout:            time.Second,
		PingInterval:            0,
		ReconnectInitialBackoff: 5 * time.Millisecond,
		ReconnectMaxBackoff:     50 * time.Millisecond,
	}, auth.NewSigner("", ""), nil, nil)
	defer func() { _ = c.Close() }()

	var ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	var before int64 = time.Now().UnixMilli()
	c.Start(ctx)

	var got = make(chan int64, 2)
	var err error = c.Subscribe(&Subscription{
		Topic: "tickers.BTCUSDT",
		TimedHandler: func(_, _ string, tsMs int64, _ []byte) {
			got <- tsMs
		},
	})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	select {
	case ts := <-got:
		if ts != publishTs {
			t.Fatalf("frame with ts: tsMs=%d want %d", ts, publishTs)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("first push not delivered")
	}

	select {
	case ts := <-got:
		var after int64 = time.Now().UnixMilli()
		if ts < before || ts > after {
			t.Fatalf("frame without ts: tsMs=%d want receive time in [%d, %d]", ts, before, after)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("second push not delivered")
	}
}

func TestConn_SubscribeRequiresAHandler(t *testing.T) {
	var c *Conn = NewConn(Config{URL: "ws://127.0.0.1:1"}, auth.NewSigner("", ""), nil, nil)
	defer func() { _ = c.Close() }()

	if err := c.Subscribe(&Subscription{Topic: "tickers.BTCUSDT"}); err == nil {
		t.Fatalf("subscription without Handler and TimedHandler must be rejected")
	}
	if err := c.Subscribe(&Subscription{
		Topic:        "tickers.BTCUSDT",
		TimedHandler: func(_, _ string, _ int64, _ []byte) {},
	}); err != nil {
		t.Fatalf("TimedHandler-only subscription rejected: %v", err)
	}
	if err := c.Subscribe(&Subscription{
		Topic:   "tickers.ETHUSDT",
		Handler: func(_, _ string, _ []byte) {},
	}); err != nil {
		t.Fatalf("Handler-only subscription rejected: %v", err)
	}
}

func TestFrameTimeMs(t *testing.T) {
	if got := frameTimeMs(&Envelope{TsMs: 42}); got != 42 {
		t.Fatalf("frameTimeMs with ts=42: got %d", got)
	}
	var before int64 = time.Now().UnixMilli()
	var got int64 = frameTimeMs(&Envelope{})
	if got < before || got > time.Now().UnixMilli() {
		t.Fatalf("frameTimeMs without ts: got %d, want receive time", got)
	}
}
