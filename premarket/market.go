/*
FILE: premarket/market.go

DESCRIPTION:
Public REST for Bybit V5 pre-market perpetual contracts (C6):
  - GetInstruments         : GET /v5/market/instruments-info
  - GetPreLaunchInstruments: convenience wrapper with status=PreLaunch
  - GetRiskLimit           : GET /v5/market/risk-limit
  - GetTickers             : GET /v5/market/tickers

Pre-market orders are placed via linears.Trading().CreateOrder — there is
no separate trading endpoint.
*/

package premarket

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"

	"github.com/shopspring/decimal"

	bybit "github.com/tonymontanov/go-bybit/v2"
	"github.com/tonymontanov/go-bybit/v2/internal/rest"
	"github.com/tonymontanov/go-bybit/v2/premarket/types"
	commontypes "github.com/tonymontanov/go-bybit/v2/types"
)

func validateContractCategory(cat commontypes.Category) error {
	if cat != commontypes.CategoryLinear && cat != commontypes.CategoryInverse {
		return bybit.NewError(bybit.ErrorKindInvalidRequest, "", "premarket: category must be linear or inverse", nil)
	}
	return nil
}

// GetInstruments returns paginated instrument specs for linear/inverse.
func (c *Client) GetInstruments(ctx context.Context, req types.InstrumentsRequest) (types.InstrumentList, error) {
	var out types.InstrumentList
	if err := validateContractCategory(req.Category); err != nil {
		return out, err
	}

	var limit int = req.Limit
	if limit <= 0 {
		limit = 500
	}
	if limit > 1000 {
		limit = 1000
	}

	var query url.Values = url.Values{}
	query.Set("category", string(req.Category))
	if req.Symbol != "" {
		query.Set("symbol", req.Symbol)
	}
	if req.Status != "" {
		query.Set("status", string(req.Status))
	}
	if req.BaseCoin != "" {
		query.Set("baseCoin", req.BaseCoin)
	}
	query.Set("limit", strconv.Itoa(limit))
	if req.Cursor != "" {
		query.Set("cursor", req.Cursor)
	}

	var resp rest.Response
	var err error
	resp, _, err = c.rest().Do(ctx, rest.Options{
		Method: "GET",
		Path:   "/v5/market/instruments-info",
		Query:  query,
		Signed: false,
		Meta: rest.RequestMeta{
			Symbols:  symbolMeta(req.Symbol),
			Category: string(bybit.RateLimitCategoryMarketData),
		},
	})
	if err != nil {
		return out, err
	}

	var payload instrumentsInfoPayload
	if err = resp.UnmarshalResult(&payload); err != nil {
		return out, bybit.NewError(bybit.ErrorKindUnknown, "", "premarket.GetInstruments: parse", err)
	}

	out.NextPageCursor = payload.NextPageCursor
	out.Instruments = make([]types.Instrument, 0, len(payload.List))
	var i int
	for i = 0; i < len(payload.List); i++ {
		out.Instruments = append(out.Instruments, convertInstrument(payload.List[i]))
	}
	return out, nil
}

// GetPreLaunchInstruments lists instruments with status=PreLaunch.
func (c *Client) GetPreLaunchInstruments(ctx context.Context, category commontypes.Category, cursor string) (types.InstrumentList, error) {
	return c.GetInstruments(ctx, types.InstrumentsRequest{
		Category: category,
		Status:   types.InstrumentStatusPreLaunch,
		Cursor:   cursor,
	})
}

// GetRiskLimit returns paginated risk-limit tiers for linear/inverse symbols.
func (c *Client) GetRiskLimit(ctx context.Context, req types.RiskLimitRequest) (types.RiskLimitList, error) {
	var out types.RiskLimitList
	if err := validateContractCategory(req.Category); err != nil {
		return out, err
	}

	var query url.Values = url.Values{}
	query.Set("category", string(req.Category))
	if req.Symbol != "" {
		query.Set("symbol", req.Symbol)
	}
	if req.Cursor != "" {
		query.Set("cursor", req.Cursor)
	}

	var resp rest.Response
	var err error
	resp, _, err = c.rest().Do(ctx, rest.Options{
		Method: "GET",
		Path:   "/v5/market/risk-limit",
		Query:  query,
		Signed: false,
		Meta: rest.RequestMeta{
			Symbols:  symbolMeta(req.Symbol),
			Category: string(bybit.RateLimitCategoryMarketData),
		},
	})
	if err != nil {
		return out, err
	}

	var payload riskLimitPayload
	if err = resp.UnmarshalResult(&payload); err != nil {
		return out, bybit.NewError(bybit.ErrorKindUnknown, "", "premarket.GetRiskLimit: parse", err)
	}

	out.NextPageCursor = payload.NextPageCursor
	out.Tiers = make([]types.RiskLimitTier, 0, len(payload.List))
	var i int
	for i = 0; i < len(payload.List); i++ {
		out.Tiers = append(out.Tiers, convertRiskLimit(payload.List[i]))
	}
	return out, nil
}

// GetTickers returns ticker snapshots with pre-market fields when applicable.
func (c *Client) GetTickers(ctx context.Context, category commontypes.Category, symbol string) (types.TickerList, error) {
	var out types.TickerList
	if err := validateContractCategory(category); err != nil {
		return out, err
	}

	var query url.Values = url.Values{}
	query.Set("category", string(category))
	if symbol != "" {
		query.Set("symbol", symbol)
	}

	var resp rest.Response
	var err error
	resp, _, err = c.rest().Do(ctx, rest.Options{
		Method: "GET",
		Path:   "/v5/market/tickers",
		Query:  query,
		Signed: false,
		Meta: rest.RequestMeta{
			Symbols:  symbolMeta(symbol),
			Category: string(bybit.RateLimitCategoryMarketData),
		},
	})
	if err != nil {
		return out, err
	}

	var payload tickersPayload
	if err = resp.UnmarshalResult(&payload); err != nil {
		return out, bybit.NewError(bybit.ErrorKindUnknown, "", "premarket.GetTickers: parse", err)
	}

	out.Tickers = make([]types.Ticker, 0, len(payload.List))
	var i int
	for i = 0; i < len(payload.List); i++ {
		out.Tickers = append(out.Tickers, convertTicker(payload.List[i]))
	}
	return out, nil
}

func symbolMeta(symbol string) []string {
	if symbol == "" {
		return nil
	}
	return []string{symbol}
}

// ---------------------------------------------------------------------
// Wire payloads + converters.
// ---------------------------------------------------------------------

type instrumentsInfoPayload struct {
	List           []rawInstrument `json:"list"`
	NextPageCursor string          `json:"nextPageCursor"`
}

type rawInstrument struct {
	Symbol         string                 `json:"symbol"`
	SymbolID       int64                  `json:"symbolId"`
	ContractType   string                 `json:"contractType"`
	Status         string                 `json:"status"`
	BaseCoin       string                 `json:"baseCoin"`
	QuoteCoin      string                 `json:"quoteCoin"`
	SettleCoin     string                 `json:"settleCoin"`
	LaunchTime     string                 `json:"launchTime"`
	PriceFilter    instrumentsPriceFilter `json:"priceFilter"`
	LotSizeFilter  instrumentsLotFilter   `json:"lotSizeFilter"`
	LeverageFilter instrumentsLevFilter   `json:"leverageFilter"`
	IsPreListing   bool                   `json:"isPreListing"`
	PreListingInfo *rawPreListingInfo     `json:"preListingInfo"`
}

type instrumentsPriceFilter struct {
	MinPrice string `json:"minPrice"`
	MaxPrice string `json:"maxPrice"`
	TickSize string `json:"tickSize"`
}

type instrumentsLotFilter struct {
	MaxOrderQty      string `json:"maxOrderQty"`
	MinOrderQty      string `json:"minOrderQty"`
	QtyStep          string `json:"qtyStep"`
	MaxMktOrderQty   string `json:"maxMktOrderQty"`
	MinNotionalValue string `json:"minNotionalValue"`
}

type instrumentsLevFilter struct {
	MinLeverage  string `json:"minLeverage"`
	MaxLeverage  string `json:"maxLeverage"`
	LeverageStep string `json:"leverageStep"`
}

type rawPreListingInfo struct {
	CurAuctionPhase string             `json:"curAuctionPhase"`
	Phases          []rawAuctionPhase  `json:"phases"`
	AuctionFeeInfo  rawAuctionFeeInfo  `json:"auctionFeeInfo"`
	SkipCallAuction bool               `json:"skipCallAuction"`
}

type rawAuctionPhase struct {
	Phase     string `json:"phase"`
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
}

type rawAuctionFeeInfo struct {
	AuctionFeeRate string `json:"auctionFeeRate"`
	TakerFeeRate   string `json:"takerFeeRate"`
	MakerFeeRate   string `json:"makerFeeRate"`
}

func convertInstrument(src rawInstrument) types.Instrument {
	var out types.Instrument = types.Instrument{
		Symbol:            src.Symbol,
		SymbolID:          src.SymbolID,
		ContractType:      src.ContractType,
		Status:            src.Status,
		BaseCoin:          src.BaseCoin,
		QuoteCoin:         src.QuoteCoin,
		SettleCoin:        src.SettleCoin,
		LaunchTimeMs:      ms(src.LaunchTime),
		TickSize:          dec(src.PriceFilter.TickSize),
		MinPrice:          dec(src.PriceFilter.MinPrice),
		MaxPrice:          dec(src.PriceFilter.MaxPrice),
		QtyStep:           dec(src.LotSizeFilter.QtyStep),
		MinOrderQty:       dec(src.LotSizeFilter.MinOrderQty),
		MaxOrderQty:       dec(src.LotSizeFilter.MaxOrderQty),
		MaxMarketOrderQty: dec(src.LotSizeFilter.MaxMktOrderQty),
		MinNotionalValue:  dec(src.LotSizeFilter.MinNotionalValue),
		MinLeverage:       dec(src.LeverageFilter.MinLeverage),
		MaxLeverage:       dec(src.LeverageFilter.MaxLeverage),
		LeverageStep:      dec(src.LeverageFilter.LeverageStep),
		IsPreListing:      src.IsPreListing,
	}
	if src.PreListingInfo != nil {
		out.PreListingInfo = convertPreListingInfo(*src.PreListingInfo)
	}
	return out
}

func convertPreListingInfo(src rawPreListingInfo) *types.PreListingInfo {
	var out types.PreListingInfo = types.PreListingInfo{
		CurAuctionPhase: types.AuctionPhase(src.CurAuctionPhase),
		SkipCallAuction: src.SkipCallAuction,
		AuctionFeeInfo: types.AuctionFeeInfo{
			AuctionFeeRate: dec(src.AuctionFeeInfo.AuctionFeeRate),
			TakerFeeRate:   dec(src.AuctionFeeInfo.TakerFeeRate),
			MakerFeeRate:   dec(src.AuctionFeeInfo.MakerFeeRate),
		},
	}
	out.Phases = make([]types.AuctionPhaseWindow, 0, len(src.Phases))
	var i int
	for i = 0; i < len(src.Phases); i++ {
		var row = src.Phases[i]
		out.Phases = append(out.Phases, types.AuctionPhaseWindow{
			Phase:       types.AuctionPhase(row.Phase),
			StartTimeMs: ms(row.StartTime),
			EndTimeMs:   ms(row.EndTime),
		})
	}
	return &out
}

type riskLimitPayload struct {
	List           []rawRiskLimitRow `json:"list"`
	NextPageCursor string            `json:"nextPageCursor"`
}

type rawRiskLimitRow struct {
	ID                int         `json:"id"`
	Symbol            string      `json:"symbol"`
	RiskLimitValue    string      `json:"riskLimitValue"`
	MaintenanceMargin json.Number `json:"maintenanceMargin"`
	InitialMargin     json.Number `json:"initialMargin"`
	IsLowestRisk      int         `json:"isLowestRisk"`
	MaxLeverage       string      `json:"maxLeverage"`
	MMDeduction       string      `json:"mmDeduction"`
}

func convertRiskLimit(src rawRiskLimitRow) types.RiskLimitTier {
	return types.RiskLimitTier{
		ID:                src.ID,
		Symbol:            src.Symbol,
		RiskLimitValue:    dec(src.RiskLimitValue),
		MaintenanceMargin: decNum(src.MaintenanceMargin),
		InitialMargin:     decNum(src.InitialMargin),
		IsLowestRisk:      src.IsLowestRisk == 1,
		MaxLeverage:       dec(src.MaxLeverage),
		MMDeduction:       dec(src.MMDeduction),
	}
}

func decNum(n json.Number) decimal.Decimal {
	if n == "" {
		return decimal.Zero
	}
	var d, err = decimal.NewFromString(n.String())
	if err != nil {
		return decimal.Zero
	}
	return d
}

type tickersPayload struct {
	List []rawTickerRow `json:"list"`
}

type rawTickerRow struct {
	Symbol             string `json:"symbol"`
	LastPrice          string `json:"lastPrice"`
	IndexPrice         string `json:"indexPrice"`
	MarkPrice          string `json:"markPrice"`
	Bid1Price          string `json:"bid1Price"`
	Ask1Price          string `json:"ask1Price"`
	Bid1Size           string `json:"bid1Size"`
	Ask1Size           string `json:"ask1Size"`
	OpenInterest       string `json:"openInterest"`
	FundingRate        string `json:"fundingRate"`
	PreOpenPrice       string `json:"preOpenPrice"`
	PreQty             string `json:"preQty"`
	CurPreListingPhase string `json:"curPreListingPhase"`
	FundingIntervalHour string `json:"fundingIntervalHour"`
}

func convertTicker(src rawTickerRow) types.Ticker {
	return types.Ticker{
		Symbol:              src.Symbol,
		LastPrice:           dec(src.LastPrice),
		IndexPrice:          dec(src.IndexPrice),
		MarkPrice:           dec(src.MarkPrice),
		Bid1Price:           dec(src.Bid1Price),
		Ask1Price:           dec(src.Ask1Price),
		Bid1Size:            dec(src.Bid1Size),
		Ask1Size:            dec(src.Ask1Size),
		OpenInterest:        dec(src.OpenInterest),
		FundingRate:         dec(src.FundingRate),
		PreOpenPrice:        dec(src.PreOpenPrice),
		PreQty:              dec(src.PreQty),
		CurPreListingPhase:  src.CurPreListingPhase,
		FundingIntervalHour: src.FundingIntervalHour,
	}
}
