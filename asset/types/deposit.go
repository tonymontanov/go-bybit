/*
FILE: asset/types/deposit.go

DESCRIPTION:
Deposit types for /v5/asset/deposit/* endpoints.
*/

package types

import (
	"github.com/shopspring/decimal"
	commontypes "github.com/tonymontanov/go-bybit/v2/types"
)

// DepositAddress — on-chain deposit address for a coin/chain pair.
type DepositAddress struct {
	Coin   string
	Chain  string
	Address string
	Tag    string
}

// DepositRecord — row from GET /v5/asset/deposit/query-record.
type DepositRecord struct {
	ID              string
	Coin            string
	Chain           string
	Amount          decimal.Decimal
	TxID            string
	Status          DepositStatus
	ToAddress       string
	Tag             string
	DepositFee      decimal.Decimal
	SuccessAtMs     int64
	Confirmations   string
	TxIndex         string
	BlockHash       string
	BatchReleaseLimit decimal.Decimal
	DepositType     string
}

// DepositRecordList — paginated deposit history.
type DepositRecordList struct {
	Records        []DepositRecord
	NextPageCursor string
}

// SetDepositAccountRequest — POST /v5/asset/deposit/deposit-to-account.
type SetDepositAccountRequest struct {
	AccountType commontypes.AccountType
}
