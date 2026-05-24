/*
FILE: linears/types/funding-rate-history.go

DESCRIPTION:
Historical funding rates for linear/inverse perpetuals
(GET /v5/market/funding/history).
*/

package types

import (
	"github.com/shopspring/decimal"
)

// FundingRateHistoryRequest — input for GetFundingRateHistory.
type FundingRateHistoryRequest struct {
	Symbol  string
	StartMs int64
	EndMs   int64
	Limit   int
}

// FundingRateRecord — one settled funding rate row.
type FundingRateRecord struct {
	Symbol      string
	FundingRate decimal.Decimal
	TimestampMs int64
}

// FundingRateHistory — funding rate history page (no cursor on wire).
type FundingRateHistory struct {
	Records []FundingRateRecord
}
