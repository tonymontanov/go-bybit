/*
FILE: broker/account.go

DESCRIPTION:
GET /v5/broker/account-info — broker master account quotas and rebate rates.
*/

package broker

import (
	"context"

	bybit "github.com/tonymontanov/go-bybit/v2"
	"github.com/tonymontanov/go-bybit/v2/broker/types"
	"github.com/tonymontanov/go-bybit/v2/internal/rest"
)

type rawFeeRebateRate struct {
	Spot        string `json:"spot"`
	Derivatives string `json:"derivatives"`
	Convert     string `json:"convert"`
}

type rawBrokerAccountInfo struct {
	SubAcctQty          string           `json:"subAcctQty"`
	MaxSubAcctQty       string           `json:"maxSubAcctQty"`
	BaseFeeRebateRate   rawFeeRebateRate `json:"baseFeeRebateRate"`
	MarkupFeeRebateRate rawFeeRebateRate `json:"markupFeeRebateRate"`
	Ts                  string           `json:"ts"`
}

// GetAccountInfo returns broker master account configuration.
func (c *Client) GetAccountInfo(ctx context.Context) (types.AccountInfo, error) {
	var resp rest.Response
	var err error
	resp, _, err = c.rest().Do(ctx, rest.Options{
		Method: "GET",
		Path:   "/v5/broker/account-info",
		Signed: true,
		Meta: rest.RequestMeta{
			Category: string(bybit.RateLimitCategoryQuery),
		},
	})
	if err != nil {
		return types.AccountInfo{}, err
	}

	var raw rawBrokerAccountInfo
	if err = resp.UnmarshalResult(&raw); err != nil {
		return types.AccountInfo{}, bybit.NewError(bybit.ErrorKindUnknown, "", "broker.GetAccountInfo: parse", err)
	}

	return types.AccountInfo{
		SubAccountQty:    raw.SubAcctQty,
		MaxSubAccountQty: raw.MaxSubAcctQty,
		BaseFeeRebateRate: types.FeeRebateRate{
			Spot:        raw.BaseFeeRebateRate.Spot,
			Derivatives: raw.BaseFeeRebateRate.Derivatives,
			Convert:     raw.BaseFeeRebateRate.Convert,
		},
		MarkupFeeRebateRate: types.FeeRebateRate{
			Spot:        raw.MarkupFeeRebateRate.Spot,
			Derivatives: raw.MarkupFeeRebateRate.Derivatives,
			Convert:     raw.MarkupFeeRebateRate.Convert,
		},
		TimestampMs: ms(raw.Ts),
	}, nil
}
