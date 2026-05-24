/*
FILE: premarket/types/risk-limit.go
*/

package types

import (
	"github.com/shopspring/decimal"

	commontypes "github.com/tonymontanov/go-bybit/v2/types"
)

// RiskLimitRequest — input for GET /v5/market/risk-limit.
type RiskLimitRequest struct {
	Category commontypes.Category
	Symbol   string
	Cursor   string
}

// RiskLimitTier — one risk-limit row for a symbol.
type RiskLimitTier struct {
	ID                int
	Symbol            string
	RiskLimitValue    decimal.Decimal
	MaintenanceMargin decimal.Decimal
	InitialMargin     decimal.Decimal
	IsLowestRisk      bool
	MaxLeverage       decimal.Decimal
	MMDeduction       decimal.Decimal
}

// RiskLimitList — paginated risk-limit page.
type RiskLimitList struct {
	Tiers          []RiskLimitTier
	NextPageCursor string
}
