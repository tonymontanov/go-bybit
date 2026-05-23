/*
FILE: spot/types/order-book-snapshot.go

DESCRIPTION:
Type alias re-export of the protocol-common
`github.com/tonymontanov/go-bybit/v2/types.OrderBookSnapshot`. Wire
shape and synchronisation model are identical for every Bybit V5
category. The alias preserves type identity — existing code that
addresses `spot/types.OrderBookSnapshot` continues to compile
unchanged.
*/

package types

import commontypes "github.com/tonymontanov/go-bybit/v2/types"

// OrderBookSnapshot — order book snapshot. See commontypes.OrderBookSnapshot.
type OrderBookSnapshot = commontypes.OrderBookSnapshot
