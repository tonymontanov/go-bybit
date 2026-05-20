/*
FILE: linears/types/create-order-request.go

DESCRIPTION:
Order creation request for the Bybit V5 linear category. The struct is
shaped to mirror the desk's connectors/types.CreateOrderRequest so that
the desk-side adapter can hand off without copying every field; Bybit
specifics (PositionIdx, ReduceOnly, CloseOnTrigger) are exposed as
optional fields.

FIELDS:
  - Symbol         — Bybit symbol, e.g. "BTCUSDT".
  - Side           — Buy/Sell.
  - OrderType      — Limit/Market. If empty the SDK falls back to the
                     OrderType derived from TimeInForce (PostOnly →
                     Limit, otherwise Limit).
  - TimeInForce    — GTC/IOC/FOK/PostOnly. PostOnly is rewritten to
                     timeInForce=PostOnly + orderType=Limit on the wire.
                     Empty → GTC for Limit, IOC for Market (Bybit default).
  - Quantity       — order quantity in BASE asset (BTC, ETH, ...). Bybit
                     V5 linear quotes quantities directly in base asset,
                     so no contract-multiplier conversion is needed.
  - Price          — limit price. Required for Limit/PostOnly, ignored
                     for Market.
  - ClientOrderID  — Bybit "orderLinkId" (≤ 36 chars, [A-Za-z0-9_.-]).
                     Optional but strongly recommended for trader-side
                     idempotency / observability.
  - PositionIdx    — 0 (one-way, default) / 1 (hedge buy) / 2 (hedge sell).
                     Caller must set it correctly for hedge accounts;
                     wrong value yields retCode 110017.
  - ReduceOnly     — reduceOnly flag.
  - CloseOnTrigger — closeOnTrigger flag (used by liquidation flows).

INVARIANTS:
  - Quantity > 0 always; the SDK validates and surfaces InvalidRequest
    locally, before contacting Bybit.
  - For OrderType==Limit (or TimeInForce==PostOnly) Price must be > 0.
*/

package types

import "github.com/shopspring/decimal"

// CreateOrderRequest — order creation request for the linear category.
type CreateOrderRequest struct {
	Symbol         string
	Side           SideType
	OrderType      OrderType
	TimeInForce    TimeInForceType
	Quantity       decimal.Decimal
	Price          decimal.Decimal
	ClientOrderID  string
	PositionIdx    PositionIdx
	ReduceOnly     bool
	CloseOnTrigger bool
}
