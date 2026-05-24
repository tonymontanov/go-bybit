/*
FILE: linears/types/open-interest.go

DESCRIPTION:
Historical open interest for linear/inverse contracts
(GET /v5/market/open-interest).
*/

package types

import "github.com/shopspring/decimal"

// OpenInterestInterval — sampling interval for open-interest history.
type OpenInterestInterval string

const (
	OpenInterestInterval5m  OpenInterestInterval = "5min"
	OpenInterestInterval15m OpenInterestInterval = "15min"
	OpenInterestInterval30m OpenInterestInterval = "30min"
	OpenInterestInterval1h  OpenInterestInterval = "1h"
	OpenInterestInterval4h  OpenInterestInterval = "4h"
	OpenInterestInterval1d  OpenInterestInterval = "1d"
)

// OpenInterestRequest — input for GetOpenInterest.
type OpenInterestRequest struct {
	Symbol       string
	IntervalTime OpenInterestInterval
	StartMs      int64
	EndMs        int64
	Limit        int
	Cursor       string
}

// OpenInterestRecord — one open-interest sample.
type OpenInterestRecord struct {
	OpenInterest decimal.Decimal
	TimestampMs  int64
}

// OpenInterestHistory — paginated open-interest page.
type OpenInterestHistory struct {
	Symbol         string
	Records        []OpenInterestRecord
	NextPageCursor string
}
