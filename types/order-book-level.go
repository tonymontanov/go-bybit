/*
FILE: types/order-book-level.go

DESCRIPTION:
A single order book level — protocol-common across every Bybit V5
category. Used by:
  - REST snapshot (GET /v5/market/orderbook);
  - the SDK orderbook engine (snapshot/delta application);
  - WebSocket "orderbook.{depth}.{symbol}" topic dispatch.

Bybit V5 represents a level as a positional [price, size] pair of
strings: ["27045.00", "0.123"]. The SDK normalises both parts into
decimal.Decimal at the boundary.
*/

package types

import "github.com/shopspring/decimal"

// OrderBookLevel — one order book level.
type OrderBookLevel struct {
	Price decimal.Decimal
	Size  decimal.Decimal
}
