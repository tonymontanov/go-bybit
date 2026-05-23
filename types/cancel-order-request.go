/*
FILE: types/cancel-order-request.go

DESCRIPTION:
Order cancellation request — protocol-common across every Bybit V5
category. Symbol is mandatory; exactly one identifier (OrderID xor
ClientOrderID) must be set.
*/

package types

// CancelOrderRequest — order cancellation request.
type CancelOrderRequest struct {
	Symbol        string
	OrderID       string
	ClientOrderID string
}
