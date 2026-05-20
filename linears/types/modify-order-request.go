/*
FILE: linears/types/modify-order-request.go

DESCRIPTION:
Order amend request for Bybit V5 linear. Bybit's POST /v5/order/amend
allows changing qty / price / takeProfit / stopLoss only — side, type
and timeInForce CANNOT be amended; for that the caller must cancel and
re-create the order.

FIELDS:
  - Symbol         — Bybit symbol.
  - OrderID        — exchange order id. Either OrderID or ClientOrderID
                     must be set.
  - ClientOrderID  — orderLinkId.
  - NewQuantity    — new quantity in base asset (optional, decimal.Zero
                     means "leave unchanged").
  - NewPrice       — new limit price (optional, decimal.Zero means
                     "leave unchanged").

INVARIANTS:
  - Exactly one identifier must be set.
  - At least one of NewQuantity / NewPrice must be non-zero, otherwise
    Bybit rejects with retCode 10001.
*/

package types

import "github.com/shopspring/decimal"

// ModifyOrderRequest — order amend request for the linear category.
type ModifyOrderRequest struct {
	Symbol        string
	OrderID       string
	ClientOrderID string
	NewQuantity   decimal.Decimal
	NewPrice      decimal.Decimal
}
