/*
FILE: linears/stream-ts_test.go

DESCRIPTION:
TsMs of WS tickers and orderbook snapshots comes from the push envelope
"ts" (the data of these topics carries no timestamp of its own); a frame
without "ts" gets the local receive time, never zero. Callers use TsMs as
the price time — a zero there made the desk skip every tick (2026-09-29).
*/

package linears

import (
	"context"
	"testing"
	"time"

	"github.com/tonymontanov/go-bybit/v2/linears/types"
)

func TestStream_WatchTicker_TsMsFromEnvelope(t *testing.T) {
	const publishTs int64 = 1700000000123

	var public *fakeWS = newFakeWS(t, func(s *fakeWS) {
		var _, ok = s.readUntilOp("subscribe", 2*time.Second)
		if !ok {
			return
		}
		_ = s.writeJSON(map[string]any{"op": "subscribe", "success": true})
		_ = s.writeJSON(map[string]any{
			"topic": "tickers.BTCUSDT", "type": "snapshot", "ts": publishTs,
			"data": map[string]any{"symbol": "BTCUSDT", "lastPrice": "60000"},
		})
		// No "ts" in the envelope — receive time is expected.
		_ = s.writeJSON(map[string]any{
			"topic": "tickers.BTCUSDT", "type": "delta",
			"data": map[string]any{"symbol": "BTCUSDT", "lastPrice": "60100"},
		})
	})

	var lc *Client = newStreamTestClient(t, public, nil)

	var ticks = make(chan types.TickerUpdate, 4)
	var ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	var before int64 = time.Now().UnixMilli()
	if err := lc.Stream().WatchTicker(ctx, "BTCUSDT",
		func(tk types.TickerUpdate) { ticks <- tk },
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
		t.Fatalf("snapshot not delivered")
	}

	select {
	case tk := <-ticks:
		var after int64 = time.Now().UnixMilli()
		if tk.TsMs < before || tk.TsMs > after {
			t.Fatalf("frame without ts: TsMs=%d want receive time in [%d, %d]", tk.TsMs, before, after)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("delta not delivered")
	}
}

func TestStream_WatchOrderBook_TsMsFromEnvelope(t *testing.T) {
	const publishTs int64 = 1700000000456

	var public *fakeWS = newFakeWS(t, func(s *fakeWS) {
		var _, ok = s.readUntilOp("subscribe", 2*time.Second)
		if !ok {
			return
		}
		_ = s.writeJSON(map[string]any{"op": "subscribe", "success": true})
		_ = s.writeJSON(map[string]any{
			"topic": "orderbook.1.BTCUSDT", "type": "snapshot", "ts": publishTs,
			"data": map[string]any{
				"s": "BTCUSDT", "u": 100, "seq": 1,
				"b": [][]string{{"60000", "1.0"}},
				"a": [][]string{{"60001", "0.7"}},
			},
		})
	})

	var lc *Client = newStreamTestClient(t, public, nil)

	var snaps = make(chan types.OrderBookSnapshot, 2)
	var ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	if err := lc.Stream().WatchOrderBook(ctx, "BTCUSDT", 1, 1,
		func(ob types.OrderBookSnapshot) { snaps <- ob },
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
