/*
FILE: broker/types/award.go
*/

package types

import "github.com/shopspring/decimal"

// VoucherSpec — POST /v5/broker/award/info result.
type VoucherSpec struct {
	ID             string
	Coin           string
	AmountUnit     string
	ProductLine    string
	SubProductLine string
	TotalAmount    decimal.Decimal
	UsedAmount     decimal.Decimal
}

// DistributeVoucherRequest — POST /v5/broker/award/distribute-award body.
type DistributeVoucherRequest struct {
	AccountID string
	AwardID   string
	SpecCode  string
	Amount    decimal.Decimal
	BrokerID  string
}

// VoucherDistributionRequest — POST /v5/broker/award/distribution-record body.
type VoucherDistributionRequest struct {
	AccountID      string
	AwardID        string
	SpecCode       string
	WithUsedAmount bool
}

// VoucherDistribution — issued voucher state.
type VoucherDistribution struct {
	AccountID     string
	AwardID       string
	SpecCode      string
	Amount        decimal.Decimal
	IsClaimed     bool
	StartAtSec    int64
	EndAtSec      int64
	EffectiveAtSec   int64
	IneffectiveAtSec int64
	UsedAmount    decimal.Decimal
}
