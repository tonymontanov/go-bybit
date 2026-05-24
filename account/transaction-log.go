/*
FILE: account/transaction-log.go

DESCRIPTION:
GET /v5/account/transaction-log — UTA transaction journal.
*/

package account

import (
	"context"
	"net/url"
	"strconv"

	bybit "github.com/tonymontanov/go-bybit/v2"
	"github.com/tonymontanov/go-bybit/v2/account/types"
	"github.com/tonymontanov/go-bybit/v2/internal/rest"
	commontypes "github.com/tonymontanov/go-bybit/v2/types"
)

type rawTransactionLogRow struct {
	ID              string `json:"id"`
	Symbol          string `json:"symbol"`
	Category        string `json:"category"`
	Side            string `json:"side"`
	TransactionTime string `json:"transactionTime"`
	Type            string `json:"type"`
	TransSubType    string `json:"transSubType"`
	Qty             string `json:"qty"`
	Size            string `json:"size"`
	Currency        string `json:"currency"`
	TradePrice      string `json:"tradePrice"`
	Funding         string `json:"funding"`
	Fee             string `json:"fee"`
	CashFlow        string `json:"cashFlow"`
	Change          string `json:"change"`
	CashBalance     string `json:"cashBalance"`
	FeeRate         string `json:"feeRate"`
	BonusChange     string `json:"bonusChange"`
	TradeID         string `json:"tradeId"`
	OrderID         string `json:"orderId"`
	OrderLinkID     string `json:"orderLinkId"`
	ExtraFees       string `json:"extraFees"`
}

type transactionLogPayload struct {
	List           []rawTransactionLogRow `json:"list"`
	NextPageCursor string                 `json:"nextPageCursor"`
}

func convertTransactionLog(raw rawTransactionLogRow) types.TransactionLogEntry {
	return types.TransactionLogEntry{
		ID:              raw.ID,
		Symbol:          raw.Symbol,
		Category:        commontypes.Category(raw.Category),
		Side:            commontypes.SideType(raw.Side),
		TransactionTime: ms(raw.TransactionTime),
		Type:            raw.Type,
		TransSubType:    raw.TransSubType,
		Qty:             dec(raw.Qty),
		Size:            dec(raw.Size),
		Currency:        raw.Currency,
		TradePrice:      dec(raw.TradePrice),
		Funding:         dec(raw.Funding),
		Fee:             dec(raw.Fee),
		CashFlow:        dec(raw.CashFlow),
		Change:          dec(raw.Change),
		CashBalance:     dec(raw.CashBalance),
		FeeRate:         dec(raw.FeeRate),
		BonusChange:     dec(raw.BonusChange),
		TradeID:         raw.TradeID,
		OrderID:         raw.OrderID,
		OrderLinkID:     raw.OrderLinkID,
		ExtraFees:       raw.ExtraFees,
	}
}

// GetTransactionLog returns paginated UTA transaction-log entries.
func (c *Client) GetTransactionLog(ctx context.Context, req types.TransactionLogRequest) (types.TransactionLogList, error) {
	var query url.Values = url.Values{}
	if req.AccountType != "" {
		query.Set("accountType", string(req.AccountType))
	}
	if req.Category != "" {
		query.Set("category", string(req.Category))
	}
	if req.Currency != "" {
		query.Set("currency", req.Currency)
	}
	if req.BaseCoin != "" {
		query.Set("baseCoin", req.BaseCoin)
	}
	if req.Type != "" {
		query.Set("type", req.Type)
	}
	if req.TransSubType != "" {
		query.Set("transSubType", req.TransSubType)
	}
	if req.StartTimeMs > 0 {
		query.Set("startTime", strconv.FormatInt(req.StartTimeMs, 10))
	}
	if req.EndTimeMs > 0 {
		query.Set("endTime", strconv.FormatInt(req.EndTimeMs, 10))
	}
	if req.Limit > 0 {
		query.Set("limit", strconv.Itoa(req.Limit))
	}
	if req.Cursor != "" {
		query.Set("cursor", req.Cursor)
	}

	var resp rest.Response
	var err error
	resp, _, err = c.rest().Do(ctx, rest.Options{
		Method: "GET",
		Path:   "/v5/account/transaction-log",
		Query:  query,
		Signed: true,
		Meta: rest.RequestMeta{
			Category: string(bybit.RateLimitCategoryQuery),
		},
	})
	if err != nil {
		return types.TransactionLogList{}, err
	}

	var payload transactionLogPayload
	if err = resp.UnmarshalResult(&payload); err != nil {
		return types.TransactionLogList{}, bybit.NewError(bybit.ErrorKindUnknown, "", "account.GetTransactionLog: parse", err)
	}

	var out types.TransactionLogList
	out.NextPageCursor = payload.NextPageCursor
	var i int
	for i = 0; i < len(payload.List); i++ {
		out.Records = append(out.Records, convertTransactionLog(payload.List[i]))
	}
	return out, nil
}
