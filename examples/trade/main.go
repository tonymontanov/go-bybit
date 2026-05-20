/*
FILE: examples/trade/main.go

DESCRIPTION:
Signed trading example: place a far-from-market PostOnly limit and then
immediately cancel it. Demonstrates auth wiring, OrderInfo handling and
the typed *bberr.Error workflow.

REQUIREMENTS:
  - BYBIT_API_KEY and BYBIT_API_SECRET env vars (Bybit V5 keys with linear
    trade permission).
  - Either Bybit testnet (cfg.Testnet = true) or a tiny limit far from the
    book; this example uses a $1 limit on BTCUSDT, which Bybit accepts as
    PostOnly and never fills in normal market conditions.

USAGE:

	BYBIT_API_KEY=... BYBIT_API_SECRET=... \
	    go run ./examples/trade
*/

package main

import (
	"context"
	"errors"
	"log"
	"os"
	"time"

	"github.com/shopspring/decimal"

	bybit "github.com/tonymontanov/go-bybit"
	"github.com/tonymontanov/go-bybit/linears"
	"github.com/tonymontanov/go-bybit/linears/types"
)

func main() {
	var key string = os.Getenv("BYBIT_API_KEY")
	var secret string = os.Getenv("BYBIT_API_SECRET")
	if key == "" || secret == "" {
		log.Fatalf("BYBIT_API_KEY and BYBIT_API_SECRET must be set")
	}

	var cfg bybit.Config = bybit.DefaultConfig()
	cfg.APIKey = key
	cfg.SecretKey = secret
	// Uncomment to use the public testnet:
	// cfg.Testnet = true

	var client, err = bybit.NewClient(cfg)
	if err != nil {
		log.Fatalf("bybit.NewClient: %v", err)
	}
	defer client.Close()

	var lc *linears.Client = client.Linears().(*linears.Client)
	var ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var order, createErr = lc.Trading().CreateOrder(ctx, types.CreateOrderRequest{
		Symbol:        "BTCUSDT",
		Side:          types.SideTypeBuy,
		OrderType:     types.OrderTypeLimit,
		TimeInForce:   types.TimeInForcePostOnly,
		Quantity:      decimal.RequireFromString("0.001"),
		Price:         decimal.RequireFromString("1"),
		ClientOrderID: "go-bybit-example-1",
	})
	if createErr != nil {
		var be *bybit.Error
		if errors.As(createErr, &be) && be.Kind == bybit.ErrorKindAuth {
			log.Fatalf("CreateOrder rejected as auth error: %v", be)
		}
		log.Fatalf("CreateOrder: %v", createErr)
	}
	log.Printf("created: orderId=%s clientOrderId=%s", order.OrderID, order.ClientOrderID)

	if cancelErr := lc.Trading().CancelOrder(ctx, types.CancelOrderRequest{
		Symbol:        "BTCUSDT",
		ClientOrderID: order.ClientOrderID,
	}); cancelErr != nil {
		log.Fatalf("CancelOrder: %v", cancelErr)
	}
	log.Printf("cancelled: orderId=%s clientOrderId=%s", order.OrderID, order.ClientOrderID)
}
