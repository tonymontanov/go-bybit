/*
FILE: spot/types/order-info.go

DESCRIPTION:
OrderInfo for the Bybit V5 spot category. Returned by Trading methods
(CreateOrder, ModifyOrder, CancelOrder, batch variants) and
Account.GetOpenOrders. Also dispatched through the WebSocket private
"order" topic.

DIFFERENCES vs LINEARS:
  - No `PositionIdx` / `ReduceOnly` (spot has no positions).
  - Adds `IsLeverage` (echo of the request flag for UTA margin spot).
  - Adds `MarketUnit` (echo of Market-order quantity interpretation).
  - All other fields share semantics with linears.

FIELDS:
  - OrderID         — Bybit "orderId".
  - ClientOrderID   — Bybit "orderLinkId".
  - Symbol          — e.g. "BTCUSDT".
  - Side            — Buy/Sell.
  - OrderType       — Limit/Market.
  - TimeInForce     — GTC/IOC/FOK/PostOnly. May be empty in synthetic
                      flows (e.g. ack of cancel-all).
  - Price           — limit price; decimal.Zero for market.
  - Quantity        — original order quantity (interpretation per
                      MarketUnit on Market orders).
  - LeavesQty       — quantity still live.
  - CumExecQty      — cumulative executed quantity.
  - CumExecValue    — cumulative executed quote value (Bybit
                      "cumExecValue"; empty/0 on Limit orders that
                      have not filled).
  - AvgPrice        — volume-weighted average fill price.
  - CumExecFee      — cumulative fee.
  - Status          — order status. See OrderStatus enum.
  - MarketUnit      — echo of request MarketUnit (empty when N/A).
  - IsLeverage      — echo of request IsLeverage flag.
  - RejectReason    — see normalisation note in linears/types/order-info.go.
  - CreatedAtMs     — order creation timestamp (ms).
  - UpdatedAtMs     — last server-side update timestamp.
  - RateLimits      — snapshot of rate-limit headers (REST only).
*/

package types

import "github.com/shopspring/decimal"

// OrderInfo — order state for the spot category.
type OrderInfo struct {
	OrderID       string
	ClientOrderID string
	Symbol        string
	Side          SideType
	OrderType     OrderType
	TimeInForce   TimeInForceType
	Price         decimal.Decimal
	Quantity      decimal.Decimal
	LeavesQty     decimal.Decimal
	CumExecQty    decimal.Decimal
	CumExecValue  decimal.Decimal
	AvgPrice      decimal.Decimal
	CumExecFee    decimal.Decimal
	Status        OrderStatus
	MarketUnit    MarketUnit
	IsLeverage    bool
	RejectReason  string
	CreatedAtMs   int64
	UpdatedAtMs   int64
	RateLimits    map[string]string
}
