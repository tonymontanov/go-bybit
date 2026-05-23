/*
FILE: examples/spot-account-info/main.go

DESCRIPTION:
Read-only smoke-test of the private REST surface for the Bybit V5 spot
profile. Does NOT place orders and does NOT modify account state.
Useful as a first check that your API keys are valid and the SDK signs
requests correctly.

COVERAGE:
  - spot.MarketData().GetSymbolInfo (public, for instrument spec)
  - spot.Account().GetWalletBalance (UNIFIED account)
  - spot.Account().GetOpenOrders

NOTE:
  - Spot has no positions — the linears example's GetPosition has no
    counterpart. Holdings show up under wallet.coin[].
  - Wallet accountType=UNIFIED is required for UTA accounts; classic
    spot keys can use AccountTypeSpot via WalletBalanceRequest.

USAGE:

	# 1) cp .env.example .env  →  fill BYBIT_API_KEY / BYBIT_API_SECRET
	# 2) optionally set BYBIT_TESTNET=1 to point at testnet
	./scripts/run.sh ./examples/spot-account-info
*/

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/tonymontanov/go-bybit/v2/examples/internal/exhelp"
	"github.com/tonymontanov/go-bybit/v2/spot"
	bybitspottypes "github.com/tonymontanov/go-bybit/v2/spot/types"
)

func main() {
	var opt exhelp.Options = exhelp.LoadEnv()
	exhelp.MustHaveKeys(opt)

	var client, sc = exhelp.NewSpotClient(opt)
	defer client.Close()

	var ctx, cancel = context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	fmt.Printf("=== Spot account info for %s (testnet=%v demo=%v) ===\n\n",
		opt.Symbol, opt.Testnet, opt.Demo)

	dumpSymbolInfo(ctx, sc, opt.Symbol)
	dumpWallet(ctx, sc)
	dumpOpenOrders(ctx, sc, opt.Symbol)
}

func dumpSymbolInfo(ctx context.Context, sc *spot.Client, symbol string) {
	var info, err = sc.MarketData().GetSymbolInfo(ctx, symbol)
	if err != nil {
		fmt.Printf("[symbol-info] error: %s\n\n", exhelp.Classify(err))
		return
	}
	fmt.Println("[symbol-info]")
	fmt.Printf("  Symbol=%s  Status=%s  MarginTrading=%s  Innovation=%v\n", info.Symbol, info.Status, info.MarginTrading, info.Innovation)
	fmt.Printf("  TickSize=%s  BasePrecision=%s  MinOrderQty=%s  MinOrderAmt=%s\n\n",
		info.TickSize, info.BasePrecision, info.MinOrderQty, info.MinOrderAmt)
}

func dumpWallet(ctx context.Context, sc *spot.Client) {
	var bal, err = sc.Account().GetWalletBalance(ctx, spot.WalletBalanceRequest{
		AccountType: bybitspottypes.AccountTypeUnified,
	})
	if err != nil {
		fmt.Printf("[wallet] error: %s\n\n", exhelp.Classify(err))
		return
	}
	fmt.Println("[wallet]")
	fmt.Printf("  AccountType        = %s\n", bal.AccountType)
	fmt.Printf("  TotalEquity        = %s\n", bal.TotalEquity)
	fmt.Printf("  AvailableBalance   = %s\n", bal.TotalAvailableBalance)
	if !bal.TotalMarginBalance.IsZero() {
		fmt.Printf("  TotalMarginBalance = %s   IM=%s  MM=%s\n",
			bal.TotalMarginBalance, bal.TotalInitialMargin, bal.TotalMaintenanceMargin)
	}
	if len(bal.Coins) == 0 {
		fmt.Println("  coins: (empty)")
	} else {
		fmt.Println("  coins:")
		var i int
		for i = 0; i < len(bal.Coins); i++ {
			var c = bal.Coins[i]
			if c.WalletBalance.IsZero() && c.Equity.IsZero() {
				continue
			}
			fmt.Printf("    %-6s wallet=%s available=%s usdValue=%s\n",
				c.Coin, c.WalletBalance, c.AvailableToWithdraw, c.UsdValue)
		}
	}
	fmt.Println()
}

func dumpOpenOrders(ctx context.Context, sc *spot.Client, symbol string) {
	var orders, err = sc.Account().GetOpenOrders(ctx, symbol)
	if err != nil {
		fmt.Printf("[open-orders] error: %s\n\n", exhelp.Classify(err))
		return
	}
	fmt.Println("[open-orders]")
	if len(orders) == 0 {
		fmt.Println("  (none)")
		fmt.Println()
		return
	}
	var i int
	for i = 0; i < len(orders); i++ {
		var o = orders[i]
		fmt.Printf("  #%d ordId=%s clOrdId=%q %s %s tif=%s state=%s qty=%s leaves=%s price=%s avg=%s isLeverage=%v\n",
			i, o.OrderID, o.ClientOrderID, o.Side, o.OrderType, o.TimeInForce, o.Status,
			o.Quantity, o.LeavesQty, o.Price, o.AvgPrice, o.IsLeverage)
	}
	if len(orders) > 0 && orders[0].RateLimits != nil {
		fmt.Printf("  rate-limit headers: %v\n", orders[0].RateLimits)
	}
	fmt.Println()
}
