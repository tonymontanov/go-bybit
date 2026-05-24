/*
FILE: affiliate/types/user-list.go
*/

package types

import "github.com/shopspring/decimal"

// AffiliateUserListRequest — filters for GET /v5/affiliate/aff-user-list.
type AffiliateUserListRequest struct {
	Size        int
	Cursor      string
	NeedDeposit bool
	Need30      bool
	Need365     bool
	StartDate   string
	EndDate     string
}

// AffiliateUser — one referred user row from aff-user-list.
type AffiliateUser struct {
	UserID              string
	RegisterTime        string
	Source              string
	Remarks             string
	IsKYC               bool
	TakerVol30Day       decimal.Decimal
	MakerVol30Day       decimal.Decimal
	TradeVol30Day       decimal.Decimal
	DepositAmount30Day  decimal.Decimal
	TakerVol365Day      decimal.Decimal
	MakerVol365Day      decimal.Decimal
	TradeVol365Day      decimal.Decimal
	DepositAmount365Day decimal.Decimal
	TakerVol            decimal.Decimal
	MakerVol            decimal.Decimal
	TradeVol            decimal.Decimal
	StartDate           string
	EndDate             string
	TradFiTradeVol      decimal.Decimal
	TradFiTradeVol30Day decimal.Decimal
	TradFiTradeVol365Day decimal.Decimal
	CommissionsVol      map[string]decimal.Decimal
	Commissions30Day    map[string]decimal.Decimal
	Commissions365Day   map[string]decimal.Decimal
}

// AffiliateUserList — paginated affiliate user list.
type AffiliateUserList struct {
	Users          []AffiliateUser
	NextPageCursor string
}
