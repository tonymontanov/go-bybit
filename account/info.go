/*
FILE: account/info.go

DESCRIPTION:
GET /v5/account/info — margin mode, UTA status, spot hedging.
*/

package account

import (
	"context"

	bybit "github.com/tonymontanov/go-bybit/v2"
	"github.com/tonymontanov/go-bybit/v2/account/types"
	"github.com/tonymontanov/go-bybit/v2/internal/rest"
)

type rawAccountInfo struct {
	UnifiedMarginStatus int    `json:"unifiedMarginStatus"`
	MarginMode          string `json:"marginMode"`
	IsMasterTrader      bool   `json:"isMasterTrader"`
	SpotHedgingStatus   string `json:"spotHedgingStatus"`
	UpdatedTime         string `json:"updatedTime"`
}

func convertAccountInfo(raw rawAccountInfo) types.AccountInfo {
	return types.AccountInfo{
		UnifiedMarginStatus: raw.UnifiedMarginStatus,
		MarginMode:          types.MarginMode(raw.MarginMode),
		IsMasterTrader:      raw.IsMasterTrader,
		SpotHedgingStatus:   types.HedgingMode(raw.SpotHedgingStatus),
		UpdatedAtMs:         ms(raw.UpdatedTime),
	}
}

// GetAccountInfo returns unified account settings (margin mode, hedging, UTA status).
func (c *Client) GetAccountInfo(ctx context.Context) (types.AccountInfo, error) {
	var resp rest.Response
	var err error
	resp, _, err = c.rest().Do(ctx, rest.Options{
		Method: "GET",
		Path:   "/v5/account/info",
		Signed: true,
		Meta: rest.RequestMeta{
			Category: string(bybit.RateLimitCategoryQuery),
		},
	})
	if err != nil {
		return types.AccountInfo{}, err
	}

	var raw rawAccountInfo
	if err = resp.UnmarshalResult(&raw); err != nil {
		return types.AccountInfo{}, bybit.NewError(bybit.ErrorKindUnknown, "", "account.GetAccountInfo: parse", err)
	}
	return convertAccountInfo(raw), nil
}
