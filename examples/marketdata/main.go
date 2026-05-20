/*
FILE: examples/marketdata/main.go

DESCRIPTION:
Minimal end-to-end example: open a public bybit.Client, fetch the
BTCUSDT linear instrument metadata and a 50-level orderbook snapshot, and
print the top of book.

No API credentials are required for the calls in this example.

USAGE:

	go run ./examples/marketdata
*/

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	bybit "github.com/tonymontanov/go-bybit"
	"github.com/tonymontanov/go-bybit/linears"
)

func main() {
	var cfg bybit.Config = bybit.DefaultConfig()
	var client, err = bybit.NewClient(cfg)
	if err != nil {
		log.Fatalf("bybit.NewClient: %v", err)
	}
	defer client.Close()

	var lc *linears.Client = client.Linears().(*linears.Client)
	var ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var info, infoErr = lc.MarketData().GetSymbolInfo(ctx, "BTCUSDT")
	if infoErr != nil {
		log.Fatalf("GetSymbolInfo: %v", infoErr)
	}
	fmt.Printf("instrument: %s   tickSize=%s   qtyStep=%s   minQty=%s\n",
		info.Symbol, info.TickSize, info.QtyStep, info.MinOrderQty)

	var ob, obErr = lc.MarketData().GetOrderBook(ctx, "BTCUSDT", 50)
	if obErr != nil {
		log.Fatalf("GetOrderBook: %v", obErr)
	}
	if len(ob.Bids) == 0 || len(ob.Asks) == 0 {
		log.Fatalf("empty orderbook")
	}
	fmt.Printf("orderbook BTCUSDT: bid=%s @ %s   ask=%s @ %s   u=%d\n",
		ob.Bids[0].Price, ob.Bids[0].Size,
		ob.Asks[0].Price, ob.Asks[0].Size,
		ob.UpdateID)
}
