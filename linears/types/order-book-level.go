/*
FILE: linears/types/order-book-level.go

DESCRIPTION:
Type alias re-export of the protocol-common
`github.com/tonymontanov/go-bybit/v2/types.OrderBookLevel`. The wire
format is byte-identical across every Bybit V5 category, so the
linears profile reuses the common type rather than redefining it.
The alias preserves type identity — code that constructs
`linears/types.OrderBookLevel{...}` continues to compile unchanged.
*/

package types

import commontypes "github.com/tonymontanov/go-bybit/v2/types"

// OrderBookLevel — one order book level. See commontypes.OrderBookLevel.
type OrderBookLevel = commontypes.OrderBookLevel
