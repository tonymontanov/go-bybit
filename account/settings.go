/*
FILE: account/settings.go

DESCRIPTION:
Account configuration writes — margin mode, collateral switch, spot hedging.
*/

package account

import (
	"context"

	bybit "github.com/tonymontanov/go-bybit/v2"
	"github.com/tonymontanov/go-bybit/v2/account/types"
	"github.com/tonymontanov/go-bybit/v2/internal/rest"
)

type rawSetMarginModeReason struct {
	ReasonCode string `json:"reasonCode"`
	ReasonMsg  string `json:"reasonMsg"`
}

type setMarginModePayload struct {
	Reasons []rawSetMarginModeReason `json:"reasons"`
}

// SetMarginMode switches the unified account margin mode.
func (c *Client) SetMarginMode(ctx context.Context, mode types.MarginMode) (types.SetMarginModeResult, error) {
	if mode == "" {
		return types.SetMarginModeResult{}, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "account.SetMarginMode: mode is empty", nil)
	}

	var resp rest.Response
	var err error
	resp, _, err = c.rest().Do(ctx, rest.Options{
		Method: "POST",
		Path:   "/v5/account/set-margin-mode",
		Body: map[string]any{
			"setMarginMode": string(mode),
		},
		Signed: true,
		Meta: rest.RequestMeta{
			Category: string(bybit.RateLimitCategoryQuery),
		},
	})
	if err != nil {
		return types.SetMarginModeResult{}, err
	}

	var payload setMarginModePayload
	if err = resp.UnmarshalResult(&payload); err != nil {
		return types.SetMarginModeResult{}, bybit.NewError(bybit.ErrorKindUnknown, "", "account.SetMarginMode: parse", err)
	}

	var out types.SetMarginModeResult
	var i int
	for i = 0; i < len(payload.Reasons); i++ {
		out.Reasons = append(out.Reasons, types.SetMarginModeReason{
			ReasonCode: payload.Reasons[i].ReasonCode,
			ReasonMsg:  payload.Reasons[i].ReasonMsg,
		})
	}
	return out, nil
}

// SetCollateralCoin toggles whether a coin is used as collateral in UTA.
func (c *Client) SetCollateralCoin(ctx context.Context, coin string, switchOn types.CollateralSwitch) error {
	if coin == "" {
		return bybit.NewError(bybit.ErrorKindInvalidRequest, "", "account.SetCollateralCoin: coin is empty", nil)
	}
	if switchOn == "" {
		return bybit.NewError(bybit.ErrorKindInvalidRequest, "", "account.SetCollateralCoin: collateralSwitch is empty", nil)
	}

	_, _, err := c.rest().Do(ctx, rest.Options{
		Method: "POST",
		Path:   "/v5/account/set-collateral-switch",
		Body: map[string]any{
			"coin":             coin,
			"collateralSwitch": string(switchOn),
		},
		Signed: true,
		Meta: rest.RequestMeta{
			Category: string(bybit.RateLimitCategoryQuery),
		},
	})
	return err
}

// SetHedgingMode enables or disables spot hedging on portfolio margin accounts.
func (c *Client) SetHedgingMode(ctx context.Context, mode types.HedgingMode) error {
	if mode == "" {
		return bybit.NewError(bybit.ErrorKindInvalidRequest, "", "account.SetHedgingMode: mode is empty", nil)
	}

	_, _, err := c.rest().Do(ctx, rest.Options{
		Method: "POST",
		Path:   "/v5/account/set-hedging-mode",
		Body: map[string]any{
			"setHedgingMode": string(mode),
		},
		Signed: true,
		Meta: rest.RequestMeta{
			Category: string(bybit.RateLimitCategoryQuery),
		},
	})
	return err
}
