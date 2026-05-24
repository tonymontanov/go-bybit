/*
FILE: examples/extended-account-info/main.go

DESCRIPTION:
Read-only smoke test for the extended account profile (C2). Does NOT
modify account state. Complements examples/account-info (linears wallet/
positions/orders).

COVERAGE:
  - account.Client.GetAccountInfo
  - account.Client.GetFeeRate (linear category)
  - account.Client.GetCollateralInfo

USAGE:

	# 1) cp .env.example .env  →  fill BYBIT_API_KEY / BYBIT_API_SECRET
	# 2) optionally set BYBIT_TESTNET=1
	./scripts/run.sh ./examples/extended-account-info
*/

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/tonymontanov/go-bybit/v2/account"
	accounttypes "github.com/tonymontanov/go-bybit/v2/account/types"
	"github.com/tonymontanov/go-bybit/v2/examples/internal/exhelp"
	commontypes "github.com/tonymontanov/go-bybit/v2/types"
)

func main() {
	var opt exhelp.Options = exhelp.LoadEnv()
	exhelp.MustHaveKeys(opt)

	var client, ac = exhelp.NewAccountClient(opt)
	defer client.Close()

	var ctx, cancel = context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	fmt.Printf("=== Extended account info (testnet=%v demo=%v) ===\n\n",
		opt.Testnet, opt.Demo)

	dumpAccountInfo(ctx, ac)
	dumpFeeRate(ctx, ac, opt.Symbol)
	dumpCollateral(ctx, ac, "USDT")
}

func dumpAccountInfo(ctx context.Context, ac *account.Client) {
	var info accounttypes.AccountInfo
	var err error
	info, err = ac.GetAccountInfo(ctx)
	if err != nil {
		fmt.Printf("[account-info] error: %s\n\n", exhelp.Classify(err))
		return
	}
	fmt.Println("[account-info]")
	fmt.Printf("  MarginMode=%s  UTAStatus=%d  SpotHedging=%s\n",
		info.MarginMode, info.UnifiedMarginStatus, info.SpotHedgingStatus)
	fmt.Printf("  IsMasterTrader=%v  UpdatedAtMs=%d\n\n",
		info.IsMasterTrader, info.UpdatedAtMs)
}

func dumpFeeRate(ctx context.Context, ac *account.Client, symbol string) {
	var list accounttypes.FeeRateList
	var err error
	list, err = ac.GetFeeRate(ctx, accounttypes.FeeRateRequest{
		Category: commontypes.CategoryLinear,
		Symbol:   symbol,
	})
	if err != nil {
		fmt.Printf("[fee-rate] error: %s\n\n", exhelp.Classify(err))
		return
	}
	fmt.Println("[fee-rate]")
	if len(list.List) == 0 {
		fmt.Println("  (empty)")
	} else {
		var row = list.List[0]
		fmt.Printf("  %s maker=%s taker=%s\n",
			row.Symbol, row.MakerFeeRate, row.TakerFeeRate)
	}
	fmt.Println()
}

func dumpCollateral(ctx context.Context, ac *account.Client, currency string) {
	var rows []accounttypes.CollateralInfo
	var err error
	rows, err = ac.GetCollateralInfo(ctx, currency)
	if err != nil {
		fmt.Printf("[collateral] error: %s\n\n", exhelp.Classify(err))
		return
	}
	fmt.Println("[collateral]")
	if len(rows) == 0 {
		fmt.Println("  (empty)")
		fmt.Println()
		return
	}
	var row = rows[0]
	fmt.Printf("  %s borrowable=%v collateralSwitch=%v hourlyRate=%s availBorrow=%s\n",
		row.Currency, row.Borrowable, row.CollateralSwitch,
		row.HourlyBorrowRate, row.AvailableToBorrow)
	fmt.Println()
}
