/*
FILE: account/types/collateral-info.go
*/

package types

import "github.com/shopspring/decimal"

// CollateralInfo — one row from GET /v5/account/collateral-info.
type CollateralInfo struct {
	Currency            string
	HourlyBorrowRate    decimal.Decimal
	MaxBorrowingAmount  decimal.Decimal
	FreeBorrowingLimit  decimal.Decimal
	FreeBorrowAmount    decimal.Decimal
	BorrowAmount        decimal.Decimal
	OtherBorrowAmount   decimal.Decimal
	AvailableToBorrow   decimal.Decimal
	Borrowable          bool
	BorrowUsageRate     decimal.Decimal
	MarginCollateral    bool
	CollateralSwitch    bool
	CollateralRatio     decimal.Decimal
	FreeBorrowingAmount string
}
