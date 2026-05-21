/*
FILE: linears/market.go

DESCRIPTION:
Public market-data sub-client for the Bybit V5 linear category. None of
these endpoints require authentication; the SDK still funnels them
through the same restDoer for unified rate-limit accounting.

Implements:
  - GetSymbolInfo        : GET /v5/market/instruments-info
  - GetOrderBook         : GET /v5/market/orderbook
  - GetHistoricalCandles : GET /v5/market/kline

BYBIT V5 SPECIFICS:
  - /v5/market/orderbook depth limits (linear): 1, 50, 200, 500.
    The SDK clamps the caller's requested depth to the nearest allowed
    value (Bybit otherwise returns retCode 10001).
  - /v5/market/kline returns klines DESCENDING by start time. The SDK
    preserves that order (no reverse) — it matches Bybit docs and lets
    callers cap the most-recent N candles cheaply (slice prefix). If
    chronological order is needed, reverse on the caller side.
  - /v5/market/instruments-info supports a "limit" query param for paged
    listings; the SDK only exposes a single-symbol form for v1.
*/

package linears

import (
	"context"
	"net/url"
	"strconv"

	bybit "github.com/tonymontanov/go-bybit"
	"github.com/tonymontanov/go-bybit/internal/rest"
	"github.com/tonymontanov/go-bybit/linears/types"
)

// MarketDataClient — public market-data sub-client.
type MarketDataClient struct {
	c *Client
}

func newMarketDataClient(c *Client) *MarketDataClient {
	return &MarketDataClient{c: c}
}

// ---------------------------------------------------------------------
// Symbol info.
// ---------------------------------------------------------------------

// GetSymbolInfo returns the instrument specification for `symbol`.
// Returns ErrorKindInvalidRequest if the exchange has no row for the
// symbol (Bybit replies with retCode 0 + empty list in that case).
func (m *MarketDataClient) GetSymbolInfo(ctx context.Context, symbol string) (types.SymbolInfo, error) {
	var out types.SymbolInfo
	if symbol == "" {
		return out, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "marketdata.GetSymbolInfo: symbol is empty", nil)
	}
	var query url.Values = url.Values{}
	query.Set("category", string(types.CategoryLinear))
	query.Set("symbol", symbol)

	var resp rest.Response
	var err error
	resp, _, err = m.c.rest().Do(ctx, rest.Options{
		Method: "GET",
		Path:   "/v5/market/instruments-info",
		Query:  query,
		Signed: false,
		Meta: rest.RequestMeta{
			Symbols:  []string{symbol},
			Category: string(bybit.RateLimitCategoryMarketData),
		},
	})
	if err != nil {
		return out, err
	}

	var payload instrumentsInfoPayload
	if err = resp.UnmarshalResult(&payload); err != nil {
		return out, bybit.NewError(bybit.ErrorKindUnknown, "", "marketdata.GetSymbolInfo: parse", err)
	}
	if len(payload.List) == 0 {
		return out, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "marketdata.GetSymbolInfo: symbol not found: "+symbol, nil)
	}
	return convertSymbolInfo(payload.List[0]), nil
}

// ---------------------------------------------------------------------
// Order book snapshot.
// ---------------------------------------------------------------------

// orderbookDepthLimits — allowed values for /v5/market/orderbook?limit=.
// Bybit will reject any other value with retCode 10001.
var orderbookDepthLimits = []int{1, 50, 200, 500}

// GetOrderBook returns a depth snapshot for `symbol`. The depth argument
// is clamped to the nearest allowed value (1, 50, 200, 500). depth ≤ 0
// resolves to 50, the SDK default.
func (m *MarketDataClient) GetOrderBook(ctx context.Context, symbol string, depth int) (types.OrderBookSnapshot, error) {
	var out types.OrderBookSnapshot
	if symbol == "" {
		return out, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "marketdata.GetOrderBook: symbol is empty", nil)
	}
	depth = clampOrderbookDepth(depth)

	var query url.Values = url.Values{}
	query.Set("category", string(types.CategoryLinear))
	query.Set("symbol", symbol)
	query.Set("limit", strconv.Itoa(depth))

	var resp rest.Response
	var err error
	resp, _, err = m.c.rest().Do(ctx, rest.Options{
		Method: "GET",
		Path:   "/v5/market/orderbook",
		Query:  query,
		Signed: false,
		Meta: rest.RequestMeta{
			Symbols:  []string{symbol},
			Category: string(bybit.RateLimitCategoryMarketData),
		},
	})
	if err != nil {
		return out, err
	}

	var payload orderbookSnapshotPayload
	if err = resp.UnmarshalResult(&payload); err != nil {
		return out, bybit.NewError(bybit.ErrorKindUnknown, "", "marketdata.GetOrderBook: parse", err)
	}
	out.Symbol = payload.Symbol
	if out.Symbol == "" {
		out.Symbol = symbol
	}
	out.UpdateID = payload.U
	out.SeqID = payload.Seq
	out.TsMs = payload.Ts
	out.Bids = convertLevels(payload.B)
	out.Asks = convertLevels(payload.A)
	return out, nil
}

// clampOrderbookDepth maps an arbitrary integer to the nearest allowed
// Bybit limit. ≤ 0 → 50 (SDK default). Above 500 → 500.
func clampOrderbookDepth(d int) int {
	if d <= 0 {
		return 50
	}
	var i int
	for i = 0; i < len(orderbookDepthLimits); i++ {
		if d <= orderbookDepthLimits[i] {
			return orderbookDepthLimits[i]
		}
	}
	return orderbookDepthLimits[len(orderbookDepthLimits)-1]
}

// ---------------------------------------------------------------------
// Historical candles.
// ---------------------------------------------------------------------

// HistoricalCandlesRequest — parameters for GetHistoricalCandles.
//
// FIELDS:
//   - Symbol    : Bybit symbol.
//   - Timeframe : kline interval enum.
//   - StartMs   : optional inclusive lower bound (epoch ms). Zero means
//     "no lower bound".
//   - EndMs     : optional inclusive upper bound. Zero means "no upper
//     bound".
//   - Limit     : page size, 1..1000. Zero or negative defaults to 200.
//     Bybit caps at 1000 per call; the SDK does NOT paginate
//     transparently here — callers requesting more than 1000
//     candles should chunk by StartMs/EndMs themselves.
type HistoricalCandlesRequest struct {
	Symbol    string
	Timeframe types.Timeframe
	StartMs   int64
	EndMs     int64
	Limit     int
}

// GetHistoricalCandles returns historical klines for the request.
//
// Order is descending by OpenTimeMs (matching Bybit's wire format).
// Use sort.Slice on the result if chronological order is needed.
func (m *MarketDataClient) GetHistoricalCandles(ctx context.Context, req HistoricalCandlesRequest) (types.Candles, error) {
	if req.Symbol == "" {
		return nil, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "marketdata.GetHistoricalCandles: symbol is empty", nil)
	}
	if req.Timeframe == "" {
		return nil, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "marketdata.GetHistoricalCandles: timeframe is empty", nil)
	}
	var limit int = req.Limit
	if limit <= 0 {
		limit = 200
	}
	if limit > 1000 {
		limit = 1000
	}

	var query url.Values = url.Values{}
	query.Set("category", string(types.CategoryLinear))
	query.Set("symbol", req.Symbol)
	query.Set("interval", req.Timeframe.Wire())
	query.Set("limit", strconv.Itoa(limit))
	if req.StartMs > 0 {
		query.Set("start", strconv.FormatInt(req.StartMs, 10))
	}
	if req.EndMs > 0 {
		query.Set("end", strconv.FormatInt(req.EndMs, 10))
	}

	var resp rest.Response
	var err error
	resp, _, err = m.c.rest().Do(ctx, rest.Options{
		Method: "GET",
		Path:   "/v5/market/kline",
		Query:  query,
		Signed: false,
		Meta: rest.RequestMeta{
			Symbols:  []string{req.Symbol},
			Category: string(bybit.RateLimitCategoryMarketData),
		},
	})
	if err != nil {
		return nil, err
	}

	var payload klinePayload
	if err = resp.UnmarshalResult(&payload); err != nil {
		return nil, bybit.NewError(bybit.ErrorKindUnknown, "", "marketdata.GetHistoricalCandles: parse", err)
	}

	var out types.Candles = make(types.Candles, 0, len(payload.List))
	var i int
	for i = 0; i < len(payload.List); i++ {
		var row []string = payload.List[i]
		if len(row) < 7 {
			continue
		}
		out = append(out, types.Candle{
			OpenTimeMs:  ms(row[0]),
			Open:        dec(row[1]),
			High:        dec(row[2]),
			Low:         dec(row[3]),
			Close:       dec(row[4]),
			Volume:      dec(row[5]),
			VolumeQuote: dec(row[6]),
		})
	}
	return out, nil
}

// ---------------------------------------------------------------------
// Wire payloads + converters.
// ---------------------------------------------------------------------

type instrumentsInfoPayload struct {
	Category       string             `json:"category"`
	List           []instrumentsEntry `json:"list"`
	NextPageCursor string             `json:"nextPageCursor"`
}

type instrumentsEntry struct {
	Symbol         string                 `json:"symbol"`
	ContractType   string                 `json:"contractType"`
	Status         string                 `json:"status"`
	BaseCoin       string                 `json:"baseCoin"`
	QuoteCoin      string                 `json:"quoteCoin"`
	SettleCoin     string                 `json:"settleCoin"`
	PriceFilter    instrumentsPriceFilter `json:"priceFilter"`
	LotSizeFilter  instrumentsLotFilter   `json:"lotSizeFilter"`
	LeverageFilter instrumentsLevFilter   `json:"leverageFilter"`
}

type instrumentsPriceFilter struct {
	MinPrice string `json:"minPrice"`
	MaxPrice string `json:"maxPrice"`
	TickSize string `json:"tickSize"`
}

type instrumentsLotFilter struct {
	MaxOrderQty         string `json:"maxOrderQty"`
	MinOrderQty         string `json:"minOrderQty"`
	QtyStep             string `json:"qtyStep"`
	PostOnlyMaxOrderQty string `json:"postOnlyMaxOrderQty"`
	MinNotionalValue    string `json:"minNotionalValue"`
	MaxMktOrderQty      string `json:"maxMktOrderQty"`
}

type instrumentsLevFilter struct {
	MinLeverage  string `json:"minLeverage"`
	MaxLeverage  string `json:"maxLeverage"`
	LeverageStep string `json:"leverageStep"`
}

func convertSymbolInfo(src instrumentsEntry) types.SymbolInfo {
	var tick = dec(src.PriceFilter.TickSize)
	var step = dec(src.LotSizeFilter.QtyStep)
	return types.SymbolInfo{
		Symbol:            src.Symbol,
		BaseCoin:          src.BaseCoin,
		QuoteCoin:         src.QuoteCoin,
		SettleCoin:        src.SettleCoin,
		ContractType:      src.ContractType,
		Status:            src.Status,
		TickSize:          tick,
		MinPrice:          dec(src.PriceFilter.MinPrice),
		MaxPrice:          dec(src.PriceFilter.MaxPrice),
		QtyStep:           step,
		MinOrderQty:       dec(src.LotSizeFilter.MinOrderQty),
		MaxOrderQty:       dec(src.LotSizeFilter.MaxOrderQty),
		MaxMarketOrderQty: dec(src.LotSizeFilter.MaxMktOrderQty),
		MinNotionalValue:  dec(src.LotSizeFilter.MinNotionalValue),
		MinLeverage:       dec(src.LeverageFilter.MinLeverage),
		MaxLeverage:       dec(src.LeverageFilter.MaxLeverage),
		LeverageStep:      dec(src.LeverageFilter.LeverageStep),
		PricePrecision:    -tick.Exponent(),
		QuantityPrecision: -step.Exponent(),
	}
}

type orderbookSnapshotPayload struct {
	Symbol string     `json:"s"`
	B      [][]string `json:"b"`
	A      [][]string `json:"a"`
	Ts     int64      `json:"ts"`
	U      int64      `json:"u"`
	Seq    int64      `json:"seq"`
}

func convertLevels(rows [][]string) []types.OrderBookLevel {
	if len(rows) == 0 {
		return nil
	}
	var out []types.OrderBookLevel = make([]types.OrderBookLevel, 0, len(rows))
	var i int
	for i = 0; i < len(rows); i++ {
		if len(rows[i]) < 2 {
			continue
		}
		out = append(out, types.OrderBookLevel{
			Price: dec(rows[i][0]),
			Size:  dec(rows[i][1]),
		})
	}
	return out
}

type klinePayload struct {
	Symbol   string     `json:"symbol"`
	Category string     `json:"category"`
	List     [][]string `json:"list"`
}
