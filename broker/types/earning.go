/*
FILE: broker/types/earning.go
*/

package types

import "github.com/shopspring/decimal"

// EarningsRequest — filters for GET /v5/broker/earnings-info.
type EarningsRequest struct {
	BizType BizType
	Begin   string
	End     string
	UID     string
	Limit   int
	Cursor  string
}

// CoinEarning — rebate total for one coin within a category bucket.
type CoinEarning struct {
	Coin    string
	Earning decimal.Decimal
}

// EarningsCategoryTotals — per-category rebate aggregates.
type EarningsCategoryTotals struct {
	Spot        []CoinEarning
	Derivatives []CoinEarning
	Options     []CoinEarning
	Convert     []CoinEarning
	Total       []CoinEarning
}

// EarningDetail — one rebate row from earnings-info details.
type EarningDetail struct {
	UserID          string
	BizType         BizType
	Symbol          string
	Coin            string
	Earning         decimal.Decimal
	MarkupEarning   decimal.Decimal
	BaseFeeEarning  decimal.Decimal
	OrderID         string
	ExecID          string
	ExecTimeMs      int64
}

// EarningsList — paginated broker earnings page.
type EarningsList struct {
	CategoryTotals EarningsCategoryTotals
	Details        []EarningDetail
	NextPageCursor string
}
