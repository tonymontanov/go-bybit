/*
FILE: asset/transfer.go

DESCRIPTION:
Internal transfers and account-coin balance queries under /v5/asset/transfer/*.
*/

package asset

import (
	"context"
	"net/url"
	"strconv"

	bybit "github.com/tonymontanov/go-bybit/v2"
	"github.com/tonymontanov/go-bybit/v2/asset/types"
	"github.com/tonymontanov/go-bybit/v2/internal/rest"
	commontypes "github.com/tonymontanov/go-bybit/v2/types"
)

type rawAccountCoinBalance struct {
	Coin            string `json:"coin"`
	WalletBalance   string `json:"walletBalance"`
	TransferBalance string `json:"transferBalance"`
	Bonus           string `json:"bonus"`
}

type allCoinsBalancePayload struct {
	Balance []rawAccountCoinBalance `json:"balance"`
}

type singleCoinBalancePayload struct {
	AccountType string                `json:"accountType"`
	Balance     rawAccountCoinBalance `json:"balance"`
}

type transferableCoinPayload struct {
	Coins []string `json:"coins"`
}

type rawInternalTransferResult struct {
	TransferID string `json:"transferId"`
	Status     string `json:"status"`
}

type rawInternalTransferRecord struct {
	TransferID      string `json:"transferId"`
	Coin            string `json:"coin"`
	Amount          string `json:"amount"`
	FromAccountType string `json:"fromAccountType"`
	ToAccountType   string `json:"toAccountType"`
	Timestamp       string `json:"timestamp"`
	Status          string `json:"status"`
}

type internalTransferListPayload struct {
	List           []rawInternalTransferRecord `json:"list"`
	NextPageCursor string                      `json:"nextPageCursor"`
}

func convertAccountCoinBalance(raw rawAccountCoinBalance) types.AccountCoinBalance {
	return types.AccountCoinBalance{
		Coin:            raw.Coin,
		WalletBalance:   dec(raw.WalletBalance),
		TransferBalance: dec(raw.TransferBalance),
		Bonus:           dec(raw.Bonus),
	}
}

// GetAllCoinsBalance returns per-coin balances for the given account type.
func (c *Client) GetAllCoinsBalance(ctx context.Context, accountType commontypes.AccountType, coin string) ([]types.AccountCoinBalance, error) {
	if accountType == "" {
		return nil, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "asset.GetAllCoinsBalance: accountType is empty", nil)
	}
	var query url.Values = url.Values{}
	query.Set("accountType", string(accountType))
	if coin != "" {
		query.Set("coin", coin)
	}

	var resp rest.Response
	var err error
	resp, _, err = c.rest().Do(ctx, rest.Options{
		Method: "GET",
		Path:   "/v5/asset/transfer/query-account-coins-balance",
		Query:  query,
		Signed: true,
		Meta: rest.RequestMeta{
			Category: string(bybit.RateLimitCategoryQuery),
		},
	})
	if err != nil {
		return nil, err
	}

	var payload allCoinsBalancePayload
	if err = resp.UnmarshalResult(&payload); err != nil {
		return nil, bybit.NewError(bybit.ErrorKindUnknown, "", "asset.GetAllCoinsBalance: parse", err)
	}

	var out []types.AccountCoinBalance = make([]types.AccountCoinBalance, 0, len(payload.Balance))
	for _, row := range payload.Balance {
		out = append(out, convertAccountCoinBalance(row))
	}
	return out, nil
}

// GetSingleCoinBalance returns the balance of one coin within an account type.
func (c *Client) GetSingleCoinBalance(ctx context.Context, accountType commontypes.AccountType, coin string, memberID string) (types.AccountCoinBalance, error) {
	var out types.AccountCoinBalance
	if accountType == "" {
		return out, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "asset.GetSingleCoinBalance: accountType is empty", nil)
	}
	if coin == "" {
		return out, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "asset.GetSingleCoinBalance: coin is empty", nil)
	}

	var query url.Values = url.Values{}
	query.Set("accountType", string(accountType))
	query.Set("coin", coin)
	if memberID != "" {
		query.Set("memberId", memberID)
	}

	var resp rest.Response
	var err error
	resp, _, err = c.rest().Do(ctx, rest.Options{
		Method: "GET",
		Path:   "/v5/asset/transfer/query-account-coin-balance",
		Query:  query,
		Signed: true,
		Meta: rest.RequestMeta{
			Category: string(bybit.RateLimitCategoryQuery),
		},
	})
	if err != nil {
		return out, err
	}

	var payload singleCoinBalancePayload
	if err = resp.UnmarshalResult(&payload); err != nil {
		return out, bybit.NewError(bybit.ErrorKindUnknown, "", "asset.GetSingleCoinBalance: parse", err)
	}
	return convertAccountCoinBalance(payload.Balance), nil
}

// GetTransferableCoins returns coins that can be transferred between the
// given account types under the same UID.
func (c *Client) GetTransferableCoins(ctx context.Context, fromAccountType, toAccountType commontypes.AccountType) ([]string, error) {
	if fromAccountType == "" || toAccountType == "" {
		return nil, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "asset.GetTransferableCoins: fromAccountType and toAccountType are required", nil)
	}

	var query url.Values = url.Values{}
	query.Set("fromAccountType", string(fromAccountType))
	query.Set("toAccountType", string(toAccountType))

	var resp rest.Response
	var err error
	resp, _, err = c.rest().Do(ctx, rest.Options{
		Method: "GET",
		Path:   "/v5/asset/transfer/query-transfer-coin-list",
		Query:  query,
		Signed: true,
		Meta: rest.RequestMeta{
			Category: string(bybit.RateLimitCategoryQuery),
		},
	})
	if err != nil {
		return nil, err
	}

	var payload transferableCoinPayload
	if err = resp.UnmarshalResult(&payload); err != nil {
		return nil, bybit.NewError(bybit.ErrorKindUnknown, "", "asset.GetTransferableCoins: parse", err)
	}
	return payload.Coins, nil
}

// CreateInternalTransfer moves funds between account types under the same UID.
func (c *Client) CreateInternalTransfer(ctx context.Context, req types.CreateInternalTransferRequest) (types.InternalTransferResult, error) {
	var out types.InternalTransferResult
	if req.TransferID == "" {
		return out, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "asset.CreateInternalTransfer: transferId is empty", nil)
	}
	if req.Coin == "" {
		return out, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "asset.CreateInternalTransfer: coin is empty", nil)
	}
	if !req.Amount.IsPositive() {
		return out, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "asset.CreateInternalTransfer: amount must be positive", nil)
	}
	if req.FromAccountType == "" || req.ToAccountType == "" {
		return out, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "asset.CreateInternalTransfer: fromAccountType and toAccountType are required", nil)
	}

	var body map[string]any = map[string]any{
		"transferId":      req.TransferID,
		"coin":            req.Coin,
		"amount":          req.Amount.String(),
		"fromAccountType": string(req.FromAccountType),
		"toAccountType":   string(req.ToAccountType),
	}

	var resp rest.Response
	var err error
	resp, _, err = c.rest().Do(ctx, rest.Options{
		Method: "POST",
		Path:   "/v5/asset/transfer/inter-transfer",
		Body:   body,
		Signed: true,
		Meta: rest.RequestMeta{
			Category: string(bybit.RateLimitCategoryQuery),
		},
	})
	if err != nil {
		return out, err
	}

	var payload rawInternalTransferResult
	if err = resp.UnmarshalResult(&payload); err != nil {
		return out, bybit.NewError(bybit.ErrorKindUnknown, "", "asset.CreateInternalTransfer: parse", err)
	}
	return types.InternalTransferResult{
		TransferID: payload.TransferID,
		Status:     types.TransferStatus(payload.Status),
	}, nil
}

// InternalTransferListRequest — filters for GetInternalTransferRecords.
type InternalTransferListRequest struct {
	TransferID string
	Coin       string
	Status     types.TransferStatus
	StartTimeMs int64
	EndTimeMs   int64
	Limit       int
	Cursor      string
}

// GetInternalTransferRecords returns paginated internal transfer history.
func (c *Client) GetInternalTransferRecords(ctx context.Context, req InternalTransferListRequest) (types.InternalTransferList, error) {
	var out types.InternalTransferList
	var query url.Values = url.Values{}
	if req.TransferID != "" {
		query.Set("transferId", req.TransferID)
	}
	if req.Coin != "" {
		query.Set("coin", req.Coin)
	}
	if req.Status != "" {
		query.Set("status", string(req.Status))
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
		Path:   "/v5/asset/transfer/query-inter-transfer-list",
		Query:  query,
		Signed: true,
		Meta: rest.RequestMeta{
			Category: string(bybit.RateLimitCategoryQuery),
		},
	})
	if err != nil {
		return out, err
	}

	var payload internalTransferListPayload
	if err = resp.UnmarshalResult(&payload); err != nil {
		return out, bybit.NewError(bybit.ErrorKindUnknown, "", "asset.GetInternalTransferRecords: parse", err)
	}

	out.NextPageCursor = payload.NextPageCursor
	out.Records = make([]types.InternalTransferRecord, 0, len(payload.List))
	for _, row := range payload.List {
		out.Records = append(out.Records, types.InternalTransferRecord{
			TransferID:      row.TransferID,
			Coin:            row.Coin,
			Amount:          dec(row.Amount),
			FromAccountType: commontypes.AccountType(row.FromAccountType),
			ToAccountType:   commontypes.AccountType(row.ToAccountType),
			TimestampMs:     ms(row.Timestamp),
			Status:          types.TransferStatus(row.Status),
		})
	}
	return out, nil
}
