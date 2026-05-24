/*
FILE: affiliate/types/user-info.go
*/

package types

import "github.com/shopspring/decimal"

// AffiliateUserInfoRequest — input for GET /v5/user/aff-customer-info.
type AffiliateUserInfoRequest struct {
	UID      string
	Coin     string
	Business BusinessLine
}

// AffiliateUserInfo — detailed stats for one affiliate client UID.
type AffiliateUserInfo struct {
	UID                  string
	VIPLevel             string
	KYCLevel             int
	TakerVol30Day        decimal.Decimal
	MakerVol30Day        decimal.Decimal
	TradeVol30Day        decimal.Decimal
	DepositAmount30Day   decimal.Decimal
	TakerVol365Day       decimal.Decimal
	MakerVol365Day       decimal.Decimal
	TradeVol365Day       decimal.Decimal
	DepositAmount365Day  decimal.Decimal
	TotalWalletBalance   string
	DepositUpdateTime    string
	VolUpdateTime        string
	TradFiTradeVol30Day  decimal.Decimal
	TradFiTradeVol365Day decimal.Decimal
	Commissions30Day     map[string]decimal.Decimal
	Commissions365Day    map[string]decimal.Decimal
	PaySendAmount30Day   decimal.Decimal
	PayFTT               decimal.Decimal
	CardFTT              decimal.Decimal
}
