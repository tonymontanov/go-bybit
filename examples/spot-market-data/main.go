/*
FILE: examples/spot-market-data/main.go

DESCRIPTION:
Read-only public REST against the Bybit V5 spot category — instrument
specification, order book snapshot, historical candles. Keys are NOT
required.

COVERAGE:
  - spot.MarketData().GetSymbolInfo
  - spot.MarketData().GetOrderBook        (snapshot)
  - spot.MarketData().GetHistoricalCandles (1m, last 5)

NOTES:
  - Bybit V5 spot accepts orderbook depth from {1, 50, 200}; the SDK
    clamps any other value to the nearest allowed level.
  - SymbolInfo for spot exposes marginTrading / innovation flags
    instead of leverage filters (the latter is a derivatives concept).

USAGE:

	go run ./examples/spot-market-data
	BYBIT_SYMBOL=ETHUSDT go run ./examples/spot-market-data
*/

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/tonymontanov/go-bybit/examples/internal/exhelp"
	"github.com/tonymontanov/go-bybit/spot"
	bybitspottypes "github.com/tonymontanov/go-bybit/spot/types"
)

func main() {
	var opt exhelp.Options = exhelp.LoadEnv()
	var client, sc = exhelp.NewSpotClient(opt)
	defer client.Close()

	var ctx, cancel = context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	fmt.Printf("=== Spot market data for %s (testnet=%v demo=%v) ===\n\n", opt.Symbol, opt.Testnet, opt.Demo)

	dumpSymbolInfo(ctx, sc, opt.Symbol)
	dumpOrderBook(ctx, sc, opt.Symbol, 1, 10)
	dumpCandles(ctx, sc, opt.Symbol, bybitspottypes.Timeframe1m, 5)
}

func dumpSymbolInfo(ctx context.Context, sc *spot.Client, symbol string) {
	var info, err = sc.MarketData().GetSymbolInfo(ctx, symbol)
	if err != nil {
		fmt.Printf("[symbol-info] error: %s\n\n", exhelp.Classify(err))
		return
	}
	fmt.Println("[symbol-info]")
	fmt.Printf("  Symbol           = %s\n", info.Symbol)
	fmt.Printf("  Base / Quote     = %s / %s\n", info.BaseCoin, info.QuoteCoin)
	fmt.Printf("  Status           = %s   Innovation=%v\n", info.Status, info.Innovation)
	fmt.Printf("  TickSize         = %s   PricePrecision=%d\n", info.TickSize, info.PricePrecision)
	fmt.Printf("  BasePrecision    = %s   QuantityPrecision=%d\n", info.BasePrecision, info.QuantityPrecision)
	fmt.Printf("  MinOrderQty      = %s   MaxOrderQty=%s\n", info.MinOrderQty, info.MaxOrderQty)
	fmt.Printf("  MinOrderAmt      = %s   MaxOrderAmt=%s\n", info.MinOrderAmt, info.MaxOrderAmt)
	fmt.Printf("  MarginTrading    = %s\n\n", info.MarginTrading)
}

func dumpOrderBook(ctx context.Context, sc *spot.Client, symbol string, requestedDepth, printRows int) {
	var ob, err = sc.MarketData().GetOrderBook(ctx, symbol, requestedDepth)
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

func dumpCandles(ctx context.Context, sc *spot.Client, symbol string, tf bybitspottypes.Timeframe, n int) {
	var candles, err = sc.MarketData().GetHistoricalCandles(ctx, spot.HistoricalCandlesRequest{
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
