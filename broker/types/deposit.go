/*
FILE: broker/types/deposit.go
*/

package types

import "github.com/shopspring/decimal"

// SubMemberDepositRequest — filters for sub-account deposit records.
type SubMemberDepositRequest struct {
	ID           string
	TxID         string
	SubMemberID  string
	Coin         string
	StartTimeMs  int64
	EndTimeMs    int64
	Limit        int
	Cursor       string
}

// SubMemberDepositRecord — one sub-account on-chain deposit row.
type SubMemberDepositRecord struct {
	ID                  string
	SubMemberID         string
	Coin                string
	Chain               string
	Amount              decimal.Decimal
	TxID                string
	Status              int
	ToAddress           string
	Tag                 string
	DepositFee          decimal.Decimal
	SuccessAt           string
	Confirmations       string
	TxIndex             string
	BlockHash           string
	BatchReleaseLimit   string
	DepositType         string
	FromAddress         string
	TaxDepositRecordsID string
	TaxStatus           int
}

// SubMemberDepositList — paginated sub-account deposit page.
type SubMemberDepositList struct {
	Records        []SubMemberDepositRecord
	NextPageCursor string
}
