/*
FILE: spot/types/trade-update.go

DESCRIPTION:
Type alias re-export of the protocol-common
`github.com/tonymontanov/go-bybit/v2/types.TradeUpdate`. The wire
format of the publicTrade.{symbol} topic is identical across every
Bybit V5 category, so the spot profile reuses the common type.
*/

package types

import commontypes "github.com/tonymontanov/go-bybit/v2/types"

// TradeUpdate — one trade event from publicTrade. See commontypes.TradeUpdate.
type TradeUpdate = commontypes.TradeUpdate
