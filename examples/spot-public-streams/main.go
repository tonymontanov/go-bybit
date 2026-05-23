/*
FILE: examples/spot-public-streams/main.go

DESCRIPTION:
Subscribes to all public WS streams of the spot profile in parallel and
prints one aggregated status line every 500 ms. Keys are NOT required.

COVERAGE:
  - spot.Stream().WatchOrderBook (engine-backed, depth 50)
  - spot.Stream().WatchTicker     (delta merging on top of last snapshot)
  - spot.Stream().WatchTrades     (publicTrade.{symbol})
  - spot.Stream().WatchKline      (1-minute klines)

USAGE:

	go run ./examples/spot-public-streams
	BYBIT_SYMBOL=ETHUSDT go run ./examples/spot-public-streams
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

type snapshot struct {
	mu sync.RWMutex

	bestBid, bestAsk string
	last             string
	usdIndex         string
	klineClose       string
	klineConfirmed   bool

	tradesCount atomic.Uint64
	obUpdates   atomic.Uint64
	tickerSeen  atomic.Uint64
	klineSeen   atomic.Uint64
}

func main() {
	var opt exhelp.Options = exhelp.LoadEnv()
	var client, sc = exhelp.NewSpotClient(opt)
	defer client.Close()

	var ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	var s snapshot
	var streamErr = func(e error) { log.Printf("stream: %s", exhelp.Classify(e)) }

	if err := sc.Stream().WatchOrderBook(ctx, opt.Symbol, 50, 1,
		func(ob bybitspottypes.OrderBookSnapshot) {
			s.obUpdates.Add(1)
			s.mu.Lock()
			if len(ob.Bids) > 0 {
				s.bestBid = ob.Bids[0].Price.String()
			}
			if len(ob.Asks) > 0 {
				s.bestAsk = ob.Asks[0].Price.String()
			}
			s.mu.Unlock()
		}, streamErr); err != nil {
		log.Fatalf("WatchOrderBook: %s", exhelp.Classify(err))
	}

	if err := sc.Stream().WatchTicker(ctx, opt.Symbol,
		func(t bybitspottypes.TickerUpdate) {
			s.tickerSeen.Add(1)
			s.mu.Lock()
			s.last = t.LastPrice.String()
			s.usdIndex = t.UsdIndexPrice.String()
			s.mu.Unlock()
		}, streamErr); err != nil {
		log.Fatalf("WatchTicker: %s", exhelp.Classify(err))
	}

	if err := sc.Stream().WatchTrades(ctx, opt.Symbol,
		func(_ bybitspottypes.TradeUpdate) {
			s.tradesCount.Add(1)
		}, streamErr); err != nil {
		log.Fatalf("WatchTrades: %s", exhelp.Classify(err))
	}

	if err := sc.Stream().WatchKline(ctx, opt.Symbol, bybitspottypes.Timeframe1m,
		func(k bybitspottypes.KlineUpdate) {
			s.klineSeen.Add(1)
			s.mu.Lock()
			s.klineClose = k.Close.String()
			s.klineConfirmed = k.Confirmed
			s.mu.Unlock()
		}, streamErr); err != nil {
		log.Fatalf("WatchKline: %s", exhelp.Classify(err))
	}

	fmt.Printf("=== spot-public-streams %s — Ctrl-C to stop ===\n\n", opt.Symbol)

	var sigCh chan os.Signal = make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	var ticker *time.Ticker = time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-sigCh:
			fmt.Printf("\nshutting down  ob=%d tk=%d kl=%d trades=%d\n",
				s.obUpdates.Load(), s.tickerSeen.Load(), s.klineSeen.Load(), s.tradesCount.Load())
			_ = sc.Stream().Close()
			return
		case <-ticker.C:
			s.mu.RLock()
			fmt.Printf("bid=%-10s ask=%-10s last=%-10s usdIdx=%-10s 1m close=%-10s confirmed=%v  trades=%d\n",
				s.bestBid, s.bestAsk, s.last, s.usdIndex,
				s.klineClose, s.klineConfirmed, s.tradesCount.Load())
			s.mu.RUnlock()
		}
	}
}
