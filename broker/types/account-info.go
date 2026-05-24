/*
FILE: broker/types/account-info.go
*/

package types

// FeeRebateRate — percentage strings returned by broker account-info.
type FeeRebateRate struct {
	Spot        string
	Derivatives string
	Convert     string
}

// AccountInfo — GET /v5/broker/account-info result.
type AccountInfo struct {
	SubAccountQty       string
	MaxSubAccountQty    string
	BaseFeeRebateRate   FeeRebateRate
	MarkupFeeRebateRate FeeRebateRate
	TimestampMs         int64
}
