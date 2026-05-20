/*
FILE: linears/types/batch-result.go

DESCRIPTION:
Bybit V5 batch endpoints (POST /v5/order/{create,amend,cancel}-batch)
return per-request status under TWO keys: result.list[i] holds the
"happy-path" identifiers (orderId, orderLinkId, ...), and
retExtInfo.list[i] holds the per-request {code, msg} pair. Even when the
top-level retCode is 0, individual rows can be rejected.

BatchOrderResult flattens these two slices into a single shape that the
caller can iterate over directly. Successful rows have Code == 0.
*/

package types

// BatchOrderResult — result of a single sub-request inside a batch call.
type BatchOrderResult struct {
	// Order is populated when the sub-request succeeded; for failed
	// rows only OrderID/ClientOrderID may be set (echoed from the
	// request) and the remaining fields are zero values.
	Order OrderInfo
	// Code is the Bybit retCode for this specific sub-request.
	// 0 (or its string form "0") means success.
	Code int
	// Message is the Bybit retMsg for this sub-request. "OK" / "" on success.
	Message string
}

// IsOK reports whether the sub-request completed successfully.
func (b BatchOrderResult) IsOK() bool {
	return b.Code == 0
}
