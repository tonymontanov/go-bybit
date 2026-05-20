/*
FILE: linears/types/cancel-order-request.go

DESCRIPTION:
Order cancellation request for Bybit V5 linear. Symbol is mandatory;
exactly one identifier (OrderID or ClientOrderID) must be set.
*/

package types

// CancelOrderRequest — order cancellation request for the linear category.
type CancelOrderRequest struct {
	Symbol        string
	OrderID       string
	ClientOrderID string
}
