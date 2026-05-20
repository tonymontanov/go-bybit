/*
FILE: linears/types/execution-info.go

DESCRIPTION:
ExecutionInfo is one fill event from the private "execution" WebSocket
topic. Each event represents a single match against a maker; partial
fills generate multiple events for the same OrderID.

FIELDS:
  - Symbol         : Bybit symbol.
  - OrderID        : Bybit order id of the order that filled.
  - ClientOrderID  : orderLinkId of the same order.
  - ExecID         : exchange-side trade id (unique within a fill).
  - Side           : taker side from the perspective of the order owner.
                     Buy means the owner bought; Sell means the owner sold.
  - ExecPrice      : execution price.
  - ExecQty        : executed quantity (base asset).
  - ExecValue      : executed value in quote (= price × qty).
  - ExecFee        : fee paid (in FeeCurrency).
  - FeeCurrency    : currency in which the fee was charged.
  - IsMaker        : true when the owner's order was the maker.
  - PositionIdx    : 0 (one-way) / 1 (hedge buy) / 2 (hedge sell).
  - ExecTimeMs     : exchange-side execution timestamp (ms).
*/

package types

import "github.com/shopspring/decimal"

// ExecutionInfo — one fill event for the linear category.
type ExecutionInfo struct {
	Symbol        string
	OrderID       string
	ClientOrderID string
	ExecID        string
	Side          SideType
	ExecPrice     decimal.Decimal
	ExecQty       decimal.Decimal
	ExecValue     decimal.Decimal
	ExecFee       decimal.Decimal
	FeeCurrency   string
	IsMaker       bool
	PositionIdx   PositionIdx
	ExecTimeMs    int64
}
