/*
FILE: spot/types/order-book-level.go

DESCRIPTION:
One order book level for the Bybit V5 spot category. Wire format and
shape are byte-identical to linears: positional [price, size] pair of
strings normalised to `decimal.Decimal` at the SDK boundary.
*/

package types

import "github.com/shopspring/decimal"

// OrderBookLevel — one order book level.
type OrderBookLevel struct {
	Price decimal.Decimal
	Size  decimal.Decimal
}
