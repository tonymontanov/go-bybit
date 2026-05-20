/*
FILE: examples/stream-orderbook/main.go

DESCRIPTION:
WebSocket example: connect to the Bybit V5 public linear stream,
subscribe to the BTCUSDT order book at depth 50, and print the top of book
on every authoritative update.

The SDK keeps a local orderbook.Engine in sync; the user handler is only
called once a snapshot has been seeded and the most recent delta applied
cleanly. Sequence gaps trigger errHandler with a *bybit.Error of kind
ErrorKindInvalidRequest; the SDK then reseeds from the next snapshot
automatically.

USAGE:

	go run ./examples/stream-orderbook
*/

package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	bybit "github.com/tonymontanov/go-bybit"
	"github.com/tonymontanov/go-bybit/linears"
	"github.com/tonymontanov/go-bybit/linears/types"
)

func main() {
	var cfg bybit.Config = bybit.DefaultConfig()
	var client, err = bybit.NewClient(cfg)
	if err != nil {
		log.Fatalf("bybit.NewClient: %v", err)
	}
	defer client.Close()

	var lc *linears.Client = client.Linears().(*linears.Client)

	var ctx, stop = signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var watchErr = lc.Stream().WatchOrderBook(ctx, "BTCUSDT", 50, 5,
		func(ob types.OrderBookSnapshot) {
			if len(ob.Bids) == 0 || len(ob.Asks) == 0 {
				return
			}
			log.Printf("u=%d   bid=%s @ %s   ask=%s @ %s",
				ob.UpdateID,
				ob.Bids[0].Price, ob.Bids[0].Size,
				ob.Asks[0].Price, ob.Asks[0].Size,
			)
		},
		func(handlerErr error) {
			log.Printf("orderbook: %v", handlerErr)
		},
	)
	if watchErr != nil {
		log.Fatalf("WatchOrderBook: %v", watchErr)
	}

	<-ctx.Done()
	_ = lc.Stream().Close()
	log.Printf("shutting down")
}
