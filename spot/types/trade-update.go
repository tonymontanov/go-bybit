/*
FILE: spot/types/trade-update.go

DESCRIPTION:
TradeUpdate for the Bybit V5 spot WebSocket "publicTrade.{symbol}"
topic. Wire shape and semantics are identical to linears (Bybit ships
trade frames in batches; the SDK fans them out so handlers receive one
TradeUpdate per call).
*/

package types

import "github.com/shopspring/decimal"

// TradeUpdate — one trade event from the publicTrade topic.
type TradeUpdate struct {
	Symbol     string
	Price      decimal.Decimal
	Size       decimal.Decimal
	Side       SideType
	TradeID    string
	TsMs       int64
	BlockTrade bool
}
