/*
FILE: examples/spot-orderbook-watcher/main.go

DESCRIPTION:
Subscribes to the public BTCUSDT spot order book at depth 50 and prints
the top of book every 500 ms. Mirrors examples/orderbook-watcher; keys
are NOT required.

NOTES:
  - Bybit V5 spot accepts orderbook depth from {1, 50, 200} — pick one
    of those when calling WatchOrderBook (the SDK clamps).
  - `displayDepth` controls how many levels reach the handler (set to
    5 here so the screen output stays readable).

USAGE:

	go run ./examples/spot-orderbook-watcher
	BYBIT_SYMBOL=ETHUSDT go run ./examples/spot-orderbook-watcher
*/

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/tonymontanov/go-bybit/v2/examples/internal/exhelp"
	bybitspottypes "github.com/tonymontanov/go-bybit/v2/spot/types"
)

func main() {
	var opt exhelp.Options = exhelp.LoadEnv()
	var client, sc = exhelp.NewSpotClient(opt)
	defer client.Close()

	var ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	var (
		mu      sync.RWMutex
		topBid  string
		topAsk  string
		bidSize string
		askSize string
		updates atomic.Uint64
		lastU   atomic.Int64
	)

	var watchErr = sc.Stream().WatchOrderBook(ctx, opt.Symbol, 50, 5,
		func(ob bybitspottypes.OrderBookSnapshot) {
			updates.Add(1)
			lastU.Store(ob.UpdateID)
			mu.Lock()
			if len(ob.Bids) > 0 {
				topBid = ob.Bids[0].Price.String()
				bidSize = ob.Bids[0].Size.String()
			}
			if len(ob.Asks) > 0 {
				topAsk = ob.Asks[0].Price.String()
				askSize = ob.Asks[0].Size.String()
			}
			mu.Unlock()
		},
		func(streamErr error) {
			log.Printf("orderbook stream: %s", exhelp.Classify(streamErr))
		},
	)
	if watchErr != nil {
		log.Fatalf("WatchOrderBook: %s", exhelp.Classify(watchErr))
	}

	fmt.Printf("=== spot-orderbook-watcher %s — Ctrl-C to stop ===\n\n", opt.Symbol)

	var sigCh chan os.Signal = make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	var ticker *time.Ticker = time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-sigCh:
			fmt.Printf("\nshutting down, total updates=%d lastU=%d\n", updates.Load(), lastU.Load())
			_ = sc.Stream().Close()
			return
		case <-ticker.C:
			mu.RLock()
			fmt.Printf("u=%-10d updates=%-6d  bid=%-12s × %-8s  ask=%-12s × %-8s\n",
				lastU.Load(), updates.Load(), topBid, bidSize, topAsk, askSize)
			mu.RUnlock()
		}
	}
}
