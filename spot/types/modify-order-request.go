/*
FILE: spot/types/modify-order-request.go

DESCRIPTION:
Order amend request for the Bybit V5 spot category. Bybit's
POST /v5/order/amend (category=spot) supports amending qty and price
only; side / type / timeInForce CANNOT be amended (cancel + re-create
in that case).

FIELDS:
  - Symbol         — Bybit symbol.
  - OrderID        — exchange order id; one of OrderID/ClientOrderID
                     must be set.
  - ClientOrderID  — orderLinkId.
  - NewQuantity    — new quantity (decimal.Zero leaves unchanged).
  - NewPrice       — new limit price (decimal.Zero leaves unchanged).

INVARIANTS:
  - Exactly one identifier (OrderID xor ClientOrderID) must be set.
  - At least one of NewQuantity / NewPrice must be non-zero; otherwise
    Bybit rejects with retCode 10001.
*/

package types

import "github.com/shopspring/decimal"

// ModifyOrderRequest — order amend request for the spot category.
type ModifyOrderRequest struct {
	Symbol        string
	OrderID       string
	ClientOrderID string
	NewQuantity   decimal.Decimal
	NewPrice      decimal.Decimal
}
