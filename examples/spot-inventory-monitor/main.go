/*
FILE: examples/spot-inventory-monitor/main.go

DESCRIPTION:
Long-running monitor for the user's own state on Bybit spot. Subscribes
to the private "order", "execution" and "wallet" topics and prints every
update in real time. Does NOT place trades — read-only.

This is the right shape for an algorithm that must keep its picture of
the exchange state warm even when no orders are flowing.

COVERAGE:
  - spot.Stream().WatchOrders        (own orders, filtered to spot)
  - spot.Stream().WatchExecutions    (own fills, spot)
  - spot.Stream().WatchWallet        (own wallet)
  - initial REST snapshots: GetWalletBalance + GetOpenOrders.

REQUIRES:
  - UTA-enabled API keys (Bybit V5 private spot streams are UTA-only).

USAGE:

	./scripts/run.sh ./examples/spot-inventory-monitor
	BYBIT_SYMBOL=ETHUSDT ./scripts/run.sh ./examples/spot-inventory-monitor
*/

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/tonymontanov/go-bybit/examples/internal/exhelp"
	"github.com/tonymontanov/go-bybit/spot"
	bybitspottypes "github.com/tonymontanov/go-bybit/spot/types"
)

func main() {
	var opt exhelp.Options = exhelp.LoadEnv()
	exhelp.MustHaveKeys(opt)

	var client, sc = exhelp.NewSpotClient(opt)
	defer client.Close()

	var ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	fmt.Printf("=== spot-inventory-monitor %s (testnet=%v demo=%v) — Ctrl-C to stop ===\n\n",
		opt.Symbol, opt.Testnet, opt.Demo)

	dumpInitialState(ctx, sc, opt.Symbol)

	var (
		ordEvents atomic.Uint64
		execs     atomic.Uint64
		walletEv  atomic.Uint64
	)

	if err := sc.Stream().WatchOrders(ctx, func(o bybitspottypes.OrderInfo) {
		ordEvents.Add(1)
		fmt.Printf("%s  [ord]  ordId=%s clOrdId=%q %s %s tif=%s state=%s qty=%s leaves=%s avg=%s isLeverage=%v reject=%q\n",
			time.Now().Format("15:04:05.000"),
			o.OrderID, o.ClientOrderID, o.Side, o.OrderType, o.TimeInForce,
			o.Status, o.Quantity, o.LeavesQty, o.AvgPrice, o.IsLeverage, o.RejectReason)
	}, func(e error) { log.Printf("WatchOrders: %s", exhelp.Classify(e)) }); err != nil {
		log.Fatalf("WatchOrders: %s", exhelp.Classify(err))
	}

	if err := sc.Stream().WatchExecutions(ctx, func(e bybitspottypes.ExecutionInfo) {
		execs.Add(1)
		fmt.Printf("%s  [exec]  ordId=%s clOrdId=%q execId=%s %s qty=%s px=%s fee=%s %s maker=%v isLeverage=%v\n",
			time.Now().Format("15:04:05.000"),
			e.OrderID, e.ClientOrderID, e.ExecID, e.Side,
			e.ExecQty, e.ExecPrice, e.ExecFee, e.FeeCurrency, e.IsMaker, e.IsLeverage)
	}, func(e error) { log.Printf("WatchExecutions: %s", exhelp.Classify(e)) }); err != nil {
		log.Fatalf("WatchExecutions: %s", exhelp.Classify(err))
	}

	if err := sc.Stream().WatchWallet(ctx, func(b bybitspottypes.Balance) {
		walletEv.Add(1)
		fmt.Printf("%s  [wallet] type=%s equity=%s avail=%s im=%s mm=%s\n",
			time.Now().Format("15:04:05.000"),
			b.AccountType, b.TotalEquity, b.TotalAvailableBalance,
			b.TotalInitialMargin, b.TotalMaintenanceMargin)
	}, func(e error) { log.Printf("WatchWallet: %s", exhelp.Classify(e)) }); err != nil {
		log.Fatalf("WatchWallet: %s", exhelp.Classify(err))
	}

	fmt.Println("\nmonitoring... (Ctrl-C to stop)")
	fmt.Println()

	var sigCh chan os.Signal = make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\n=== Shutting down ===")
	fmt.Printf("orders     : %d\n", ordEvents.Load())
	fmt.Printf("executions : %d\n", execs.Load())
	fmt.Printf("wallet     : %d\n", walletEv.Load())
	_ = sc.Stream().Close()
}

func dumpInitialState(ctx context.Context, sc *spot.Client, symbol string) {
	var bal, balErr = sc.Account().GetWalletBalance(ctx, spot.WalletBalanceRequest{
		AccountType: bybitspottypes.AccountTypeUnified,
	})
	if balErr != nil {
		fmt.Printf("[init] wallet error: %s\n", exhelp.Classify(balErr))
	} else {
		fmt.Printf("[init] wallet equity=%s avail=%s\n",
			bal.TotalEquity, bal.TotalAvailableBalance)
		var i int
		for i = 0; i < len(bal.Coins); i++ {
			var c = bal.Coins[i]
			if c.WalletBalance.IsZero() {
				continue
			}
			fmt.Printf("[init]   %-6s wallet=%s available=%s usd=%s\n",
				c.Coin, c.WalletBalance, c.AvailableToWithdraw, c.UsdValue)
		}
	}

	var orders, ordErr = sc.Account().GetOpenOrders(ctx, symbol)
	if ordErr != nil {
		fmt.Printf("[init] open-orders error: %s\n", exhelp.Classify(ordErr))
	} else if len(orders) == 0 {
		fmt.Printf("[init] no open orders for %s\n", symbol)
	} else {
		fmt.Printf("[init] %d open order(s) for %s\n", len(orders), symbol)
	}
	fmt.Println()
}
