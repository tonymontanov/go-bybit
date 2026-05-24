/*
FILE: asset/types/transfer.go

DESCRIPTION:
Internal transfer types for /v5/asset/transfer/* endpoints.
*/

package types

import (
	"github.com/shopspring/decimal"
	commontypes "github.com/tonymontanov/go-bybit/v2/types"
)

// AccountCoinBalance — single-coin balance within an account type.
type AccountCoinBalance struct {
	Coin            string
	WalletBalance   decimal.Decimal
	TransferBalance decimal.Decimal
	Bonus           decimal.Decimal
}

// CreateInternalTransferRequest — POST /v5/asset/transfer/inter-transfer.
type CreateInternalTransferRequest struct {
	TransferID      string
	Coin            string
	Amount          decimal.Decimal
	FromAccountType commontypes.AccountType
	ToAccountType   commontypes.AccountType
}

// InternalTransferResult — immediate response from create inter-transfer.
type InternalTransferResult struct {
	TransferID string
	Status     TransferStatus
}

// InternalTransferRecord — row from query-inter-transfer-list.
type InternalTransferRecord struct {
	TransferID      string
	Coin            string
	Amount          decimal.Decimal
	FromAccountType commontypes.AccountType
	ToAccountType   commontypes.AccountType
	TimestampMs     int64
	Status          TransferStatus
}

// InternalTransferList — paginated internal transfer history.
type InternalTransferList struct {
	Records        []InternalTransferRecord
	NextPageCursor string
}
