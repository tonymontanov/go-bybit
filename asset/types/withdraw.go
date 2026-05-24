/*
FILE: asset/types/withdraw.go

DESCRIPTION:
Withdrawal types for /v5/asset/withdraw/* endpoints.
*/

package types

import (
	"github.com/shopspring/decimal"
	commontypes "github.com/tonymontanov/go-bybit/v2/types"
)

// CreateWithdrawRequest — POST /v5/asset/withdraw/create.
type CreateWithdrawRequest struct {
	Coin        string
	Chain       string
	Address     string
	Tag         string
	Amount      decimal.Decimal
	TimestampMs int64
	ForceChain  int // 0 internal if possible, 1 on-chain, 2 UID withdraw
	AccountType commontypes.AccountType
	FeeType     int
	RequestID   string
}

// CreateWithdrawResult — immediate response from withdraw/create.
type CreateWithdrawResult struct {
	ID string
}

// WithdrawRecord — row from GET /v5/asset/withdraw/query-record.
type WithdrawRecord struct {
	ID          string
	TxID        string
	Coin        string
	Chain       string
	Amount      decimal.Decimal
	WithdrawFee decimal.Decimal
	Status      WithdrawStatus
	ToAddress   string
	Tag         string
	CreateTimeMs int64
	UpdateTimeMs int64
}

// WithdrawRecordList — paginated withdrawal history.
type WithdrawRecordList struct {
	Records        []WithdrawRecord
	NextPageCursor string
}

// WithdrawableAmount — GET /v5/asset/withdraw/withdrawable-amount.
type WithdrawableAmount struct {
	LimitAmountUsd       decimal.Decimal
	WithdrawableAmount   decimal.Decimal
	WithdrawableAmountUsd decimal.Decimal
}
