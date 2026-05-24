/*
FILE: broker/deposit.go

DESCRIPTION:
GET /v5/broker/asset/query-sub-member-deposit-record — sub-account deposits.
*/

package broker

import (
	"context"
	"net/url"
	"strconv"

	bybit "github.com/tonymontanov/go-bybit/v2"
	"github.com/tonymontanov/go-bybit/v2/broker/types"
	"github.com/tonymontanov/go-bybit/v2/internal/rest"
)

type rawSubMemberDepositRow struct {
	ID                  string `json:"id"`
	SubMemberID         string `json:"subMemberId"`
	Coin                string `json:"coin"`
	Chain               string `json:"chain"`
	Amount              string `json:"amount"`
	TxID                string `json:"txID"`
	Status              int    `json:"status"`
	ToAddress           string `json:"toAddress"`
	Tag                 string `json:"tag"`
	DepositFee          string `json:"depositFee"`
	SuccessAt           string `json:"successAt"`
	Confirmations       string `json:"confirmations"`
	TxIndex             string `json:"txIndex"`
	BlockHash           string `json:"blockHash"`
	BatchReleaseLimit   string `json:"batchReleaseLimit"`
	DepositType         string `json:"depositType"`
	FromAddress         string `json:"fromAddress"`
	TaxDepositRecordsID string `json:"taxDepositRecordsId"`
	TaxStatus           int    `json:"taxStatus"`
}

type subMemberDepositPayload struct {
	Rows           []rawSubMemberDepositRow `json:"rows"`
	NextPageCursor string                   `json:"nextPageCursor"`
}

func convertSubMemberDeposit(raw rawSubMemberDepositRow) types.SubMemberDepositRecord {
	return types.SubMemberDepositRecord{
		ID:                  raw.ID,
		SubMemberID:         raw.SubMemberID,
		Coin:                raw.Coin,
		Chain:               raw.Chain,
		Amount:              dec(raw.Amount),
		TxID:                raw.TxID,
		Status:              raw.Status,
		ToAddress:           raw.ToAddress,
		Tag:                 raw.Tag,
		DepositFee:          dec(raw.DepositFee),
		SuccessAt:           raw.SuccessAt,
		Confirmations:       raw.Confirmations,
		TxIndex:             raw.TxIndex,
		BlockHash:           raw.BlockHash,
		BatchReleaseLimit:   raw.BatchReleaseLimit,
		DepositType:         raw.DepositType,
		FromAddress:         raw.FromAddress,
		TaxDepositRecordsID: raw.TaxDepositRecordsID,
		TaxStatus:           raw.TaxStatus,
	}
}

// GetSubMemberDepositRecords returns on-chain deposit records for sub-accounts.
func (c *Client) GetSubMemberDepositRecords(ctx context.Context, req types.SubMemberDepositRequest) (types.SubMemberDepositList, error) {
	var limit int = req.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 50 {
		limit = 50
	}

	var query url.Values = url.Values{}
	if req.ID != "" {
		query.Set("id", req.ID)
	}
	if req.TxID != "" {
		query.Set("txID", req.TxID)
	}
	if req.SubMemberID != "" {
		query.Set("subMemberId", req.SubMemberID)
	}
	if req.Coin != "" {
		query.Set("coin", req.Coin)
	}
	if req.StartTimeMs > 0 {
		query.Set("startTime", strconv.FormatInt(req.StartTimeMs, 10))
	}
	if req.EndTimeMs > 0 {
		query.Set("endTime", strconv.FormatInt(req.EndTimeMs, 10))
	}
	query.Set("limit", strconv.Itoa(limit))
	if req.Cursor != "" {
		query.Set("cursor", req.Cursor)
	}

	var resp rest.Response
	var err error
	resp, _, err = c.rest().Do(ctx, rest.Options{
		Method: "GET",
		Path:   "/v5/broker/asset/query-sub-member-deposit-record",
		Query:  query,
		Signed: true,
		Meta: rest.RequestMeta{
			Category: string(bybit.RateLimitCategoryQuery),
		},
	})
	if err != nil {
		return types.SubMemberDepositList{}, err
	}

	var payload subMemberDepositPayload
	if err = resp.UnmarshalResult(&payload); err != nil {
		return types.SubMemberDepositList{}, bybit.NewError(bybit.ErrorKindUnknown, "", "broker.GetSubMemberDepositRecords: parse", err)
	}

	var out types.SubMemberDepositList
	out.NextPageCursor = payload.NextPageCursor
	out.Records = make([]types.SubMemberDepositRecord, 0, len(payload.Rows))
	var i int
	for i = 0; i < len(payload.Rows); i++ {
		out.Records = append(out.Records, convertSubMemberDeposit(payload.Rows[i]))
	}
	return out, nil
}
