/*
FILE: linears/types/order-info.go

DESCRIPTION:
OrderInfo is returned by Trading methods (CreateOrder, ModifyOrder,
CancelOrder, batch variants) and by Account.GetOpenOrders. The same
struct is also dispatched through the WebSocket private "order" topic
once the Stream subclient is implemented (M3) — values come from the
GET /v5/order/realtime / WS topic and are normalized into Go types here.

FIELDS:
  - OrderID         — Bybit "orderId".
  - ClientOrderID   — Bybit "orderLinkId".
  - Symbol          — symbol (e.g. "BTCUSDT").
  - Side            — Buy/Sell.
  - OrderType       — Limit/Market.
  - TimeInForce     — GTC/IOC/FOK/PostOnly. May be empty in synthetic
                      flows where Bybit omits it (e.g. ack of cancel-all).
  - Price           — limit price. decimal.Zero for market orders.
  - Quantity        — original order quantity in base asset.
  - LeavesQty       — quantity still live (not filled or cancelled).
  - CumExecQty      — cumulative executed quantity.
  - AvgPrice        — volume-weighted average fill price.
  - CumExecFee      — cumulative fee in quote (USDT/USDC).
  - Status          — order status. See OrderStatus enum for the canonical
                      subset; unknown statuses are surfaced verbatim.
  - PositionIdx     — 0/1/2 — copy of the request value.
  - ReduceOnly      — reduceOnly flag echo.
  - RejectReason    — Bybit rejection reason. The SDK normalizes Bybit's
                      "EC_NoError" sentinel (used on every non-rejected
                      order) to the empty string so callers can branch
                      on `RejectReason != ""`. Other reason codes
                      (e.g. "EC_PostOnlyWillTakeLiquidity") are
                      surfaced verbatim.
  - CreatedAtMs     — order creation timestamp (ms since epoch).
  - UpdatedAtMs     — last server-side update timestamp.
  - RateLimits      — snapshot of rate-limit headers received with the
                      response. Populated by Trading methods that call
                      REST; nil for events that came via WebSocket.

NOTE:
  - Bybit returns timestamps as decimal strings (e.g. "1684738540000")
    in REST and as int64 in WS pushes. Both are normalized to int64 ms
    here.
  - All decimal values are stored as decimal.Decimal — lossless and
    comparable without epsilon tricks.
*/

package types

import "github.com/shopspring/decimal"

// OrderInfo — order state for the linear category.
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
	AvgPrice      decimal.Decimal
	CumExecFee    decimal.Decimal
	Status        OrderStatus
	PositionIdx   PositionIdx
	ReduceOnly    bool
	RejectReason  string
	CreatedAtMs   int64
	UpdatedAtMs   int64
	RateLimits    map[string]string
}
