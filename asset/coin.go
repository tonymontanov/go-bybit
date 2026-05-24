/*
FILE: asset/coin.go

DESCRIPTION:
Coin metadata — GET /v5/asset/coin/query-info.
*/

package asset

import (
	"context"
	"net/url"

	bybit "github.com/tonymontanov/go-bybit/v2"
	"github.com/tonymontanov/go-bybit/v2/asset/types"
	"github.com/tonymontanov/go-bybit/v2/internal/rest"
)

type rawCoinInfoRow struct {
	Name   string           `json:"name"`
	Coin   string           `json:"coin"`
	Chains []rawCoinChainRow `json:"chains"`
}

type rawCoinChainRow struct {
	Chain                 string `json:"chain"`
	ChainType             string `json:"chainType"`
	Confirmation          string `json:"confirmation"`
	WithdrawFee           string `json:"withdrawFee"`
	DepositMin            string `json:"depositMin"`
	WithdrawMin           string `json:"withdrawMin"`
	MinAccuracy           string `json:"minAccuracy"`
	ChainDeposit          string `json:"chainDeposit"`
	ChainWithdraw         string `json:"chainWithdraw"`
	WithdrawPercentageFee string `json:"withdrawPercentageFee"`
	ContractAddress       string `json:"contractAddress"`
	SafeConfirmNumber     string `json:"safeConfirmNumber"`
	WithdrawMax           string `json:"withdrawMax"`
}

type coinInfoPayload struct {
	Rows []rawCoinInfoRow `json:"rows"`
}

func convertCoinInfo(row rawCoinInfoRow) types.CoinInfo {
	var out types.CoinInfo = types.CoinInfo{
		Name: row.Name,
		Coin: row.Coin,
	}
	for _, ch := range row.Chains {
		out.Chains = append(out.Chains, types.CoinChainInfo{
			Chain:                 ch.Chain,
			ChainType:             ch.ChainType,
			Confirmation:          ch.Confirmation,
			WithdrawFee:           dec(ch.WithdrawFee),
			DepositMin:            dec(ch.DepositMin),
			WithdrawMin:           dec(ch.WithdrawMin),
			MinAccuracy:           ch.MinAccuracy,
			ChainDeposit:          ch.ChainDeposit,
			ChainWithdraw:         ch.ChainWithdraw,
			WithdrawPercentageFee: dec(ch.WithdrawPercentageFee),
			ContractAddress:       ch.ContractAddress,
			SafeConfirmNumber:     ch.SafeConfirmNumber,
			WithdrawMax:           dec(ch.WithdrawMax),
		})
	}
	return out
}

// GetCoinInfo returns coin metadata and per-chain deposit/withdraw
// constraints. Coin filter is optional — empty returns all coins.
func (c *Client) GetCoinInfo(ctx context.Context, coin string) ([]types.CoinInfo, error) {
	var query url.Values = url.Values{}
	if coin != "" {
		query.Set("coin", coin)
	}

	var resp rest.Response
	var err error
	resp, _, err = c.rest().Do(ctx, rest.Options{
		Method: "GET",
		Path:   "/v5/asset/coin/query-info",
		Query:  query,
		Signed: true,
		Meta: rest.RequestMeta{
			Category: string(bybit.RateLimitCategoryQuery),
		},
	})
	if err != nil {
		return nil, err
	}

	var payload coinInfoPayload
	if err = resp.UnmarshalResult(&payload); err != nil {
		return nil, bybit.NewError(bybit.ErrorKindUnknown, "", "asset.GetCoinInfo: parse", err)
	}

	var out []types.CoinInfo = make([]types.CoinInfo, 0, len(payload.Rows))
	for _, row := range payload.Rows {
		out = append(out, convertCoinInfo(row))
	}
	return out, nil
}
