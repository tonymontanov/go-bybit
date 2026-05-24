/*
FILE: examples/broker-account-info/main.go

DESCRIPTION:
Read-only smoke test for the exchange broker profile (C4). Requires a
broker master account API key.

COVERAGE:
  - broker.Client.GetAccountInfo
  - broker.Client.GetEarnings (last 7 days default window)

USAGE:

	# 1) cp .env.example .env  →  fill BYBIT_API_KEY / BYBIT_API_SECRET
	# 2) optionally set BYBIT_TESTNET=1
	./scripts/run.sh ./examples/broker-account-info
*/

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/tonymontanov/go-bybit/v2/broker"
	brokertypes "github.com/tonymontanov/go-bybit/v2/broker/types"
	"github.com/tonymontanov/go-bybit/v2/examples/internal/exhelp"
)

func main() {
	var opt exhelp.Options = exhelp.LoadEnv()
	exhelp.MustHaveKeys(opt)

	var client, bc = exhelp.NewBrokerClient(opt)
	defer client.Close()

	var ctx, cancel = context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	fmt.Printf("=== Broker account info (testnet=%v demo=%v) ===\n\n",
		opt.Testnet, opt.Demo)

	dumpBrokerAccount(ctx, bc)
	dumpBrokerEarnings(ctx, bc)
}

func dumpBrokerAccount(ctx context.Context, bc *broker.Client) {
	var info brokertypes.AccountInfo
	var err error
	info, err = bc.GetAccountInfo(ctx)
	if err != nil {
		fmt.Printf("[broker-account] error: %s\n\n", exhelp.Classify(err))
		return
	}
	fmt.Println("[broker-account]")
	fmt.Printf("  subAccounts=%s / max=%s\n", info.SubAccountQty, info.MaxSubAccountQty)
	fmt.Printf("  baseRebate spot=%s derivatives=%s\n",
		info.BaseFeeRebateRate.Spot, info.BaseFeeRebateRate.Derivatives)
	fmt.Printf("  markupRebate spot=%s derivatives=%s convert=%s\n\n",
		info.MarkupFeeRebateRate.Spot, info.MarkupFeeRebateRate.Derivatives,
		info.MarkupFeeRebateRate.Convert)
}

func dumpBrokerEarnings(ctx context.Context, bc *broker.Client) {
	var page brokertypes.EarningsList
	var err error
	page, err = bc.GetEarnings(ctx, brokertypes.EarningsRequest{
		Limit: 5,
	})
	if err != nil {
		fmt.Printf("[broker-earnings] error: %s\n\n", exhelp.Classify(err))
		return
	}
	fmt.Printf("[broker-earnings details=%d]\n", len(page.Details))
	if len(page.Details) == 0 {
		fmt.Println("  (empty)")
		fmt.Println()
		return
	}
	var row = page.Details[0]
	fmt.Printf("  uid=%s %s %s earning=%s execMs=%d\n\n",
		row.UserID, row.BizType, row.Symbol, row.Earning, row.ExecTimeMs)
}
