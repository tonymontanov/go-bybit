/*
FILE: broker/earning.go

DESCRIPTION:
GET /v5/broker/earnings-info — exchange broker rebate earnings.
*/

package broker

import (
	"context"
	"net/url"
	"strconv"

	bybit "github.com/tonymontanov/go-bybit/v2"
	"github.com/tonymontanov/go-bybit/v2/broker/types"
	"github.com/tonymontanov/go-bybit/v2/internal/rest"
)

type rawCoinEarning struct {
	Coin    string `json:"coin"`
	Earning string `json:"earning"`
}

type rawEarningsCategoryTotals struct {
	Spot        []rawCoinEarning `json:"spot"`
	Derivatives []rawCoinEarning `json:"derivatives"`
	Options     []rawCoinEarning `json:"options"`
	Convert     []rawCoinEarning `json:"convert"`
	Total       []rawCoinEarning `json:"total"`
}

type rawEarningDetail struct {
	UserID         string `json:"userId"`
	BizType        string `json:"bizType"`
	Symbol         string `json:"symbol"`
	Coin           string `json:"coin"`
	Earning        string `json:"earning"`
	MarkupEarning  string `json:"markupEarning"`
	BaseFeeEarning string `json:"baseFeeEarning"`
	OrderID        string `json:"orderId"`
	ExecID         string `json:"execId"`
	ExecTime       string `json:"execTime"`
}

type earningsPayload struct {
	TotalEarningCat rawEarningsCategoryTotals `json:"totalEarningCat"`
	Details         []rawEarningDetail          `json:"details"`
	NextPageCursor  string                      `json:"nextPageCursor"`
}

func convertCoinEarnings(rows []rawCoinEarning) []types.CoinEarning {
	var out []types.CoinEarning = make([]types.CoinEarning, 0, len(rows))
	var i int
	for i = 0; i < len(rows); i++ {
		out = append(out, types.CoinEarning{
			Coin:    rows[i].Coin,
			Earning: dec(rows[i].Earning),
		})
	}
	return out
}

// GetEarnings returns broker rebate earnings for sub-accounts.
//
// Bybit requires begin and end to be supplied together or omitted entirely.
func (c *Client) GetEarnings(ctx context.Context, req types.EarningsRequest) (types.EarningsList, error) {
	if (req.Begin != "" && req.End == "") || (req.Begin == "" && req.End != "") {
		return types.EarningsList{}, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "broker.GetEarnings: begin and end must be supplied together", nil)
	}

	var limit int = req.Limit
	if limit <= 0 {
		limit = 1000
	}
	if limit > 1000 {
		limit = 1000
	}

	var query url.Values = url.Values{}
	if req.BizType != "" {
		query.Set("bizType", string(req.BizType))
	}
	if req.Begin != "" {
		query.Set("begin", req.Begin)
	}
	if req.End != "" {
		query.Set("end", req.End)
	}
	if req.UID != "" {
		query.Set("uid", req.UID)
	}
	query.Set("limit", strconv.Itoa(limit))
	if req.Cursor != "" {
		query.Set("cursor", req.Cursor)
	}

	var resp rest.Response
	var err error
	resp, _, err = c.rest().Do(ctx, rest.Options{
		Method: "GET",
		Path:   "/v5/broker/earnings-info",
		Query:  query,
		Signed: true,
		Meta: rest.RequestMeta{
			Category: string(bybit.RateLimitCategoryQuery),
		},
	})
	if err != nil {
		return types.EarningsList{}, err
	}

	var payload earningsPayload
	if err = resp.UnmarshalResult(&payload); err != nil {
		return types.EarningsList{}, bybit.NewError(bybit.ErrorKindUnknown, "", "broker.GetEarnings: parse", err)
	}

	var out types.EarningsList
	out.CategoryTotals = types.EarningsCategoryTotals{
		Spot:        convertCoinEarnings(payload.TotalEarningCat.Spot),
		Derivatives: convertCoinEarnings(payload.TotalEarningCat.Derivatives),
		Options:     convertCoinEarnings(payload.TotalEarningCat.Options),
		Convert:     convertCoinEarnings(payload.TotalEarningCat.Convert),
		Total:       convertCoinEarnings(payload.TotalEarningCat.Total),
	}
	out.NextPageCursor = payload.NextPageCursor
	out.Details = make([]types.EarningDetail, 0, len(payload.Details))
	var i int
	for i = 0; i < len(payload.Details); i++ {
		var row = payload.Details[i]
		out.Details = append(out.Details, types.EarningDetail{
			UserID:         row.UserID,
			BizType:        types.BizType(row.BizType),
			Symbol:         row.Symbol,
			Coin:           row.Coin,
			Earning:        dec(row.Earning),
			MarkupEarning:  dec(row.MarkupEarning),
			BaseFeeEarning: dec(row.BaseFeeEarning),
			OrderID:        row.OrderID,
			ExecID:         row.ExecID,
			ExecTimeMs:     ms(row.ExecTime),
		})
	}
	return out, nil
}
