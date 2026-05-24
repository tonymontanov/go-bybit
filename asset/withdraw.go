/*
FILE: asset/withdraw.go

DESCRIPTION:
Withdrawal endpoints under /v5/asset/withdraw/*.
*/

package asset

import (
	"context"
	"net/url"
	"strconv"
	"time"

	bybit "github.com/tonymontanov/go-bybit/v2"
	"github.com/tonymontanov/go-bybit/v2/asset/types"
	"github.com/tonymontanov/go-bybit/v2/internal/rest"
)

type rawCreateWithdrawResult struct {
	ID string `json:"id"`
}

type rawWithdrawRecord struct {
	WithdrawID  string `json:"withdrawId"`
	TxID        string `json:"txID"`
	Coin        string `json:"coin"`
	Chain       string `json:"chain"`
	Amount      string `json:"amount"`
	WithdrawFee string `json:"withdrawFee"`
	Status      string `json:"status"`
	ToAddress   string `json:"toAddress"`
	Tag         string `json:"tag"`
	CreateTime  string `json:"createTime"`
	UpdateTime  string `json:"updateTime"`
}

type withdrawRecordListPayload struct {
	Rows           []rawWithdrawRecord `json:"rows"`
	NextPageCursor string              `json:"nextPageCursor"`
}

type withdrawableAmountPayload struct {
	LimitAmountUsd        string `json:"limitAmountUsd"`
	WithdrawableAmount    string `json:"withdrawableAmount"`
	WithdrawableAmountUsd string `json:"withdrawableAmountUsd"`
}

// CreateWithdraw submits a withdrawal request. TimestampMs defaults to
// time.Now() when zero (Bybit requires a fresh ms timestamp for replay
// protection).
func (c *Client) CreateWithdraw(ctx context.Context, req types.CreateWithdrawRequest) (types.CreateWithdrawResult, error) {
	var out types.CreateWithdrawResult
	if req.Coin == "" {
		return out, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "asset.CreateWithdraw: coin is empty", nil)
	}
	if req.Address == "" {
		return out, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "asset.CreateWithdraw: address is empty", nil)
	}
	if !req.Amount.IsPositive() {
		return out, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "asset.CreateWithdraw: amount must be positive", nil)
	}
	if req.AccountType == "" {
		return out, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "asset.CreateWithdraw: accountType is empty", nil)
	}

	var tsMs int64 = req.TimestampMs
	if tsMs <= 0 {
		tsMs = time.Now().UnixMilli()
	}

	var body map[string]any = map[string]any{
		"coin":        req.Coin,
		"address":     req.Address,
		"amount":      req.Amount.String(),
		"timestamp":   tsMs,
		"accountType": string(req.AccountType),
	}
	if req.Chain != "" {
		body["chain"] = req.Chain
	}
	if req.Tag != "" {
		body["tag"] = req.Tag
	}
	if req.ForceChain != 0 {
		body["forceChain"] = req.ForceChain
	}
	if req.FeeType != 0 {
		body["feeType"] = req.FeeType
	}
	if req.RequestID != "" {
		body["requestId"] = req.RequestID
	}

	var resp rest.Response
	var err error
	resp, _, err = c.rest().Do(ctx, rest.Options{
		Method: "POST",
		Path:   "/v5/asset/withdraw/create",
		Body:   body,
		Signed: true,
		Meta: rest.RequestMeta{
			Category: string(bybit.RateLimitCategoryQuery),
		},
	})
	if err != nil {
		return out, err
	}

	var payload rawCreateWithdrawResult
	if err = resp.UnmarshalResult(&payload); err != nil {
		return out, bybit.NewError(bybit.ErrorKindUnknown, "", "asset.CreateWithdraw: parse", err)
	}
	return types.CreateWithdrawResult{ID: payload.ID}, nil
}

// CancelWithdraw cancels a pending withdrawal by ID.
func (c *Client) CancelWithdraw(ctx context.Context, withdrawID string) error {
	if withdrawID == "" {
		return bybit.NewError(bybit.ErrorKindInvalidRequest, "", "asset.CancelWithdraw: withdrawId is empty", nil)
	}

	var body map[string]any = map[string]any{
		"id": withdrawID,
	}

	var resp rest.Response
	var err error
	resp, _, err = c.rest().Do(ctx, rest.Options{
		Method: "POST",
		Path:   "/v5/asset/withdraw/cancel",
		Body:   body,
		Signed: true,
		Meta: rest.RequestMeta{
			Category: string(bybit.RateLimitCategoryQuery),
		},
	})
	if err != nil {
		return err
	}
	_ = resp
	return nil
}

// WithdrawRecordListRequest — filters for GetWithdrawRecords.
type WithdrawRecordListRequest struct {
	WithdrawID  string
	Coin        string
	WithdrawType int
	StartTimeMs int64
	EndTimeMs   int64
	Limit       int
	Cursor      string
}

// GetWithdrawRecords returns paginated withdrawal history.
func (c *Client) GetWithdrawRecords(ctx context.Context, req WithdrawRecordListRequest) (types.WithdrawRecordList, error) {
	var out types.WithdrawRecordList
	var query url.Values = url.Values{}
	if req.WithdrawID != "" {
		query.Set("withdrawID", req.WithdrawID)
	}
	if req.Coin != "" {
		query.Set("coin", req.Coin)
	}
	if req.WithdrawType > 0 {
		query.Set("withdrawType", strconv.Itoa(req.WithdrawType))
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
		Path:   "/v5/asset/withdraw/query-record",
		Query:  query,
		Signed: true,
		Meta: rest.RequestMeta{
			Category: string(bybit.RateLimitCategoryQuery),
		},
	})
	if err != nil {
		return out, err
	}

	var payload withdrawRecordListPayload
	if err = resp.UnmarshalResult(&payload); err != nil {
		return out, bybit.NewError(bybit.ErrorKindUnknown, "", "asset.GetWithdrawRecords: parse", err)
	}

	out.NextPageCursor = payload.NextPageCursor
	out.Records = make([]types.WithdrawRecord, 0, len(payload.Rows))
	for _, row := range payload.Rows {
		out.Records = append(out.Records, types.WithdrawRecord{
			ID:           row.WithdrawID,
			TxID:         row.TxID,
			Coin:         row.Coin,
			Chain:        row.Chain,
			Amount:       dec(row.Amount),
			WithdrawFee:  dec(row.WithdrawFee),
			Status:       types.WithdrawStatus(row.Status),
			ToAddress:    row.ToAddress,
			Tag:          row.Tag,
			CreateTimeMs: ms(row.CreateTime),
			UpdateTimeMs: ms(row.UpdateTime),
		})
	}
	return out, nil
}

// GetWithdrawableAmount returns the withdrawable balance for a coin.
func (c *Client) GetWithdrawableAmount(ctx context.Context, coin string) (types.WithdrawableAmount, error) {
	var out types.WithdrawableAmount
	if coin == "" {
		return out, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "asset.GetWithdrawableAmount: coin is empty", nil)
	}

	var query url.Values = url.Values{}
	query.Set("coin", coin)

	var resp rest.Response
	var err error
	resp, _, err = c.rest().Do(ctx, rest.Options{
		Method: "GET",
		Path:   "/v5/asset/withdraw/withdrawable-amount",
		Query:  query,
		Signed: true,
		Meta: rest.RequestMeta{
			Category: string(bybit.RateLimitCategoryQuery),
		},
	})
	if err != nil {
		return out, err
	}

	var payload withdrawableAmountPayload
	if err = resp.UnmarshalResult(&payload); err != nil {
		return out, bybit.NewError(bybit.ErrorKindUnknown, "", "asset.GetWithdrawableAmount: parse", err)
	}
	return types.WithdrawableAmount{
		LimitAmountUsd:        dec(payload.LimitAmountUsd),
		WithdrawableAmount:    dec(payload.WithdrawableAmount),
		WithdrawableAmountUsd: dec(payload.WithdrawableAmountUsd),
	}, nil
}
