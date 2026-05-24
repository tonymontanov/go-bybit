/*
FILE: account/types/transaction-log.go
*/

package types

import (
	"github.com/shopspring/decimal"

	commontypes "github.com/tonymontanov/go-bybit/v2/types"
)

// TransactionLogRequest — filters for GET /v5/account/transaction-log.
type TransactionLogRequest struct {
	AccountType  commontypes.AccountType
	Category     commontypes.Category
	Currency     string
	BaseCoin     string
	Type         string
	TransSubType string
	StartTimeMs  int64
	EndTimeMs    int64
	Limit        int
	Cursor       string
}

// TransactionLogEntry — one UTA transaction-log row.
type TransactionLogEntry struct {
	ID              string
	Symbol          string
	Category        commontypes.Category
	Side            commontypes.SideType
	TransactionTime int64
	Type            string
	TransSubType    string
	Qty             decimal.Decimal
	Size            decimal.Decimal
	Currency        string
	TradePrice      decimal.Decimal
	Funding         decimal.Decimal
	Fee             decimal.Decimal
	CashFlow        decimal.Decimal
	Change          decimal.Decimal
	CashBalance     decimal.Decimal
	FeeRate         decimal.Decimal
	BonusChange     decimal.Decimal
	TradeID         string
	OrderID         string
	OrderLinkID     string
	ExtraFees       string
}

// TransactionLogList — paginated transaction-log page.
type TransactionLogList struct {
	Records        []TransactionLogEntry
	NextPageCursor string
}
