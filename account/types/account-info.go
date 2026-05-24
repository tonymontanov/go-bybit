/*
FILE: account/types/account-info.go
*/

package types

// AccountInfo — GET /v5/account/info result.
type AccountInfo struct {
	UnifiedMarginStatus int
	MarginMode          MarginMode
	IsMasterTrader      bool
	SpotHedgingStatus   HedgingMode
	UpdatedAtMs         int64
}
