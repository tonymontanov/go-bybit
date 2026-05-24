/*
FILE: examples/premarket-instruments/main.go

DESCRIPTION:
Read-only smoke test for the pre-market profile (C6). Keys are NOT required.

COVERAGE:
  - premarket.Client.GetPreLaunchInstruments
  - premarket.Client.GetRiskLimit
  - premarket.Client.GetTickers

USAGE:

	go run ./examples/premarket-instruments
	BYBIT_SYMBOL=BIOUSDT go run ./examples/premarket-instruments
	./scripts/run.sh ./examples/premarket-instruments
*/

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/tonymontanov/go-bybit/v2/examples/internal/exhelp"
	"github.com/tonymontanov/go-bybit/v2/premarket"
	pmtypes "github.com/tonymontanov/go-bybit/v2/premarket/types"
	commontypes "github.com/tonymontanov/go-bybit/v2/types"
)

func main() {
	var opt exhelp.Options = exhelp.LoadEnv()
	var client, pc = exhelp.NewPreMarketClient(opt)
	defer client.Close()

	var ctx, cancel = context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	fmt.Printf("=== Pre-market instruments (testnet=%v demo=%v) ===\n\n",
		opt.Testnet, opt.Demo)

	dumpPreLaunch(ctx, pc)
	if opt.Symbol != "" {
		dumpRiskLimit(ctx, pc, opt.Symbol)
		dumpTicker(ctx, pc, opt.Symbol)
	}
}

func dumpPreLaunch(ctx context.Context, pc *premarket.Client) {
	var page, err = pc.GetPreLaunchInstruments(ctx, commontypes.CategoryLinear, "")
	if err != nil {
		fmt.Printf("[prelaunch-instruments] error: %s\n\n", exhelp.Classify(err))
		return
	}
	fmt.Printf("[prelaunch-instruments count=%d cursor=%q]\n", len(page.Instruments), page.NextPageCursor)
	if len(page.Instruments) == 0 {
		fmt.Println("  (empty — no PreLaunch linear symbols right now)")
		fmt.Println()
		return
	}
	var inst = page.Instruments[0]
	fmt.Printf("  symbol=%s status=%s isPreListing=%v\n", inst.Symbol, inst.Status, inst.IsPreListing)
	if inst.PreListingInfo != nil {
		fmt.Printf("  curAuctionPhase=%s phases=%d takerFee=%s\n\n",
			inst.PreListingInfo.CurAuctionPhase,
			len(inst.PreListingInfo.Phases),
			inst.PreListingInfo.AuctionFeeInfo.TakerFeeRate)
	} else {
		fmt.Println()
	}
}

func dumpRiskLimit(ctx context.Context, pc *premarket.Client, symbol string) {
	var page, err = pc.GetRiskLimit(ctx, pmtypes.RiskLimitRequest{
		Category: commontypes.CategoryLinear,
		Symbol:   symbol,
	})
	if err != nil {
		fmt.Printf("[risk-limit %s] error: %s\n\n", symbol, exhelp.Classify(err))
		return
	}
	fmt.Printf("[risk-limit %s tiers=%d]\n", symbol, len(page.Tiers))
	if len(page.Tiers) > 0 {
		var tier = page.Tiers[0]
		fmt.Printf("  id=%d riskLimit=%s maxLeverage=%s\n\n",
			tier.ID, tier.RiskLimitValue, tier.MaxLeverage)
	} else {
		fmt.Println()
	}
}

func dumpTicker(ctx context.Context, pc *premarket.Client, symbol string) {
	var page, err = pc.GetTickers(ctx, commontypes.CategoryLinear, symbol)
	if err != nil {
		fmt.Printf("[ticker %s] error: %s\n\n", symbol, exhelp.Classify(err))
		return
	}
	if len(page.Tickers) == 0 {
		fmt.Printf("[ticker %s] (empty)\n\n", symbol)
		return
	}
	var tk = page.Tickers[0]
	fmt.Printf("[ticker %s last=%s preOpen=%s preQty=%s phase=%q]\n\n",
		tk.Symbol, tk.LastPrice, tk.PreOpenPrice, tk.PreQty, tk.CurPreListingPhase)
}
