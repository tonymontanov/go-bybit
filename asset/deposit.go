/*
FILE: asset/deposit.go

DESCRIPTION:
Deposit endpoints under /v5/asset/deposit/*.
*/

package asset

import (
	"context"
	"net/url"
	"strconv"

	bybit "github.com/tonymontanov/go-bybit/v2"
	"github.com/tonymontanov/go-bybit/v2/asset/types"
	"github.com/tonymontanov/go-bybit/v2/internal/rest"
)

type rawDepositAddress struct {
	Coin    string `json:"coin"`
	Chain   string `json:"chain"`
	Address string `json:"address"`
	Tag     string `json:"tag"`
}

type depositAddressPayload struct {
	Chains []rawDepositAddress `json:"chains"`
}

type rawDepositRecord struct {
	Coin              string `json:"coin"`
	Chain             string `json:"chain"`
	Amount            string `json:"amount"`
	TxID              string `json:"txID"`
	Status            string `json:"status"`
	ToAddress         string `json:"toAddress"`
	Tag               string `json:"tag"`
	DepositFee        string `json:"depositFee"`
	SuccessAt         string `json:"successAt"`
	Confirmations     string `json:"confirmations"`
	TxIndex           string `json:"txIndex"`
	BlockHash         string `json:"blockHash"`
	BatchReleaseLimit string `json:"batchReleaseLimit"`
	DepositType       string `json:"depositType"`
	ID                string `json:"id"`
}

type depositRecordListPayload struct {
	Rows           []rawDepositRecord `json:"rows"`
	NextPageCursor string             `json:"nextPageCursor"`
}

type allowedDepositCoinPayload struct {
	Coins []string `json:"coins"`
}

// GetDepositAddress returns on-chain deposit addresses for a coin.
// Chain filter is optional.
func (c *Client) GetDepositAddress(ctx context.Context, coin, chain string) ([]types.DepositAddress, error) {
	if coin == "" {
		return nil, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "asset.GetDepositAddress: coin is empty", nil)
	}
	var query url.Values = url.Values{}
	query.Set("coin", coin)
	if chain != "" {
		query.Set("chain", chain)
	}

	var resp rest.Response
	var err error
	resp, _, err = c.rest().Do(ctx, rest.Options{
		Method: "GET",
		Path:   "/v5/asset/deposit/query-address",
		Query:  query,
		Signed: true,
		Meta: rest.RequestMeta{
			Category: string(bybit.RateLimitCategoryQuery),
		},
	})
	if err != nil {
		return nil, err
	}

	var payload depositAddressPayload
	if err = resp.UnmarshalResult(&payload); err != nil {
		return nil, bybit.NewError(bybit.ErrorKindUnknown, "", "asset.GetDepositAddress: parse", err)
	}

	var out []types.DepositAddress = make([]types.DepositAddress, 0, len(payload.Chains))
	for _, row := range payload.Chains {
		out = append(out, types.DepositAddress{
			Coin:    row.Coin,
			Chain:   row.Chain,
			Address: row.Address,
			Tag:     row.Tag,
		})
	}
	return out, nil
}

// DepositRecordListRequest — filters for GetDepositRecords.
type DepositRecordListRequest struct {
	Coin        string
	TxID        string
	StartTimeMs int64
	EndTimeMs   int64
	Limit       int
	Cursor      string
}

// GetDepositRecords returns paginated deposit history.
func (c *Client) GetDepositRecords(ctx context.Context, req DepositRecordListRequest) (types.DepositRecordList, error) {
	var out types.DepositRecordList
	var query url.Values = url.Values{}
	if req.Coin != "" {
		query.Set("coin", req.Coin)
	}
	if req.TxID != "" {
		query.Set("txID", req.TxID)
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
		Path:   "/v5/asset/deposit/query-record",
		Query:  query,
		Signed: true,
		Meta: rest.RequestMeta{
			Category: string(bybit.RateLimitCategoryQuery),
		},
	})
	if err != nil {
		return out, err
	}

	var payload depositRecordListPayload
	if err = resp.UnmarshalResult(&payload); err != nil {
		return out, bybit.NewError(bybit.ErrorKindUnknown, "", "asset.GetDepositRecords: parse", err)
	}

	out.NextPageCursor = payload.NextPageCursor
	out.Records = make([]types.DepositRecord, 0, len(payload.Rows))
	for _, row := range payload.Rows {
		out.Records = append(out.Records, types.DepositRecord{
			ID:                row.ID,
			Coin:              row.Coin,
			Chain:             row.Chain,
			Amount:            dec(row.Amount),
			TxID:              row.TxID,
			Status:            types.DepositStatus(row.Status),
			ToAddress:         row.ToAddress,
			Tag:               row.Tag,
			DepositFee:        dec(row.DepositFee),
			SuccessAtMs:       ms(row.SuccessAt),
			Confirmations:     row.Confirmations,
			TxIndex:           row.TxIndex,
			BlockHash:         row.BlockHash,
			BatchReleaseLimit: dec(row.BatchReleaseLimit),
			DepositType:       row.DepositType,
		})
	}
	return out, nil
}

// GetAllowedDepositCoins returns coins currently allowed for deposit.
func (c *Client) GetAllowedDepositCoins(ctx context.Context, coin, chain, limit, cursor string) ([]string, string, error) {
	var query url.Values = url.Values{}
	if coin != "" {
		query.Set("coin", coin)
	}
	if chain != "" {
		query.Set("chain", chain)
	}
	if limit != "" {
		query.Set("limit", limit)
	}
	if cursor != "" {
		query.Set("cursor", cursor)
	}

	var resp rest.Response
	var err error
	resp, _, err = c.rest().Do(ctx, rest.Options{
		Method: "GET",
		Path:   "/v5/asset/deposit/query-allowed-list",
		Query:  query,
		Signed: true,
		Meta: rest.RequestMeta{
			Category: string(bybit.RateLimitCategoryQuery),
		},
	})
	if err != nil {
		return nil, "", err
	}

	var payload struct {
		Coins          []string `json:"coins"`
		NextPageCursor string   `json:"nextPageCursor"`
	}
	if err = resp.UnmarshalResult(&payload); err != nil {
		return nil, "", bybit.NewError(bybit.ErrorKindUnknown, "", "asset.GetAllowedDepositCoins: parse", err)
	}
	return payload.Coins, payload.NextPageCursor, nil
}

// SetDepositAccount sets the default account type incoming deposits route to.
func (c *Client) SetDepositAccount(ctx context.Context, req types.SetDepositAccountRequest) error {
	if req.AccountType == "" {
		return bybit.NewError(bybit.ErrorKindInvalidRequest, "", "asset.SetDepositAccount: accountType is empty", nil)
	}

	var body map[string]any = map[string]any{
		"accountType": string(req.AccountType),
	}

	var resp rest.Response
	var err error
	resp, _, err = c.rest().Do(ctx, rest.Options{
		Method: "POST",
		Path:   "/v5/asset/deposit/deposit-to-account",
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
