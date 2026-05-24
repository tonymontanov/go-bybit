/*
FILE: examples/asset-coin-info/main.go

Read-only smoke-test of the Bybit V5 asset REST surface (coin metadata,
account coin balances, withdrawable amount). Does NOT submit transfers
or withdrawals.

	./scripts/run.sh ./examples/asset-coin-info
*/

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/tonymontanov/go-bybit/v2/asset"
	"github.com/tonymontanov/go-bybit/v2/examples/internal/exhelp"
	commontypes "github.com/tonymontanov/go-bybit/v2/types"
)

func main() {
	var opt exhelp.Options = exhelp.LoadEnv()
	exhelp.MustHaveKeys(opt)

	var client, ac = exhelp.NewAssetClient(opt)
	defer client.Close()

	var ctx, cancel = context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	fmt.Printf("=== Asset info (testnet=%v demo=%v) ===\n\n", opt.Testnet, opt.Demo)

	dumpCoinInfo(ctx, ac, "USDT")
	dumpAllCoinsBalance(ctx, ac)
	dumpWithdrawable(ctx, ac, "USDT")
}

func dumpCoinInfo(ctx context.Context, ac *asset.Client, coin string) {
	var rows, err = ac.GetCoinInfo(ctx, coin)
	if err != nil {
		fmt.Printf("[coin-info] error: %s\n\n", exhelp.Classify(err))
		return
	}
	fmt.Println("[coin-info]")
	for _, row := range rows {
		fmt.Printf("  %s (%s) chains=%d\n", row.Coin, row.Name, len(row.Chains))
		for _, ch := range row.Chains {
			fmt.Printf("    chain=%s deposit=%s withdraw=%s fee=%s\n",
				ch.Chain, ch.ChainDeposit, ch.ChainWithdraw, ch.WithdrawFee)
		}
	}
	fmt.Println()
}

func dumpAllCoinsBalance(ctx context.Context, ac *asset.Client) {
	var bal, err = ac.GetAllCoinsBalance(ctx, commontypes.AccountTypeUnified, "")
	if err != nil {
		fmt.Printf("[coins-balance] error: %s\n\n", exhelp.Classify(err))
		return
	}
	fmt.Println("[coins-balance UNIFIED]")
	for _, c := range bal {
		if c.WalletBalance.IsZero() {
			continue
		}
		fmt.Printf("  %-6s wallet=%s transfer=%s\n", c.Coin, c.WalletBalance, c.TransferBalance)
	}
	fmt.Println()
}

func dumpWithdrawable(ctx context.Context, ac *asset.Client, coin string) {
	var amt, err = ac.GetWithdrawableAmount(ctx, coin)
	if err != nil {
		fmt.Printf("[withdrawable] error: %s\n\n", exhelp.Classify(err))
		return
	}
	fmt.Println("[withdrawable]")
	fmt.Printf("  coin=%s amount=%s usd=%s limitUsd=%s\n\n",
		coin, amt.WithdrawableAmount, amt.WithdrawableAmountUsd, amt.LimitAmountUsd)
}
