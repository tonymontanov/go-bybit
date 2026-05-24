/*
FILE: account/greeks.go

DESCRIPTION:
GET /v5/asset/coin-greeks — account greeks by base coin.
*/

package account

import (
	"context"
	"net/url"

	bybit "github.com/tonymontanov/go-bybit/v2"
	"github.com/tonymontanov/go-bybit/v2/account/types"
	"github.com/tonymontanov/go-bybit/v2/internal/rest"
)

type rawCoinGreeksRow struct {
	BaseCoin   string `json:"baseCoin"`
	TotalDelta string `json:"totalDelta"`
	TotalGamma string `json:"totalGamma"`
	TotalVega  string `json:"totalVega"`
	TotalTheta string `json:"totalTheta"`
}

type coinGreeksPayload struct {
	List []rawCoinGreeksRow `json:"list"`
}

func convertCoinGreeks(raw rawCoinGreeksRow) types.CoinGreeks {
	return types.CoinGreeks{
		BaseCoin:   raw.BaseCoin,
		TotalDelta: dec(raw.TotalDelta),
		TotalGamma: dec(raw.TotalGamma),
		TotalVega:  dec(raw.TotalVega),
		TotalTheta: dec(raw.TotalTheta),
	}
}

// GetCoinGreeks returns account greeks for one or all base coins.
func (c *Client) GetCoinGreeks(ctx context.Context, baseCoin string) ([]types.CoinGreeks, error) {
	var query url.Values = url.Values{}
	if baseCoin != "" {
		query.Set("baseCoin", baseCoin)
	}

	var resp rest.Response
	var err error
	resp, _, err = c.rest().Do(ctx, rest.Options{
		Method: "GET",
		Path:   "/v5/asset/coin-greeks",
		Query:  query,
		Signed: true,
		Meta: rest.RequestMeta{
			Category: string(bybit.RateLimitCategoryQuery),
		},
	})
	if err != nil {
		return nil, err
	}

	var payload coinGreeksPayload
	if err = resp.UnmarshalResult(&payload); err != nil {
		return nil, bybit.NewError(bybit.ErrorKindUnknown, "", "account.GetCoinGreeks: parse", err)
	}

	var out []types.CoinGreeks = make([]types.CoinGreeks, 0, len(payload.List))
	var i int
	for i = 0; i < len(payload.List); i++ {
		out = append(out, convertCoinGreeks(payload.List[i]))
	}
	return out, nil
}
