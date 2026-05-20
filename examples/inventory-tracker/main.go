/*
FILE: examples/inventory-tracker/main.go

DESCRIPTION:
WARNING: this example PLACES A REAL TRADE on the exchange. It:

  1. subscribes to the private "order", "position", "execution" and "wallet"
     WS topics;
  2. prints the current REST snapshot;
  3. sends a small MARKET-BUY (qty defaults to 0.001 BTCUSDT);
  4. shows the live inventory updates flowing in;
  5. after BYBIT_HOLD_SECONDS holds the position open;
  6. closes the position via Account.ClosePosition (market reduce-only);
  7. waits a few seconds for the close updates and prints the final state.

CONFIG:
  - BYBIT_SYMBOL          (default BTCUSDT)
  - BYBIT_QUANTITY        (default 0.001 BTC)
  - BYBIT_HOLD_SECONDS    (default 5)
  - BYBIT_TESTNET / BYBIT_DEMO    pick a safe environment for the FIRST run.
  - BYBIT_ALLOW_LIVE=1    required against PRODUCTION (bypassed if Testnet/Demo).

COST:
  market BUY + ClosePosition → spread×2 + fee×2. For BTCUSDT 0.001 at $60k
  this is ~ $0.10 in fees on production. Trade SMALL.

USAGE:

	# 1) Bybit Testnet — recommended for first run.
	BYBIT_TESTNET=1 ./scripts/run.sh ./examples/inventory-tracker
	# 2) Production — explicit opt-in.
	BYBIT_ALLOW_LIVE=1 ./scripts/run.sh ./examples/inventory-tracker
*/

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/shopspring/decimal"

	"github.com/tonymontanov/go-bybit/examples/internal/exhelp"
	"github.com/tonymontanov/go-bybit/linears"
	"github.com/tonymontanov/go-bybit/linears/types"
)

func main() {
	var opt exhelp.Options = exhelp.LoadEnv()
	exhelp.MustHaveKeys(opt)
	exhelp.MustAllowLive(opt)

	var client, lc = exhelp.NewClient(opt)
	defer client.Close()

	var ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	var sigCh chan os.Signal = make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("interrupt — cancelling")
		cancel()
	}()

	fmt.Printf("=== inventory-tracker %s qty=%s hold=%ds (testnet=%v demo=%v) ===\n\n",
		opt.Symbol, opt.Quantity, opt.HoldSeconds, opt.Testnet, opt.Demo)

	dumpInitialState(ctx, lc, opt.Symbol)

	var inv inventoryState
	subscribeOrders(ctx, lc, &inv)
	subscribePositions(ctx, lc, &inv)
	subscribeExecutions(ctx, lc, &inv)
	subscribeWallet(ctx, lc, &inv)

	fmt.Println("warming up streams...")
	select {
	case <-time.After(2 * time.Second):
	case <-ctx.Done():
		return
	}

	var clOrdID string = "inv" + fmt.Sprint(time.Now().UnixMilli())
	fmt.Printf("\n>>> placing MARKET BUY %s qty=%s clOrdId=%s\n", opt.Symbol, opt.Quantity, clOrdID)
	var openOrder, openErr = lc.Trading().CreateOrder(ctx, types.CreateOrderRequest{
		Symbol:        opt.Symbol,
		Side:          types.SideTypeBuy,
		OrderType:     types.OrderTypeMarket,
		Quantity:      opt.Quantity,
		ClientOrderID: clOrdID,
	})
	if openErr != nil {
		log.Fatalf("CreateOrder(market buy): %s", exhelp.Classify(openErr))
	}
	fmt.Printf(">>> placed: ordId=%s clOrdId=%s\n", openOrder.OrderID, openOrder.ClientOrderID)

	fmt.Printf(">>> holding for %ds, streaming inventory updates...\n\n", opt.HoldSeconds)
	var hold *time.Timer = time.NewTimer(time.Duration(opt.HoldSeconds) * time.Second)
	select {
	case <-hold.C:
	case <-ctx.Done():
		log.Println("ctx done during hold")
		return
	}

	fmt.Println("\n>>> closing position")
	var closeOrder, closeErr = lc.Account().ClosePosition(ctx, opt.Symbol, types.PositionIdxOneWay)
	if closeErr != nil {
		log.Fatalf("ClosePosition: %s", exhelp.Classify(closeErr))
	}
	if closeOrder.OrderID == "" {
		fmt.Println(">>> no live position to close (already flat)")
	} else {
		fmt.Printf(">>> close request accepted: ordId=%s\n", closeOrder.OrderID)
	}

	select {
	case <-time.After(3 * time.Second):
	case <-ctx.Done():
		return
	}

	fmt.Println("\n=== Final state ===")
	fmt.Printf("order events     : %d\n", inv.ordEvents.Load())
	fmt.Printf("position events  : %d\n", inv.posEvents.Load())
	fmt.Printf("execution events : %d\n", inv.execEvents.Load())
	fmt.Printf("wallet events    : %d\n", inv.walletEvents.Load())

	var finalPos, ferr = lc.Account().GetPosition(context.Background(), opt.Symbol)
	if ferr != nil {
		fmt.Printf("final position fetch error: %s\n", exhelp.Classify(ferr))
		return
	}
	if len(finalPos) == 0 || allEmpty(finalPos) {
		fmt.Println("final position: FLAT (good — closed cleanly)")
		return
	}
	var i int
	for i = 0; i < len(finalPos); i++ {
		if finalPos[i].Quantity.IsZero() {
			continue
		}
		fmt.Printf("final position: %s qty=%s avgPx=%s upl=%s (LEFTOVER — close manually!)\n",
			finalPos[i].Side, finalPos[i].Quantity, finalPos[i].AvgEntryPrice, finalPos[i].UnrealizedPnL)
	}
}

type inventoryState struct {
	ordEvents    atomic.Uint64
	posEvents    atomic.Uint64
	execEvents   atomic.Uint64
	walletEvents atomic.Uint64

	mu        sync.RWMutex
	lastQty   decimal.Decimal
	lastAvgPx decimal.Decimal
}

func dumpInitialState(ctx context.Context, lc *linears.Client, symbol string) {
	var bal, balErr = lc.Account().GetWalletBalance(ctx, linears.WalletBalanceRequest{
		AccountType: types.AccountTypeUnified,
	})
	if balErr != nil {
		fmt.Printf("[init] wallet error: %s\n", exhelp.Classify(balErr))
	} else {
		fmt.Printf("[init] wallet equity=%s avail=%s im=%s upl=%s\n",
			bal.TotalEquity, bal.TotalAvailableBalance, bal.TotalInitialMargin, bal.TotalPerpUPL)
	}

	var pos, posErr = lc.Account().GetPosition(ctx, symbol)
	if posErr != nil {
		fmt.Printf("[init] position error: %s\n", exhelp.Classify(posErr))
		return
	}
	if len(pos) == 0 || allEmpty(pos) {
		fmt.Printf("[init] position: FLAT\n\n")
		return
	}
	var i int
	for i = 0; i < len(pos); i++ {
		if pos[i].Quantity.IsZero() {
			continue
		}
		fmt.Printf("[init] pos %s qty=%s avgPx=%s upl=%s\n",
			pos[i].Side, pos[i].Quantity, pos[i].AvgEntryPrice, pos[i].UnrealizedPnL)
	}
	fmt.Println()
}

func subscribeOrders(ctx context.Context, lc *linears.Client, inv *inventoryState) {
	var err = lc.Stream().WatchOrders(ctx, func(o types.OrderInfo) {
		inv.ordEvents.Add(1)
		fmt.Printf("  [ord]  ordId=%s clOrdId=%q %s %s state=%s qty=%s leaves=%s avg=%s\n",
			o.OrderID, o.ClientOrderID, o.Side, o.OrderType, o.Status,
			o.Quantity, o.LeavesQty, o.AvgPrice)
	}, func(e error) { log.Printf("WatchOrders: %s", exhelp.Classify(e)) })
	if err != nil {
		log.Fatalf("WatchOrders: %s", exhelp.Classify(err))
	}
}

func subscribePositions(ctx context.Context, lc *linears.Client, inv *inventoryState) {
	var err = lc.Stream().WatchPositions(ctx, func(p types.PositionInfo) {
		inv.posEvents.Add(1)
		inv.mu.Lock()
		inv.lastQty = p.Quantity
		inv.lastAvgPx = p.AvgEntryPrice
		inv.mu.Unlock()
		fmt.Printf("  [pos]  %s qty=%s avgPx=%s mark=%s liqPx=%s upl=%s realized=%s\n",
			p.Side, p.Quantity, p.AvgEntryPrice, p.MarkPrice,
			p.LiqPrice, p.UnrealizedPnL, p.RealizedPnL)
	}, func(e error) { log.Printf("WatchPositions: %s", exhelp.Classify(e)) })
	if err != nil {
		log.Fatalf("WatchPositions: %s", exhelp.Classify(err))
	}
}

func subscribeExecutions(ctx context.Context, lc *linears.Client, inv *inventoryState) {
	var err = lc.Stream().WatchExecutions(ctx, func(e types.ExecutionInfo) {
		inv.execEvents.Add(1)
		fmt.Printf("  [exec] ordId=%s execId=%s %s qty=%s px=%s fee=%s %s maker=%v\n",
			e.OrderID, e.ExecID, e.Side, e.ExecQty, e.ExecPrice,
			e.ExecFee, e.FeeCurrency, e.IsMaker)
	}, func(e error) { log.Printf("WatchExecutions: %s", exhelp.Classify(e)) })
	if err != nil {
		log.Fatalf("WatchExecutions: %s", exhelp.Classify(err))
	}
}

func subscribeWallet(ctx context.Context, lc *linears.Client, inv *inventoryState) {
	var err = lc.Stream().WatchWallet(ctx, func(b types.Balance) {
		inv.walletEvents.Add(1)
		fmt.Printf("  [wallet] type=%s equity=%s avail=%s im=%s upl=%s\n",
			b.AccountType, b.TotalEquity, b.TotalAvailableBalance,
			b.TotalInitialMargin, b.TotalPerpUPL)
	}, func(e error) { log.Printf("WatchWallet: %s", exhelp.Classify(e)) })
	if err != nil {
		log.Fatalf("WatchWallet: %s", exhelp.Classify(err))
	}
}

func allEmpty(rows []types.PositionInfo) bool {
	var i int
	for i = 0; i < len(rows); i++ {
		if !rows[i].Quantity.IsZero() {
			return false
		}
	}
	return true
}
