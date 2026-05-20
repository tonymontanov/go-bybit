/*
FILE: examples/inventory-monitor/main.go

DESCRIPTION:
Long-running monitor for the user's own state on Bybit. Subscribes to
the private "order", "position", "execution" and "wallet" topics and
prints every update in real time. Does NOT place trades — read-only.

This is the right shape for an algorithm that must keep its picture of
the exchange state warm even when no orders are flowing.

COVERAGE:
  - linears.Stream().WatchOrders        (own orders, filtered to linear)
  - linears.Stream().WatchPositions     (own positions, linear)
  - linears.Stream().WatchExecutions    (own fills, linear)
  - linears.Stream().WatchWallet        (own wallet)
  - initial REST snapshots: GetWalletBalance + GetPosition + GetOpenOrders.

USAGE:

	./scripts/run.sh ./examples/inventory-monitor
	BYBIT_SYMBOL=ETHUSDT ./scripts/run.sh ./examples/inventory-monitor

To verify the monitor stays alive on pos==0:
  1. start it;
  2. open and close a small position from the Bybit web UI;
  3. updates flow through, then quietly resume — no exit.
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
	"github.com/tonymontanov/go-bybit/linears"
	"github.com/tonymontanov/go-bybit/linears/types"
)

func main() {
	var opt exhelp.Options = exhelp.LoadEnv()
	exhelp.MustHaveKeys(opt)

	var client, lc = exhelp.NewClient(opt)
	defer client.Close()

	var ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	fmt.Printf("=== inventory-monitor %s (testnet=%v demo=%v) — Ctrl-C to stop ===\n\n",
		opt.Symbol, opt.Testnet, opt.Demo)

	dumpInitialState(ctx, lc, opt.Symbol)

	var (
		ordEvents atomic.Uint64
		posEvents atomic.Uint64
		execs     atomic.Uint64
		walletEv  atomic.Uint64
	)

	if err := lc.Stream().WatchOrders(ctx, func(o types.OrderInfo) {
		ordEvents.Add(1)
		fmt.Printf("%s  [ord]  ordId=%s clOrdId=%q %s %s tif=%s state=%s qty=%s leaves=%s avg=%s reject=%q\n",
			time.Now().Format("15:04:05.000"),
			o.OrderID, o.ClientOrderID, o.Side, o.OrderType, o.TimeInForce,
			o.Status, o.Quantity, o.LeavesQty, o.AvgPrice, o.RejectReason)
	}, func(e error) { log.Printf("WatchOrders: %s", exhelp.Classify(e)) }); err != nil {
		log.Fatalf("WatchOrders: %s", exhelp.Classify(err))
	}

	if err := lc.Stream().WatchPositions(ctx, func(p types.PositionInfo) {
		posEvents.Add(1)
		fmt.Printf("%s  [pos]  %s qty=%s avgPx=%s mark=%s liqPx=%s lev=%s uPnL=%s realized=%s\n",
			time.Now().Format("15:04:05.000"),
			p.Side, p.Quantity, p.AvgEntryPrice, p.MarkPrice,
			p.LiqPrice, p.Leverage, p.UnrealizedPnL, p.RealizedPnL)
	}, func(e error) { log.Printf("WatchPositions: %s", exhelp.Classify(e)) }); err != nil {
		log.Fatalf("WatchPositions: %s", exhelp.Classify(err))
	}

	if err := lc.Stream().WatchExecutions(ctx, func(e types.ExecutionInfo) {
		execs.Add(1)
		fmt.Printf("%s  [exec]  ordId=%s clOrdId=%q execId=%s %s qty=%s px=%s fee=%s %s maker=%v\n",
			time.Now().Format("15:04:05.000"),
			e.OrderID, e.ClientOrderID, e.ExecID, e.Side,
			e.ExecQty, e.ExecPrice, e.ExecFee, e.FeeCurrency, e.IsMaker)
	}, func(e error) { log.Printf("WatchExecutions: %s", exhelp.Classify(e)) }); err != nil {
		log.Fatalf("WatchExecutions: %s", exhelp.Classify(err))
	}

	if err := lc.Stream().WatchWallet(ctx, func(b types.Balance) {
		walletEv.Add(1)
		fmt.Printf("%s  [wallet] type=%s equity=%s avail=%s im=%s mm=%s upl=%s\n",
			time.Now().Format("15:04:05.000"),
			b.AccountType, b.TotalEquity, b.TotalAvailableBalance,
			b.TotalInitialMargin, b.TotalMaintenanceMargin, b.TotalPerpUPL)
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
	fmt.Printf("positions  : %d\n", posEvents.Load())
	fmt.Printf("executions : %d\n", execs.Load())
	fmt.Printf("wallet     : %d\n", walletEv.Load())
	_ = lc.Stream().Close()
}

func dumpInitialState(ctx context.Context, lc *linears.Client, symbol string) {
	var bal, balErr = lc.Account().GetWalletBalance(ctx, linears.WalletBalanceRequest{
		AccountType: types.AccountTypeUnified,
	})
	if balErr != nil {
		fmt.Printf("[init] wallet error: %s\n", exhelp.Classify(balErr))
	} else {
		fmt.Printf("[init] wallet equity=%s avail=%s im=%s mm=%s upl=%s\n",
			bal.TotalEquity, bal.TotalAvailableBalance,
			bal.TotalInitialMargin, bal.TotalMaintenanceMargin, bal.TotalPerpUPL)
	}

	var pos, posErr = lc.Account().GetPosition(ctx, symbol)
	if posErr != nil {
		fmt.Printf("[init] position error: %s\n", exhelp.Classify(posErr))
	} else if len(pos) == 0 {
		fmt.Printf("[init] no position rows for %s\n", symbol)
	} else {
		var i int
		for i = 0; i < len(pos); i++ {
			var p = pos[i]
			fmt.Printf("[init] pos idx=%d %s qty=%s avgPx=%s mark=%s upl=%s liq=%s\n",
				p.PositionIdx, p.Side, p.Quantity, p.AvgEntryPrice,
				p.MarkPrice, p.UnrealizedPnL, p.LiqPrice)
		}
	}

	var orders, ordErr = lc.Account().GetOpenOrders(ctx, symbol)
	if ordErr != nil {
		fmt.Printf("[init] open-orders error: %s\n", exhelp.Classify(ordErr))
	} else if len(orders) == 0 {
		fmt.Printf("[init] no open orders for %s\n", symbol)
	} else {
		fmt.Printf("[init] %d open order(s) for %s\n", len(orders), symbol)
	}
	fmt.Println()
}
