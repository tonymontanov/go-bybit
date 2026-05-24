/*
FILE: affiliate/user-info.go

DESCRIPTION:
GET /v5/user/aff-customer-info — affiliate client detail by UID.
*/

package affiliate

import (
	"context"
	"net/url"

	bybit "github.com/tonymontanov/go-bybit/v2"
	"github.com/tonymontanov/go-bybit/v2/affiliate/types"
	"github.com/tonymontanov/go-bybit/v2/internal/rest"
)

type rawAffiliateUserInfo struct {
	UID                  string            `json:"uid"`
	VIPLevel             string            `json:"vipLevel"`
	KycLevel             int               `json:"KycLevel"`
	TakerVol30Day        string            `json:"takerVol30Day"`
	MakerVol30Day        string            `json:"makerVol30Day"`
	TradeVol30Day        string            `json:"tradeVol30Day"`
	DepositAmount30Day   string            `json:"depositAmount30Day"`
	TakerVol365Day       string            `json:"takerVol365Day"`
	MakerVol365Day       string            `json:"makerVol365Day"`
	TradeVol365Day       string            `json:"tradeVol365Day"`
	DepositAmount365Day  string            `json:"depositAmount365Day"`
	TotalWalletBalance   string            `json:"totalWalletBalance"`
	DepositUpdateTime    string            `json:"depositUpdateTime"`
	VolUpdateTime        string            `json:"volUpdateTime"`
	TradFiTradeVol30Day  string            `json:"tradfiTradeVol30Day"`
	TradFiTradeVol365Day string            `json:"tradfiTradeVol365Day"`
	Commissions30Day     map[string]string `json:"commissions30Day"`
	Commissions365Day    map[string]string `json:"commissions365Day"`
	PaySendAmount30Day   string            `json:"paySendAmount30Day"`
	PayFTT               string            `json:"payFtt"`
	CardFTT              string            `json:"cardFtt"`
}

// GetAffiliateUserInfo returns trading/deposit stats for one affiliate client.
func (c *Client) GetAffiliateUserInfo(ctx context.Context, req types.AffiliateUserInfoRequest) (types.AffiliateUserInfo, error) {
	if req.UID == "" {
		return types.AffiliateUserInfo{}, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "affiliate.GetAffiliateUserInfo: uid is empty", nil)
	}

	var query url.Values = url.Values{}
	query.Set("uid", req.UID)
	if req.Coin != "" {
		query.Set("coin", req.Coin)
	}
	if req.Business != "" {
		query.Set("business", string(req.Business))
	}

	var resp rest.Response
	var err error
	resp, _, err = c.rest().Do(ctx, rest.Options{
		Method: "GET",
		Path:   "/v5/user/aff-customer-info",
		Query:  query,
		Signed: true,
		Meta: rest.RequestMeta{
			Category: string(bybit.RateLimitCategoryQuery),
		},
	})
	if err != nil {
		return types.AffiliateUserInfo{}, err
	}

	var raw rawAffiliateUserInfo
	if err = resp.UnmarshalResult(&raw); err != nil {
		return types.AffiliateUserInfo{}, bybit.NewError(bybit.ErrorKindUnknown, "", "affiliate.GetAffiliateUserInfo: parse", err)
	}

	return types.AffiliateUserInfo{
		UID:                  raw.UID,
		VIPLevel:             raw.VIPLevel,
		KYCLevel:             raw.KycLevel,
		TakerVol30Day:        dec(raw.TakerVol30Day),
		MakerVol30Day:        dec(raw.MakerVol30Day),
		TradeVol30Day:        dec(raw.TradeVol30Day),
		DepositAmount30Day:   dec(raw.DepositAmount30Day),
		TakerVol365Day:       dec(raw.TakerVol365Day),
		MakerVol365Day:       dec(raw.MakerVol365Day),
		TradeVol365Day:       dec(raw.TradeVol365Day),
		DepositAmount365Day:  dec(raw.DepositAmount365Day),
		TotalWalletBalance:   raw.TotalWalletBalance,
		DepositUpdateTime:    raw.DepositUpdateTime,
		VolUpdateTime:        raw.VolUpdateTime,
		TradFiTradeVol30Day:  dec(raw.TradFiTradeVol30Day),
		TradFiTradeVol365Day: dec(raw.TradFiTradeVol365Day),
		Commissions30Day:     commissionMap(raw.Commissions30Day),
		Commissions365Day:    commissionMap(raw.Commissions365Day),
		PaySendAmount30Day:   dec(raw.PaySendAmount30Day),
		PayFTT:               dec(raw.PayFTT),
		CardFTT:              dec(raw.CardFTT),
	}, nil
}
