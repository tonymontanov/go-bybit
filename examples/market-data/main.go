/*
FILE: examples/market-data/main.go

DESCRIPTION:
Read-only public REST: instrument specification, order book snapshot,
historical candles. Keys and .env are NOT required for this example —
everything is public.

COVERAGE:
  - linears.MarketData().GetSymbolInfo
  - linears.MarketData().GetOrderBook        (snapshot)
  - linears.MarketData().GetHistoricalCandles (1m, last 5)

NOTES:
  - Bybit V5 only accepts orderbook depth from {1, 50, 200, 500}; the
    SDK clamps any other value to the nearest allowed level. We request
    a small depth and print the TOP-N levels (per side) to keep the
    output readable.

USAGE:

	go run ./examples/market-data
	BYBIT_SYMBOL=ETHUSDT go run ./examples/market-data
	# or with .env / scripts:
	./scripts/run.sh ./examples/market-data
*/

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/tonymontanov/go-bybit/examples/internal/exhelp"
	"github.com/tonymontanov/go-bybit/linears"
	"github.com/tonymontanov/go-bybit/linears/types"
)

func main() {
	var opt exhelp.Options = exhelp.LoadEnv()
	var client, lc = exhelp.NewClient(opt)
	defer client.Close()

	var ctx, cancel = context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	fmt.Printf("=== Market data for %s (testnet=%v demo=%v) ===\n\n", opt.Symbol, opt.Testnet, opt.Demo)

	dumpSymbolInfo(ctx, lc, opt.Symbol)
	// requestedDepth=1 → Bybit returns one level per side (no clamping).
	// printRows caps the on-screen output regardless of returned depth.
	dumpOrderBook(ctx, lc, opt.Symbol, 1, 10)
	dumpCandles(ctx, lc, opt.Symbol, types.Timeframe1m, 5)
}

func dumpSymbolInfo(ctx context.Context, lc *linears.Client, symbol string) {
	var info, err = lc.MarketData().GetSymbolInfo(ctx, symbol)
	if err != nil {
		fmt.Printf("[symbol-info] error: %s\n\n", exhelp.Classify(err))
		return
	}
	fmt.Println("[symbol-info]")
	fmt.Printf("  Symbol           = %s\n", info.Symbol)
	fmt.Printf("  Base / Quote     = %s / %s\n", info.BaseCoin, info.QuoteCoin)
	fmt.Printf("  ContractType     = %s   Status=%s\n", info.ContractType, info.Status)
	fmt.Printf("  TickSize         = %s   PricePrecision=%d\n", info.TickSize, info.PricePrecision)
	fmt.Printf("  QtyStep          = %s   QuantityPrecision=%d\n", info.QtyStep, info.QuantityPrecision)
	fmt.Printf("  MinOrderQty      = %s\n", info.MinOrderQty)
	fmt.Printf("  MaxOrderQty      = %s   MaxMarketOrderQty=%s\n", info.MaxOrderQty, info.MaxMarketOrderQty)
	fmt.Printf("  Leverage range   = [%s … %s] step %s\n\n",
		info.MinLeverage, info.MaxLeverage, info.LeverageStep)
}

func dumpOrderBook(ctx context.Context, lc *linears.Client, symbol string, requestedDepth, printRows int) {
	var ob, err = lc.MarketData().GetOrderBook(ctx, symbol, requestedDepth)
	if err != nil {
		fmt.Printf("[order-book] error: %s\n\n", exhelp.Classify(err))
		return
	}
	var asks int = len(ob.Asks)
	var bids int = len(ob.Bids)
	fmt.Printf("[order-book requested=%d returned=asks:%d/bids:%d ts=%d u=%d]\n",
		requestedDepth, asks, bids, ob.TsMs, ob.UpdateID)

	var askShown int = printRows
	if askShown > asks {
		askShown = asks
	}
	var bidShown int = printRows
	if bidShown > bids {
		bidShown = bids
	}

	fmt.Printf("  asks (top down → close to mid, showing %d/%d):\n", askShown, asks)
	var i int
	for i = askShown - 1; i >= 0; i-- {
		fmt.Printf("    %s × %s\n", ob.Asks[i].Price, ob.Asks[i].Size)
	}
	if asks > 0 && bids > 0 {
		var spread = ob.Asks[0].Price.Sub(ob.Bids[0].Price)
		fmt.Printf("  --- spread = %s ---\n", spread)
	}
	fmt.Printf("  bids (showing %d/%d):\n", bidShown, bids)
	for i = 0; i < bidShown; i++ {
		fmt.Printf("    %s × %s\n", ob.Bids[i].Price, ob.Bids[i].Size)
	}
	fmt.Println()
}

func dumpCandles(ctx context.Context, lc *linears.Client, symbol string, tf types.Timeframe, n int) {
	var candles, err = lc.MarketData().GetHistoricalCandles(ctx, linears.HistoricalCandlesRequest{
		Symbol:    symbol,
		Timeframe: tf,
		Limit:     n,
	})
	if err != nil {
		fmt.Printf("[candles] error: %s\n\n", exhelp.Classify(err))
		return
	}
	fmt.Printf("[candles %s count=%d (newest first)]\n", tf, len(candles))
	var i int
	for i = 0; i < len(candles); i++ {
		var c = candles[i]
		var t time.Time = time.UnixMilli(c.OpenTimeMs)
		fmt.Printf("  %s  O=%s  H=%s  L=%s  C=%s  V=%s\n",
			t.UTC().Format("2006-01-02 15:04:05"),
			c.Open, c.High, c.Low, c.Close, c.Volume)
	}
	fmt.Println()
}
