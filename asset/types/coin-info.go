/*
FILE: asset/types/coin-info.go

DESCRIPTION:
Coin metadata from GET /v5/asset/coin/query-info.
*/

package types

import "github.com/shopspring/decimal"

// CoinChainInfo — per-chain deposit/withdraw constraints for a coin.
type CoinChainInfo struct {
	Chain                   string
	ChainType               string
	Confirmation            string
	WithdrawFee             decimal.Decimal
	DepositMin              decimal.Decimal
	WithdrawMin             decimal.Decimal
	MinAccuracy             string
	ChainDeposit            string // "0" suspend, "1" normal
	ChainWithdraw           string // "0" suspend, "1" normal
	WithdrawPercentageFee   decimal.Decimal
	ContractAddress         string
	SafeConfirmNumber       string
	WithdrawMax             decimal.Decimal
}

// CoinInfo — coin specification with chain breakdown.
type CoinInfo struct {
	Name  string
	Coin  string
	Chains []CoinChainInfo
}
