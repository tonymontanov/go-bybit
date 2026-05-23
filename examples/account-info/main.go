/*
FILE: examples/account-info/main.go

DESCRIPTION:
Read-only smoke-test of the private REST surface. Does NOT place orders
and does NOT modify account state. Useful as a first check that your API
keys are valid and the SDK signs requests correctly.

COVERAGE:
  - linears.MarketData().GetSymbolInfo (public, for instrument spec)
  - linears.Account().GetWalletBalance (UNIFIED account)
  - linears.Account().GetPosition
  - linears.Account().GetOpenOrders

USAGE:

	# 1) cp .env.example .env  →  fill BYBIT_API_KEY / BYBIT_API_SECRET
	# 2) optionally set BYBIT_TESTNET=1 to point at testnet
	./scripts/run.sh ./examples/account-info
*/

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/tonymontanov/go-bybit/v2/examples/internal/exhelp"
	"github.com/tonymontanov/go-bybit/v2/linears"
	"github.com/tonymontanov/go-bybit/v2/linears/types"
)

func main() {
	var opt exhelp.Options = exhelp.LoadEnv()
	exhelp.MustHaveKeys(opt)

	var client, lc = exhelp.NewClient(opt)
	defer client.Close()

	var ctx, cancel = context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	fmt.Printf("=== Account info for %s (testnet=%v demo=%v) ===\n\n",
		opt.Symbol, opt.Testnet, opt.Demo)

	dumpSymbolInfo(ctx, lc, opt.Symbol)
	dumpWallet(ctx, lc)
	dumpPosition(ctx, lc, opt.Symbol)
	dumpOpenOrders(ctx, lc, opt.Symbol)
}

func dumpSymbolInfo(ctx context.Context, lc *linears.Client, symbol string) {
	var info, err = lc.MarketData().GetSymbolInfo(ctx, symbol)
	if err != nil {
		fmt.Printf("[symbol-info] error: %s\n\n", exhelp.Classify(err))
		return
	}
	fmt.Println("[symbol-info]")
	fmt.Printf("  Symbol=%s  Status=%s\n", info.Symbol, info.Status)
	fmt.Printf("  TickSize=%s  QtyStep=%s  MinOrderQty=%s\n\n",
		info.TickSize, info.QtyStep, info.MinOrderQty)
}

func dumpWallet(ctx context.Context, lc *linears.Client) {
	var bal, err = lc.Account().GetWalletBalance(ctx, linears.WalletBalanceRequest{
		AccountType: types.AccountTypeUnified,
	})
	if err != nil {
		fmt.Printf("[wallet] error: %s\n\n", exhelp.Classify(err))
		return
	}
	fmt.Println("[wallet]")
	fmt.Printf("  AccountType        = %s\n", bal.AccountType)
	fmt.Printf("  TotalEquity        = %s\n", bal.TotalEquity)
	fmt.Printf("  AvailableBalance   = %s\n", bal.TotalAvailableBalance)
	fmt.Printf("  TotalMarginBalance = %s   IM=%s  MM=%s\n",
		bal.TotalMarginBalance, bal.TotalInitialMargin, bal.TotalMaintenanceMargin)
	fmt.Printf("  AccountIMRate=%s  AccountMMRate=%s  PerpUPL=%s\n",
		bal.AccountIMRate, bal.AccountMMRate, bal.TotalPerpUPL)
	if len(bal.Coins) == 0 {
		fmt.Println("  coins: (empty)")
	} else {
		fmt.Println("  coins:")
		var i int
		for i = 0; i < len(bal.Coins); i++ {
			var c = bal.Coins[i]
			fmt.Printf("    %-6s equity=%s wallet=%s avail=%s uPnL=%s realized=%s\n",
				c.Coin, c.Equity, c.WalletBalance,
				c.AvailableToWithdraw, c.UnrealizedPnL, c.CumRealizedPnL)
		}
	}
	fmt.Println()
}

func dumpPosition(ctx context.Context, lc *linears.Client, symbol string) {
	var rows, err = lc.Account().GetPosition(ctx, symbol)
	if err != nil {
		fmt.Printf("[position] error: %s\n\n", exhelp.Classify(err))
		return
	}
	fmt.Println("[position]")
	if len(rows) == 0 {
		fmt.Println("  (no rows; this account never traded the symbol)")
		fmt.Println()
		return
	}
	var i int
	for i = 0; i < len(rows); i++ {
		var p = rows[i]
		if p.IsEmpty() {
			fmt.Printf("  idx=%d  EMPTY (Side=%q)\n", p.PositionIdx, p.Side)
			continue
		}
		fmt.Printf("  idx=%d  %s qty=%s avgPx=%s mark=%s liqPx=%s lev=%s uPnL=%s value=%s updMs=%d\n",
			p.PositionIdx, p.Side,
			p.Quantity, p.AvgEntryPrice, p.MarkPrice, p.LiqPrice, p.Leverage,
			p.UnrealizedPnL, p.PositionValue, p.UpdatedAtMs)
	}
	fmt.Println()
}

func dumpOpenOrders(ctx context.Context, lc *linears.Client, symbol string) {
	var orders, err = lc.Account().GetOpenOrders(ctx, symbol)
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
		fmt.Printf("  #%d ordId=%s clOrdId=%q %s %s tif=%s state=%s qty=%s leaves=%s price=%s avg=%s\n",
			i, o.OrderID, o.ClientOrderID, o.Side, o.OrderType, o.TimeInForce, o.Status,
			o.Quantity, o.LeavesQty, o.Price, o.AvgPrice)
	}
	if len(orders) > 0 && orders[0].RateLimits != nil {
		fmt.Printf("  rate-limit headers: %v\n", orders[0].RateLimits)
	}
	fmt.Println()
}
