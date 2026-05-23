/*
FILE: examples/public-streams/main.go

DESCRIPTION:
Subscribes to ALL public WS streams of the linears profile in parallel
and prints one aggregated status line every 500 ms. Keys are NOT required.

COVERAGE:
  - linears.Stream().WatchOrderBook (engine-backed, depth 50)
  - linears.Stream().WatchTicker     (delta merging on top of last snapshot)
  - linears.Stream().WatchTrades     (publicTrade.{symbol})
  - linears.Stream().WatchKline      (1-minute klines)

USAGE:

	go run ./examples/public-streams
	BYBIT_SYMBOL=ETHUSDT go run ./examples/public-streams
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
	"github.com/tonymontanov/go-bybit/v2/linears/types"
)

// snapshot — current values updated from different channels. Each field
// has its own lock domain to keep the print loop allocation-free.
type snapshot struct {
	mu sync.RWMutex

	bestBid, bestAsk string
	last, mark       string
	fundingRate      string
	klineClose       string
	klineConfirmed   bool

	tradesCount atomic.Uint64
	obUpdates   atomic.Uint64
	tickerSeen  atomic.Uint64
	klineSeen   atomic.Uint64
}

func main() {
	var opt exhelp.Options = exhelp.LoadEnv()
	var client, lc = exhelp.NewClient(opt)
	defer client.Close()

	var ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	var s snapshot
	var streamErr = func(e error) { log.Printf("stream: %s", exhelp.Classify(e)) }

	if err := lc.Stream().WatchOrderBook(ctx, opt.Symbol, 50, 1,
		func(ob types.OrderBookSnapshot) {
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

	if err := lc.Stream().WatchTicker(ctx, opt.Symbol,
		func(t types.TickerUpdate) {
			s.tickerSeen.Add(1)
			s.mu.Lock()
			s.last = t.LastPrice.String()
			s.mark = t.MarkPrice.String()
			s.fundingRate = t.FundingRate.String()
			s.mu.Unlock()
		}, streamErr); err != nil {
		log.Fatalf("WatchTicker: %s", exhelp.Classify(err))
	}

	if err := lc.Stream().WatchTrades(ctx, opt.Symbol,
		func(_ types.TradeUpdate) {
			s.tradesCount.Add(1)
		}, streamErr); err != nil {
		log.Fatalf("WatchTrades: %s", exhelp.Classify(err))
	}

	if err := lc.Stream().WatchKline(ctx, opt.Symbol, types.Timeframe1m,
		func(k types.KlineUpdate) {
			s.klineSeen.Add(1)
			s.mu.Lock()
			s.klineClose = k.Close.String()
			s.klineConfirmed = k.Confirmed
			s.mu.Unlock()
		}, streamErr); err != nil {
		log.Fatalf("WatchKline: %s", exhelp.Classify(err))
	}

	fmt.Printf("=== public-streams %s — Ctrl-C to stop ===\n\n", opt.Symbol)

	var sigCh chan os.Signal = make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	var ticker *time.Ticker = time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-sigCh:
			fmt.Printf("\nshutting down  ob=%d tk=%d kl=%d trades=%d\n",
				s.obUpdates.Load(), s.tickerSeen.Load(), s.klineSeen.Load(), s.tradesCount.Load())
			_ = lc.Stream().Close()
			return
		case <-ticker.C:
			s.mu.RLock()
			fmt.Printf("bid=%-10s ask=%-10s last=%-10s mark=%-10s fund=%-10s 1m close=%-10s confirmed=%v  trades=%d\n",
				s.bestBid, s.bestAsk, s.last, s.mark, s.fundingRate,
				s.klineClose, s.klineConfirmed, s.tradesCount.Load())
			s.mu.RUnlock()
		}
	}
}
