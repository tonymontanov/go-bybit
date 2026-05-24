/*
FILE: affiliate/user-list.go

DESCRIPTION:
GET /v5/affiliate/aff-user-list — paginated affiliate client list.
*/

package affiliate

import (
	"context"
	"net/url"
	"strconv"

	bybit "github.com/tonymontanov/go-bybit/v2"
	"github.com/tonymontanov/go-bybit/v2/affiliate/types"
	"github.com/tonymontanov/go-bybit/v2/internal/rest"
)

type rawAffiliateUser struct {
	UserID               string            `json:"userId"`
	RegisterTime         string            `json:"registerTime"`
	Source               string            `json:"source"`
	Remarks              string            `json:"remarks"`
	IsKYC                bool              `json:"isKyc"`
	TakerVol30Day        string            `json:"takerVol30Day"`
	MakerVol30Day        string            `json:"makerVol30Day"`
	TradeVol30Day        string            `json:"tradeVol30Day"`
	DepositAmount30Day   string            `json:"depositAmount30Day"`
	TakerVol365Day       string            `json:"takerVol365Day"`
	MakerVol365Day       string            `json:"makerVol365Day"`
	TradeVol365Day       string            `json:"tradeVol365Day"`
	DepositAmount365Day  string            `json:"depositAmount365Day"`
	TakerVol             string            `json:"takerVol"`
	MakerVol             string            `json:"makerVol"`
	TradeVol             string            `json:"tradeVol"`
	StartDate            string            `json:"startDate"`
	EndDate              string            `json:"endDate"`
	TradFiTradeVol       string            `json:"tradfiTradeVol"`
	TradFiTradeVol30Day  string            `json:"tradfiTradeVol30Day"`
	TradFiTradeVol365Day string            `json:"tradfiTradeVol365Day"`
	CommissionsVol       map[string]string `json:"commissionsVol"`
	Commissions30Day     map[string]string `json:"commissions30Day"`
	Commissions365Day    map[string]string `json:"commissions365Day"`
}

type affiliateUserListPayload struct {
	List           []rawAffiliateUser `json:"list"`
	NextPageCursor string             `json:"nextPageCursor"`
}

func convertAffiliateUser(raw rawAffiliateUser) types.AffiliateUser {
	return types.AffiliateUser{
		UserID:               raw.UserID,
		RegisterTime:         raw.RegisterTime,
		Source:               raw.Source,
		Remarks:              raw.Remarks,
		IsKYC:                raw.IsKYC,
		TakerVol30Day:        dec(raw.TakerVol30Day),
		MakerVol30Day:        dec(raw.MakerVol30Day),
		TradeVol30Day:        dec(raw.TradeVol30Day),
		DepositAmount30Day:   dec(raw.DepositAmount30Day),
		TakerVol365Day:       dec(raw.TakerVol365Day),
		MakerVol365Day:       dec(raw.MakerVol365Day),
		TradeVol365Day:       dec(raw.TradeVol365Day),
		DepositAmount365Day:  dec(raw.DepositAmount365Day),
		TakerVol:             dec(raw.TakerVol),
		MakerVol:             dec(raw.MakerVol),
		TradeVol:             dec(raw.TradeVol),
		StartDate:            raw.StartDate,
		EndDate:              raw.EndDate,
		TradFiTradeVol:       dec(raw.TradFiTradeVol),
		TradFiTradeVol30Day:  dec(raw.TradFiTradeVol30Day),
		TradFiTradeVol365Day: dec(raw.TradFiTradeVol365Day),
		CommissionsVol:       commissionMap(raw.CommissionsVol),
		Commissions30Day:     commissionMap(raw.Commissions30Day),
		Commissions365Day:    commissionMap(raw.Commissions365Day),
	}
}

// GetAffiliateUserList returns paginated affiliate client rows.
func (c *Client) GetAffiliateUserList(ctx context.Context, req types.AffiliateUserListRequest) (types.AffiliateUserList, error) {
	if (req.StartDate != "" && req.EndDate == "") || (req.StartDate == "" && req.EndDate != "") {
		return types.AffiliateUserList{}, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "affiliate.GetAffiliateUserList: startDate and endDate must be supplied together", nil)
	}

	var query url.Values = url.Values{}
	if req.Size > 0 {
		if req.Size > 100 {
			req.Size = 100
		}
		query.Set("size", strconv.Itoa(req.Size))
	}
	if req.Cursor != "" {
		query.Set("cursor", req.Cursor)
	}
	if req.NeedDeposit {
		query.Set("needDeposit", "true")
	}
	if req.Need30 {
		query.Set("need30", "true")
	}
	if req.Need365 {
		query.Set("need365", "true")
	}
	if req.StartDate != "" {
		query.Set("startDate", req.StartDate)
	}
	if req.EndDate != "" {
		query.Set("endDate", req.EndDate)
	}

	var resp rest.Response
	var err error
	resp, _, err = c.rest().Do(ctx, rest.Options{
		Method: "GET",
		Path:   "/v5/affiliate/aff-user-list",
		Query:  query,
		Signed: true,
		Meta: rest.RequestMeta{
			Category: string(bybit.RateLimitCategoryQuery),
		},
	})
	if err != nil {
		return types.AffiliateUserList{}, err
	}

	var payload affiliateUserListPayload
	if err = resp.UnmarshalResult(&payload); err != nil {
		return types.AffiliateUserList{}, bybit.NewError(bybit.ErrorKindUnknown, "", "affiliate.GetAffiliateUserList: parse", err)
	}

	var out types.AffiliateUserList
	out.NextPageCursor = payload.NextPageCursor
	out.Users = make([]types.AffiliateUser, 0, len(payload.List))
	var i int
	for i = 0; i < len(payload.List); i++ {
		out.Users = append(out.Users, convertAffiliateUser(payload.List[i]))
	}
	return out, nil
}
