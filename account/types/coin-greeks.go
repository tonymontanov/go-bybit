/*
FILE: account/types/coin-greeks.go
*/

package types

import "github.com/shopspring/decimal"

// CoinGreeks — account greeks for one base coin (GET /v5/asset/coin-greeks).
type CoinGreeks struct {
	BaseCoin   string
	TotalDelta decimal.Decimal
	TotalGamma decimal.Decimal
	TotalVega  decimal.Decimal
	TotalTheta decimal.Decimal
}
