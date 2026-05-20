/*
FILE: linears/types/trade-update.go

DESCRIPTION:
TradeUpdate is one element of the "publicTrade.{symbol}" WebSocket topic.
Bybit ships trade frames in batches; the dispatcher fans them out so
handlers receive one TradeUpdate per call.

FIELDS:
  - Symbol     : Bybit symbol.
  - Price      : trade price.
  - Size       : trade size in base asset.
  - Side       : taker side (Buy = aggressor bought, Sell = aggressor sold).
  - TradeID    : Bybit trade id ("i" field).
  - TsMs       : trade match timestamp (ms).
  - BlockTrade : true for block trades (BT=true on the wire).
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
