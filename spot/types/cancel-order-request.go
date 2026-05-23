/*
FILE: spot/types/cancel-order-request.go

DESCRIPTION:
Order cancellation request for the Bybit V5 spot category. Symbol is
mandatory; exactly one identifier (OrderID xor ClientOrderID) must be
set.
*/

package types

// CancelOrderRequest — order cancellation request for the spot category.
type CancelOrderRequest struct {
	Symbol        string
	OrderID       string
	ClientOrderID string
}
