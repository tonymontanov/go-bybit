/*
FILE: examples/simple-trade/main.go

DESCRIPTION:
Minimal REST trading scenario:

  1. read keys from .env (BYBIT_API_KEY / BYBIT_API_SECRET);
  2. fetch SymbolInfo and the current top-of-book to compute a safe far-from-mid
     PostOnly limit price (rounded to TickSize);
  3. place a small PostOnly limit order;
  4. modify its quantity;
  5. cancel.

WARNINGS:
  - Will refuse to run against PRODUCTION unless BYBIT_ALLOW_LIVE=1.
    Use BYBIT_TESTNET=1 to bypass that check (recommended for first runs).
  - The order is PostOnly far from mid → it should NOT fill in normal
    market conditions, but a Bybit outage / strange book can theoretically
    pull it. Trade SMALL.

USAGE:

	./scripts/run.sh ./examples/simple-trade
*/

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/shopspring/decimal"

	"github.com/tonymontanov/go-bybit/examples/internal/exhelp"
	"github.com/tonymontanov/go-bybit/linears/types"
)

func main() {
	var opt exhelp.Options = exhelp.LoadEnv()
	exhelp.MustHaveKeys(opt)
	exhelp.MustAllowLive(opt)

	var client, lc = exhelp.NewClient(opt)
	defer client.Close()

	var ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Printf("=== simple-trade %s qty=%s (testnet=%v demo=%v) ===\n\n",
		opt.Symbol, opt.Quantity, opt.Testnet, opt.Demo)

	var info, infoErr = lc.MarketData().GetSymbolInfo(ctx, opt.Symbol)
	if infoErr != nil {
		log.Fatalf("GetSymbolInfo: %s", exhelp.Classify(infoErr))
	}

	var ob, obErr = lc.MarketData().GetOrderBook(ctx, opt.Symbol, 1)
	if obErr != nil {
		log.Fatalf("GetOrderBook: %s", exhelp.Classify(obErr))
	}
	if len(ob.Bids) == 0 {
		log.Fatalf("empty bids; cannot price safely")
	}
	var bestBid decimal.Decimal = ob.Bids[0].Price
	// PostOnly buy at -10% of the best bid, rounded down to TickSize.
	// At -10% the order can never cross the spread.
	var rawPrice decimal.Decimal = bestBid.Mul(decimal.RequireFromString("0.90"))
	var safePrice decimal.Decimal = quantize(rawPrice, info.TickSize)
	if safePrice.IsZero() || safePrice.IsNegative() {
		log.Fatalf("computed price <= 0: %s", safePrice)
	}
	var clOrdID string = "ex" + fmt.Sprint(time.Now().UnixMilli())

	fmt.Printf("placing PostOnly buy: qty=%s price=%s (bid=%s tick=%s) clOrdId=%s\n",
		opt.Quantity, safePrice, bestBid, info.TickSize, clOrdID)

	var info1, createErr = lc.Trading().CreateOrder(ctx, types.CreateOrderRequest{
		Symbol:        opt.Symbol,
		Side:          types.SideTypeBuy,
		OrderType:     types.OrderTypeLimit,
		TimeInForce:   types.TimeInForcePostOnly,
		Quantity:      opt.Quantity,
		Price:         safePrice,
		ClientOrderID: clOrdID,
	})
	if createErr != nil {
		log.Fatalf("CreateOrder: %s", exhelp.Classify(createErr))
	}
	fmt.Printf("placed: orderId=%s clOrdId=%s rate-limit=%v\n",
		info1.OrderID, info1.ClientOrderID, info1.RateLimits)

	var newQty decimal.Decimal = quantize(opt.Quantity.Mul(decimal.NewFromInt(2)), info.QtyStep)
	fmt.Printf("amending qty → %s\n", newQty)
	var info2, modErr = lc.Trading().ModifyOrder(ctx, types.ModifyOrderRequest{
		Symbol:      opt.Symbol,
		OrderID:     info1.OrderID,
		NewQuantity: newQty,
	})
	if modErr != nil {
		log.Printf("ModifyOrder: %s", exhelp.Classify(modErr))
	} else {
		fmt.Printf("modified: orderId=%s qty=%s price=%s\n",
			info2.OrderID, info2.Quantity, info2.Price)
	}

	fmt.Println("cancelling")
	if err := lc.Trading().CancelOrder(ctx, types.CancelOrderRequest{
		Symbol:  opt.Symbol,
		OrderID: info1.OrderID,
	}); err != nil {
		log.Fatalf("CancelOrder: %s", exhelp.Classify(err))
	}
	fmt.Println("cancelled OK")
}

// quantize rounds value DOWN to the nearest multiple of step. If step is
// zero, the value is returned unchanged. The result is guaranteed to be
// representable on the wire (no surplus decimals).
func quantize(value, step decimal.Decimal) decimal.Decimal {
	if step.IsZero() {
		return value
	}
	var rounded = value.Div(step).Floor().Mul(step)
	return rounded.Truncate(precisionOf(step))
}

// precisionOf returns the number of decimal places in v (e.g. "0.001" → 3).
func precisionOf(v decimal.Decimal) int32 {
	var s string = v.String()
	var i int = -1
	var n int = len(s)
	var k int
	for k = 0; k < n; k++ {
		if s[k] == '.' {
			i = k
			break
		}
	}
	if i < 0 {
		return 0
	}
	return int32(n - i - 1)
}
