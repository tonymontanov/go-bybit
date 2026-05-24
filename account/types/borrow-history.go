/*
FILE: account/types/borrow-history.go
*/

package types

import "github.com/shopspring/decimal"

// BorrowHistoryRequest — filters for GET /v5/account/borrow-history.
type BorrowHistoryRequest struct {
	Currency    string
	StartTimeMs int64
	EndTimeMs   int64
	Limit       int
	Cursor      string
}

// BorrowHistoryEntry — one borrow-interest record.
type BorrowHistoryEntry struct {
	Currency                  string
	CreatedAtMs               int64
	BorrowCost                decimal.Decimal
	HourlyBorrowRate          decimal.Decimal
	InterestBearingBorrowSize decimal.Decimal
	CostExemption             decimal.Decimal
	BorrowAmount              decimal.Decimal
	UnrealisedLoss            decimal.Decimal
	FreeBorrowedAmount        decimal.Decimal
}

// BorrowHistoryList — paginated borrow-history page.
type BorrowHistoryList struct {
	Records        []BorrowHistoryEntry
	NextPageCursor string
}
