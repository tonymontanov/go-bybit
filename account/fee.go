/*
FILE: account/fee.go

DESCRIPTION:
GET /v5/account/fee-rate — maker/taker fee schedule.
*/

package account

import (
	"context"
	"net/url"

	bybit "github.com/tonymontanov/go-bybit/v2"
	"github.com/tonymontanov/go-bybit/v2/account/types"
	"github.com/tonymontanov/go-bybit/v2/internal/rest"
)

type rawFeeRateRow struct {
	Symbol       string `json:"symbol"`
	BaseCoin     string `json:"baseCoin"`
	TakerFeeRate string `json:"takerFeeRate"`
	MakerFeeRate string `json:"makerFeeRate"`
}

type feeRatePayload struct {
	Category string          `json:"category"`
	List     []rawFeeRateRow `json:"list"`
}

// GetFeeRate returns trading fee rates for the requested category.
func (c *Client) GetFeeRate(ctx context.Context, req types.FeeRateRequest) (types.FeeRateList, error) {
	if req.Category == "" {
		return types.FeeRateList{}, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "account.GetFeeRate: category is empty", nil)
	}

	var query url.Values = url.Values{}
	query.Set("category", string(req.Category))
	if req.Symbol != "" {
		query.Set("symbol", req.Symbol)
	}
	if req.BaseCoin != "" {
		query.Set("baseCoin", req.BaseCoin)
	}

	var resp rest.Response
	var err error
	resp, _, err = c.rest().Do(ctx, rest.Options{
		Method: "GET",
		Path:   "/v5/account/fee-rate",
		Query:  query,
		Signed: true,
		Meta: rest.RequestMeta{
			Category: string(bybit.RateLimitCategoryQuery),
		},
	})
	if err != nil {
		return types.FeeRateList{}, err
	}

	var payload feeRatePayload
	if err = resp.UnmarshalResult(&payload); err != nil {
		return types.FeeRateList{}, bybit.NewError(bybit.ErrorKindUnknown, "", "account.GetFeeRate: parse", err)
	}

	var out types.FeeRateList
	var i int
	for i = 0; i < len(payload.List); i++ {
		var row = payload.List[i]
		out.List = append(out.List, types.FeeRate{
			Category:     req.Category,
			Symbol:       row.Symbol,
			BaseCoin:     row.BaseCoin,
			TakerFeeRate: dec(row.TakerFeeRate),
			MakerFeeRate: dec(row.MakerFeeRate),
		})
	}
	return out, nil
}
