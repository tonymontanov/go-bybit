/*
FILE: account/collateral.go

DESCRIPTION:
GET /v5/account/collateral-info — borrow rates and collateral switches.
*/

package account

import (
	"context"
	"net/url"

	bybit "github.com/tonymontanov/go-bybit/v2"
	"github.com/tonymontanov/go-bybit/v2/account/types"
	"github.com/tonymontanov/go-bybit/v2/internal/rest"
)

type rawCollateralRow struct {
	Currency            string `json:"currency"`
	HourlyBorrowRate    string `json:"hourlyBorrowRate"`
	MaxBorrowingAmount  string `json:"maxBorrowingAmount"`
	FreeBorrowingLimit  string `json:"freeBorrowingLimit"`
	FreeBorrowAmount    string `json:"freeBorrowAmount"`
	BorrowAmount        string `json:"borrowAmount"`
	OtherBorrowAmount   string `json:"otherBorrowAmount"`
	AvailableToBorrow   string `json:"availableToBorrow"`
	Borrowable          bool   `json:"borrowable"`
	BorrowUsageRate     string `json:"borrowUsageRate"`
	MarginCollateral    bool   `json:"marginCollateral"`
	CollateralSwitch    bool   `json:"collateralSwitch"`
	CollateralRatio     string `json:"collateralRatio"`
	FreeBorrowingAmount string `json:"freeBorrowingAmount"`
}

type collateralPayload struct {
	List []rawCollateralRow `json:"list"`
}

func convertCollateral(raw rawCollateralRow) types.CollateralInfo {
	return types.CollateralInfo{
		Currency:            raw.Currency,
		HourlyBorrowRate:    dec(raw.HourlyBorrowRate),
		MaxBorrowingAmount:  dec(raw.MaxBorrowingAmount),
		FreeBorrowingLimit:  dec(raw.FreeBorrowingLimit),
		FreeBorrowAmount:    dec(raw.FreeBorrowAmount),
		BorrowAmount:        dec(raw.BorrowAmount),
		OtherBorrowAmount:   dec(raw.OtherBorrowAmount),
		AvailableToBorrow:   dec(raw.AvailableToBorrow),
		Borrowable:          raw.Borrowable,
		BorrowUsageRate:     dec(raw.BorrowUsageRate),
		MarginCollateral:    raw.MarginCollateral,
		CollateralSwitch:    raw.CollateralSwitch,
		CollateralRatio:     dec(raw.CollateralRatio),
		FreeBorrowingAmount: raw.FreeBorrowingAmount,
	}
}

// GetCollateralInfo returns collateral and borrow metadata for UTA coins.
func (c *Client) GetCollateralInfo(ctx context.Context, currency string) ([]types.CollateralInfo, error) {
	var query url.Values = url.Values{}
	if currency != "" {
		query.Set("currency", currency)
	}

	var resp rest.Response
	var err error
	resp, _, err = c.rest().Do(ctx, rest.Options{
		Method: "GET",
		Path:   "/v5/account/collateral-info",
		Query:  query,
		Signed: true,
		Meta: rest.RequestMeta{
			Category: string(bybit.RateLimitCategoryQuery),
		},
	})
	if err != nil {
		return nil, err
	}

	var payload collateralPayload
	if err = resp.UnmarshalResult(&payload); err != nil {
		return nil, bybit.NewError(bybit.ErrorKindUnknown, "", "account.GetCollateralInfo: parse", err)
	}

	var out []types.CollateralInfo = make([]types.CollateralInfo, 0, len(payload.List))
	var i int
	for i = 0; i < len(payload.List); i++ {
		out = append(out, convertCollateral(payload.List[i]))
	}
	return out, nil
}
