/*
FILE: linears/market-extended.go

DESCRIPTION:
Extended public market-data endpoints for the linear category (C3):
  - GetFundingRateHistory : GET /v5/market/funding/history
  - GetOpenInterest       : GET /v5/market/open-interest
*/

package linears

import (
	"context"
	"net/url"
	"strconv"

	bybit "github.com/tonymontanov/go-bybit/v2"
	"github.com/tonymontanov/go-bybit/v2/internal/rest"
	"github.com/tonymontanov/go-bybit/v2/linears/types"
)

type rawFundingRateRow struct {
	Symbol               string `json:"symbol"`
	FundingRate          string `json:"fundingRate"`
	FundingRateTimestamp string `json:"fundingRateTimestamp"`
}

type fundingRateHistoryPayload struct {
	Category string              `json:"category"`
	List     []rawFundingRateRow `json:"list"`
}

type rawOpenInterestRow struct {
	OpenInterest string `json:"openInterest"`
	Timestamp    string `json:"timestamp"`
}

type openInterestPayload struct {
	Category       string               `json:"category"`
	Symbol         string               `json:"symbol"`
	List           []rawOpenInterestRow `json:"list"`
	NextPageCursor string               `json:"nextPageCursor"`
}

// GetFundingRateHistory returns settled funding rates for a symbol.
//
// Bybit rejects requests with only startTime and no endTime (retCode 10001).
// The SDK mirrors that constraint client-side.
func (m *MarketDataClient) GetFundingRateHistory(ctx context.Context, req types.FundingRateHistoryRequest) (types.FundingRateHistory, error) {
	var out types.FundingRateHistory
	if req.Symbol == "" {
		return out, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "marketdata.GetFundingRateHistory: symbol is empty", nil)
	}
	if req.StartMs > 0 && req.EndMs == 0 {
		return out, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "marketdata.GetFundingRateHistory: endTime is required when startTime is set", nil)
	}

	var limit int = req.Limit
	if limit <= 0 {
		limit = 200
	}
	if limit > 200 {
		limit = 200
	}

	var query url.Values = url.Values{}
	query.Set("category", string(types.CategoryLinear))
	query.Set("symbol", req.Symbol)
	query.Set("limit", strconv.Itoa(limit))
	if req.StartMs > 0 {
		query.Set("startTime", strconv.FormatInt(req.StartMs, 10))
	}
	if req.EndMs > 0 {
		query.Set("endTime", strconv.FormatInt(req.EndMs, 10))
	}

	var resp rest.Response
	var err error
	resp, _, err = m.c.rest().Do(ctx, rest.Options{
		Method: "GET",
		Path:   "/v5/market/funding/history",
		Query:  query,
		Signed: false,
		Meta: rest.RequestMeta{
			Symbols:  []string{req.Symbol},
			Category: string(bybit.RateLimitCategoryMarketData),
		},
	})
	if err != nil {
		return out, err
	}

	var payload fundingRateHistoryPayload
	if err = resp.UnmarshalResult(&payload); err != nil {
		return out, bybit.NewError(bybit.ErrorKindUnknown, "", "marketdata.GetFundingRateHistory: parse", err)
	}

	out.Records = make([]types.FundingRateRecord, 0, len(payload.List))
	var i int
	for i = 0; i < len(payload.List); i++ {
		var row = payload.List[i]
		out.Records = append(out.Records, types.FundingRateRecord{
			Symbol:      row.Symbol,
			FundingRate: dec(row.FundingRate),
			TimestampMs: ms(row.FundingRateTimestamp),
		})
	}
	return out, nil
}

// GetOpenInterest returns paginated open-interest history for a symbol.
func (m *MarketDataClient) GetOpenInterest(ctx context.Context, req types.OpenInterestRequest) (types.OpenInterestHistory, error) {
	var out types.OpenInterestHistory
	if req.Symbol == "" {
		return out, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "marketdata.GetOpenInterest: symbol is empty", nil)
	}
	if req.IntervalTime == "" {
		return out, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "marketdata.GetOpenInterest: intervalTime is empty", nil)
	}

	var limit int = req.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	var query url.Values = url.Values{}
	query.Set("category", string(types.CategoryLinear))
	query.Set("symbol", req.Symbol)
	query.Set("intervalTime", string(req.IntervalTime))
	query.Set("limit", strconv.Itoa(limit))
	if req.StartMs > 0 {
		query.Set("startTime", strconv.FormatInt(req.StartMs, 10))
	}
	if req.EndMs > 0 {
		query.Set("endTime", strconv.FormatInt(req.EndMs, 10))
	}
	if req.Cursor != "" {
		query.Set("cursor", req.Cursor)
	}

	var resp rest.Response
	var err error
	resp, _, err = m.c.rest().Do(ctx, rest.Options{
		Method: "GET",
		Path:   "/v5/market/open-interest",
		Query:  query,
		Signed: false,
		Meta: rest.RequestMeta{
			Symbols:  []string{req.Symbol},
			Category: string(bybit.RateLimitCategoryMarketData),
		},
	})
	if err != nil {
		return out, err
	}

	var payload openInterestPayload
	if err = resp.UnmarshalResult(&payload); err != nil {
		return out, bybit.NewError(bybit.ErrorKindUnknown, "", "marketdata.GetOpenInterest: parse", err)
	}

	out.Symbol = payload.Symbol
	if out.Symbol == "" {
		out.Symbol = req.Symbol
	}
	out.NextPageCursor = payload.NextPageCursor
	out.Records = make([]types.OpenInterestRecord, 0, len(payload.List))
	var i int
	for i = 0; i < len(payload.List); i++ {
		var row = payload.List[i]
		out.Records = append(out.Records, types.OpenInterestRecord{
			OpenInterest: dec(row.OpenInterest),
			TimestampMs:  ms(row.Timestamp),
		})
	}
	return out, nil
}
