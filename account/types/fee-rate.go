/*
FILE: account/types/fee-rate.go
*/

package types

import (
	"github.com/shopspring/decimal"

	commontypes "github.com/tonymontanov/go-bybit/v2/types"
)

// FeeRateRequest — input for GET /v5/account/fee-rate.
type FeeRateRequest struct {
	Category commontypes.Category
	Symbol   string
	BaseCoin string
}

// FeeRate — maker/taker fee for one symbol or base coin.
type FeeRate struct {
	Category     commontypes.Category
	Symbol       string
	BaseCoin     string
	TakerFeeRate decimal.Decimal
	MakerFeeRate decimal.Decimal
}

// FeeRateList — paginated fee-rate response (single page; no cursor on wire).
type FeeRateList struct {
	List []FeeRate
}
