/*
FILE: account/borrow.go

DESCRIPTION:
GET /v5/account/borrow-history — interest accrual records.
*/

package account

import (
	"context"
	"net/url"
	"strconv"

	bybit "github.com/tonymontanov/go-bybit/v2"
	"github.com/tonymontanov/go-bybit/v2/account/types"
	"github.com/tonymontanov/go-bybit/v2/internal/rest"
)

type rawBorrowHistoryRow struct {
	Currency                  string `json:"currency"`
	CreatedTime               int64  `json:"createdTime"`
	BorrowCost                string `json:"borrowCost"`
	HourlyBorrowRate          string `json:"hourlyBorrowRate"`
	InterestBearingBorrowSize string `json:"InterestBearingBorrowSize"`
	CostExemption             string `json:"costExemption"`
	BorrowAmount              string `json:"borrowAmount"`
	UnrealisedLoss            string `json:"unrealisedLoss"`
	FreeBorrowedAmount        string `json:"freeBorrowedAmount"`
}

type borrowHistoryPayload struct {
	List           []rawBorrowHistoryRow `json:"list"`
	NextPageCursor string                `json:"nextPageCursor"`
}

func convertBorrowHistory(raw rawBorrowHistoryRow) types.BorrowHistoryEntry {
	return types.BorrowHistoryEntry{
		Currency:                  raw.Currency,
		CreatedAtMs:               raw.CreatedTime,
		BorrowCost:                dec(raw.BorrowCost),
		HourlyBorrowRate:          dec(raw.HourlyBorrowRate),
		InterestBearingBorrowSize: dec(raw.InterestBearingBorrowSize),
		CostExemption:             dec(raw.CostExemption),
		BorrowAmount:              dec(raw.BorrowAmount),
		UnrealisedLoss:            dec(raw.UnrealisedLoss),
		FreeBorrowedAmount:        dec(raw.FreeBorrowedAmount),
	}
}

// GetBorrowHistory returns paginated borrow-interest records.
func (c *Client) GetBorrowHistory(ctx context.Context, req types.BorrowHistoryRequest) (types.BorrowHistoryList, error) {
	var query url.Values = url.Values{}
	if req.Currency != "" {
		query.Set("currency", req.Currency)
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
		Path:   "/v5/account/borrow-history",
		Query:  query,
		Signed: true,
		Meta: rest.RequestMeta{
			Category: string(bybit.RateLimitCategoryQuery),
		},
	})
	if err != nil {
		return types.BorrowHistoryList{}, err
	}

	var payload borrowHistoryPayload
	if err = resp.UnmarshalResult(&payload); err != nil {
		return types.BorrowHistoryList{}, bybit.NewError(bybit.ErrorKindUnknown, "", "account.GetBorrowHistory: parse", err)
	}

	var out types.BorrowHistoryList
	out.NextPageCursor = payload.NextPageCursor
	var i int
	for i = 0; i < len(payload.List); i++ {
		out.Records = append(out.Records, convertBorrowHistory(payload.List[i]))
	}
	return out, nil
}
