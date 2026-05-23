/*
FILE: spot/types/batch-result.go

DESCRIPTION:
Result of a single sub-request inside a Bybit V5 batch call
(POST /v5/order/{create,amend,cancel}-batch with category=spot). Wire
shape is identical to linears: result.list[i] carries identifiers and
retExtInfo.list[i] carries the {code, msg} verdict.

Note: Bybit V5 spot batch endpoints REQUIRE the UTA account (Unified
Trading Account). Batch calls on classic spot accounts are rejected
with retCode 10005.
*/

package types

// BatchOrderResult — result of a single sub-request inside a batch call.
type BatchOrderResult struct {
	// Order is populated when the sub-request succeeded; for failed
	// rows only OrderID/ClientOrderID may be set (echoed from the
	// request) and the remaining fields are zero values.
	Order OrderInfo
	// Code is the Bybit retCode for this sub-request. 0 == success.
	Code int
	// Message is the Bybit retMsg for this sub-request.
	Message string
}

// IsOK reports whether the sub-request completed successfully.
func (b BatchOrderResult) IsOK() bool {
	return b.Code == 0
}
