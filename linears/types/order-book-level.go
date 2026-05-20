/*
FILE: linears/types/order-book-level.go

DESCRIPTION:
A single order book level. Used in:
  - REST snapshot (GetOrderBook);
  - orderbook engine snapshot/delta application (M2);
  - WebSocket "orderbook.{depth}.{symbol}" topic dispatch (M3).

Bybit V5 represents a level as a positional [price, size] pair of
strings: ["27045.00", "0.123"]. The SDK normalizes both parts into
decimal.Decimal at the boundary.
*/

package types

import "github.com/shopspring/decimal"

// OrderBookLevel — one order book level.
type OrderBookLevel struct {
	Price decimal.Decimal
	Size  decimal.Decimal
}
