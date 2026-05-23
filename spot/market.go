/*
FILE: spot/market.go

DESCRIPTION:
Public market-data sub-client for the Bybit V5 spot category. None of
these endpoints require authentication; the SDK still routes them
through the same restDoer for unified rate-limit accounting.

Implements:
  - GetSymbolInfo        — GET /v5/market/instruments-info?category=spot
  - GetOrderBook         — GET /v5/market/orderbook?category=spot
  - GetHistoricalCandles — GET /v5/market/kline?category=spot

BYBIT V5 SPECIFICS (spot):
  - /v5/market/orderbook depth limits (spot): 1, 50, 200. The SDK
    clamps the caller's requested depth to the nearest allowed value
    (Bybit otherwise returns retCode 10001).
  - /v5/market/kline returns klines DESCENDING by start time (matches
    Bybit docs and lets callers cap the most-recent N candles cheaply).
  - /v5/market/instruments-info exposes spot-specific fields:
    `marginTrading`, `innovation`, `lotSizeFilter.basePrecision`,
    `quotePrecision`, `minOrderQty`, `maxOrderQty`, `minOrderAmt`,
    `maxOrderAmt`. Leverage filter is absent.
*/

package spot

import (
	"context"
	"net/url"
	"strconv"

	"github.com/shopspring/decimal"

	bybit "github.com/tonymontanov/go-bybit"
	"github.com/tonymontanov/go-bybit/internal/rest"
	"github.com/tonymontanov/go-bybit/internal/v5common"
	bybitspottypes "github.com/tonymontanov/go-bybit/spot/types"
)

// MarketDataClient — public market-data sub-client.
type MarketDataClient struct {
	c *Client
}

func newMarketDataClient(c *Client) *MarketDataClient {
	return &MarketDataClient{c: c}
}

// HistoricalCandlesRequest — parameters for GetHistoricalCandles.
//
// FIELDS:
//   - Symbol    : Bybit spot symbol (e.g. "BTCUSDT").
//   - Timeframe : kline interval enum.
//   - StartMs   : optional inclusive lower bound (epoch ms). Zero =
//     "no lower bound".
//   - EndMs     : optional inclusive upper bound. Zero = "no upper
//     bound".
//   - Limit     : page size, 1..1000. Zero or negative defaults to 200.
//     Bybit caps at 1000 per call; the SDK does NOT paginate
//     transparently here — callers requesting more than 1000
//     candles should chunk by StartMs/EndMs themselves.
type HistoricalCandlesRequest struct {
	Symbol    string
	Timeframe bybitspottypes.Timeframe
	StartMs   int64
	EndMs     int64
	Limit     int
}

// ---------------------------------------------------------------------
// Symbol info.
// ---------------------------------------------------------------------

// GetSymbolInfo returns the instrument specification for `symbol`.
// Returns ErrorKindInvalidRequest if the exchange has no row for the
// symbol (Bybit replies with retCode 0 + empty list in that case).
func (m *MarketDataClient) GetSymbolInfo(ctx context.Context, symbol string) (bybitspottypes.SymbolInfo, error) {
	var out bybitspottypes.SymbolInfo
	if symbol == "" {
		return out, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "marketdata.GetSymbolInfo: symbol is empty", nil)
	}
	var query url.Values = url.Values{}
	query.Set("category", string(bybitspottypes.CategorySpot))
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

	var payload spotInstrumentsInfoPayload
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

// orderbookDepthLimits — allowed values for /v5/market/orderbook?limit=
// when category=spot. Bybit will reject any other value with retCode
// 10001. The list MUST stay sorted ascending — v5common.ClampOrderbook
// Depth assumes that.
var orderbookDepthLimits = []int{1, 50, 200}

// GetOrderBook returns a depth snapshot for `symbol`. The depth argument
// is clamped to the nearest allowed value (1, 50, 200). depth ≤ 0
// resolves to 50, the SDK default.
func (m *MarketDataClient) GetOrderBook(ctx context.Context, symbol string, depth int) (bybitspottypes.OrderBookSnapshot, error) {
	var out bybitspottypes.OrderBookSnapshot
	if symbol == "" {
		return out, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "marketdata.GetOrderBook: symbol is empty", nil)
	}
	depth = clampOrderbookDepth(depth)

	var query url.Values = url.Values{}
	query.Set("category", string(bybitspottypes.CategorySpot))
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
// Bybit spot limit. ≤ 0 → 50 (SDK default). Above 200 → 200. Wraps
// v5common.ClampOrderbookDepth.
func clampOrderbookDepth(d int) int {
	if d <= 0 {
		return 50
	}
	return v5common.ClampOrderbookDepth(d, orderbookDepthLimits)
}

// ---------------------------------------------------------------------
// Historical candles.
// ---------------------------------------------------------------------

// GetHistoricalCandles returns historical klines for the request.
//
// Order is descending by OpenTimeMs (matching Bybit's wire format).
// Use sort.Slice on the result if chronological order is needed.
func (m *MarketDataClient) GetHistoricalCandles(ctx context.Context, req HistoricalCandlesRequest) (bybitspottypes.Candles, error) {
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
	query.Set("category", string(bybitspottypes.CategorySpot))
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

	var out bybitspottypes.Candles = make(bybitspottypes.Candles, 0, len(payload.List))
	var i int
	for i = 0; i < len(payload.List); i++ {
		var row []string = payload.List[i]
		if len(row) < 7 {
			continue
		}
		out = append(out, bybitspottypes.Candle{
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

type spotInstrumentsInfoPayload struct {
	Category       string                 `json:"category"`
	List           []spotInstrumentsEntry `json:"list"`
	NextPageCursor string                 `json:"nextPageCursor"`
}

// spotInstrumentsEntry mirrors Bybit's /v5/market/instruments-info row
// for category=spot. Note: marginTrading is bool-ish on the wire (a
// string enum) and innovation is a "0"/"1" string per Bybit docs.
type spotInstrumentsEntry struct {
	Symbol        string                     `json:"symbol"`
	BaseCoin      string                     `json:"baseCoin"`
	QuoteCoin     string                     `json:"quoteCoin"`
	Status        string                     `json:"status"`
	Innovation    string                     `json:"innovation"`
	MarginTrading string                     `json:"marginTrading"`
	PriceFilter   spotInstrumentsPriceFilter `json:"priceFilter"`
	LotSizeFilter spotInstrumentsLotFilter   `json:"lotSizeFilter"`
}

type spotInstrumentsPriceFilter struct {
	MinPrice string `json:"minPrice"`
	MaxPrice string `json:"maxPrice"`
	TickSize string `json:"tickSize"`
}

type spotInstrumentsLotFilter struct {
	BasePrecision  string `json:"basePrecision"`
	QuotePrecision string `json:"quotePrecision"`
	MinOrderQty    string `json:"minOrderQty"`
	MaxOrderQty    string `json:"maxOrderQty"`
	MinOrderAmt    string `json:"minOrderAmt"`
	MaxOrderAmt    string `json:"maxOrderAmt"`
}

func convertSymbolInfo(src spotInstrumentsEntry) bybitspottypes.SymbolInfo {
	var tick = dec(src.PriceFilter.TickSize)
	var basePrec = dec(src.LotSizeFilter.BasePrecision)
	return bybitspottypes.SymbolInfo{
		Symbol:            src.Symbol,
		BaseCoin:          src.BaseCoin,
		QuoteCoin:         src.QuoteCoin,
		Status:            src.Status,
		TickSize:          tick,
		MinPrice:          dec(src.PriceFilter.MinPrice),
		MaxPrice:          dec(src.PriceFilter.MaxPrice),
		BasePrecision:     basePrec,
		QuotePrecision:    dec(src.LotSizeFilter.QuotePrecision),
		MinOrderQty:       dec(src.LotSizeFilter.MinOrderQty),
		MaxOrderQty:       dec(src.LotSizeFilter.MaxOrderQty),
		MinOrderAmt:       dec(src.LotSizeFilter.MinOrderAmt),
		MaxOrderAmt:       dec(src.LotSizeFilter.MaxOrderAmt),
		MarginTrading:     bybitspottypes.MarginTrading(src.MarginTrading),
		Innovation:        src.Innovation == "1",
		PricePrecision:    -tick.Exponent(),
		QuantityPrecision: -basePrec.Exponent(),
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

func convertLevels(rows [][]string) []bybitspottypes.OrderBookLevel {
	return v5common.ConvertOrderBookLevels(rows, func(p, s decimal.Decimal) bybitspottypes.OrderBookLevel {
		return bybitspottypes.OrderBookLevel{Price: p, Size: s}
	})
}

type klinePayload struct {
	Symbol   string     `json:"symbol"`
	Category string     `json:"category"`
	List     [][]string `json:"list"`
}
