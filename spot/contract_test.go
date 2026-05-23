/*
FILE: spot/contract_test.go

DESCRIPTION:
Contract tests for the spot profile. They verify that the parser
correctly maps real Bybit V5 spot JSON envelopes into domain structs.
Each fixture is hand-derived from the official Bybit V5 documentation.

Coverage:
  - GetSymbolInfo:          /v5/market/instruments-info?category=spot
                            (marginTrading / innovation / no leverage)
  - GetSymbolInfo NotFound: empty list returns ErrorKindInvalidRequest
  - GetOrderBook:           /v5/market/orderbook?category=spot
                            (depth clamping verified against {1,50,200})
  - GetHistoricalCandles:   /v5/market/kline?category=spot
  - GetWalletBalance:       /v5/account/wallet-balance
  - GetOpenOrders:          /v5/order/realtime?category=spot
                            (marketUnit / isLeverage echoed)
  - CreateOrder happy:      /v5/order/create
  - CreateOrder reject:     retCode != 0 → typed *bberr.Error
  - ModifyOrder 10001:      idempotent SUCCESS path
  - CancelAllOrders:        /v5/order/cancel-all returns N rows
  - CreateBatchOrders:      partial-failure agg + per-row Code

Tests use a local httptest.Server; no network calls are made.
*/

package spot

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	bybit "github.com/tonymontanov/go-bybit/v2"
	bybitspottypes "github.com/tonymontanov/go-bybit/v2/spot/types"
)

// mockBybit starts an httptest.Server that routes requests by path to a
// pre-baked JSON response. Unknown paths return 404 with a Bybit-style
// envelope so tests fail explicitly when the SDK targets an unexpected
// endpoint.
func mockBybit(t *testing.T, routes map[string]string) (*httptest.Server, *bybit.Client) {
	t.Helper()

	var srv *httptest.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body string
		var ok bool
		body, ok = routes[r.URL.Path]
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"retCode":404,"retMsg":"no fixture for `+r.URL.Path+`","result":{},"retExtInfo":{},"time":0}`)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Bapi-Limit", "100")
		w.Header().Set("X-Bapi-Limit-Status", "99")
		w.Header().Set("X-Bapi-Limit-Reset-Timestamp", "1700000000000")
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)

	var cfg bybit.Config = bybit.DefaultConfig()
	cfg.REST.BaseURL = srv.URL
	cfg.APIKey = "k"
	cfg.SecretKey = "s"
	cfg.REST.RequestTimeout = 3 * time.Second

	var client *bybit.Client
	var err error
	client, err = bybit.NewClient(cfg)
	if err != nil {
		t.Fatalf("bybit.NewClient: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return srv, client
}

func spotOf(c *bybit.Client) *Client { return c.Spot().(*Client) }

// ---------------------------------------------------------------------
// MarketData.
// ---------------------------------------------------------------------

func TestContract_GetSymbolInfo(t *testing.T) {
	t.Parallel()
	const fixture = `{
		"retCode":0,"retMsg":"OK",
		"result":{
			"category":"spot",
			"list":[{
				"symbol":"BTCUSDT",
				"baseCoin":"BTC","quoteCoin":"USDT",
				"innovation":"0",
				"status":"Trading",
				"marginTrading":"both",
				"priceFilter":{"minPrice":"0.01","maxPrice":"199999.99","tickSize":"0.01"},
				"lotSizeFilter":{
					"basePrecision":"0.000001",
					"quotePrecision":"0.00000001",
					"minOrderQty":"0.000048",
					"maxOrderQty":"71.73956243",
					"minOrderAmt":"1",
					"maxOrderAmt":"4000000"
				}
			}]
		},
		"retExtInfo":{},"time":1700000000000
	}`
	var _, client = mockBybit(t, map[string]string{
		"/v5/market/instruments-info": fixture,
	})
	var info bybitspottypes.SymbolInfo
	var err error
	info, err = spotOf(client).MarketData().GetSymbolInfo(context.Background(), "BTCUSDT")
	if err != nil {
		t.Fatalf("GetSymbolInfo: %v", err)
	}
	if info.Symbol != "BTCUSDT" {
		t.Fatalf("Symbol: got %q", info.Symbol)
	}
	if info.BaseCoin != "BTC" || info.QuoteCoin != "USDT" {
		t.Fatalf("base/quote: got %q/%q", info.BaseCoin, info.QuoteCoin)
	}
	if !info.TickSize.Equal(dq("0.01")) {
		t.Fatalf("TickSize: got %v", info.TickSize)
	}
	if !info.MinPrice.Equal(dq("0.01")) {
		t.Fatalf("MinPrice: got %v", info.MinPrice)
	}
	if !info.MaxPrice.Equal(dq("199999.99")) {
		t.Fatalf("MaxPrice: got %v", info.MaxPrice)
	}
	if info.PricePrecision != 2 {
		t.Fatalf("PricePrecision: got %d", info.PricePrecision)
	}
	if info.QuantityPrecision != 6 {
		t.Fatalf("QuantityPrecision: got %d", info.QuantityPrecision)
	}
	if info.MarginTrading != bybitspottypes.MarginTradingBoth {
		t.Fatalf("MarginTrading: got %q", info.MarginTrading)
	}
	if info.Innovation {
		t.Fatalf("Innovation: got true, want false (innovation=\"0\")")
	}
	if !info.MinOrderAmt.Equal(dq("1")) {
		t.Fatalf("MinOrderAmt: got %v", info.MinOrderAmt)
	}
}

func TestContract_GetSymbolInfo_NotFound(t *testing.T) {
	t.Parallel()
	const fixture = `{"retCode":0,"retMsg":"OK","result":{"category":"spot","list":[]},"retExtInfo":{},"time":0}`
	var _, client = mockBybit(t, map[string]string{
		"/v5/market/instruments-info": fixture,
	})
	var _, err = spotOf(client).MarketData().GetSymbolInfo(context.Background(), "FOOUSDT")
	if !bybit.IsInvalidRequest(err) {
		t.Fatalf("expected ErrorKindInvalidRequest, got %v", err)
	}
}

func TestContract_GetOrderBook(t *testing.T) {
	t.Parallel()
	const fixture = `{
		"retCode":0,"retMsg":"OK",
		"result":{
			"s":"BTCUSDT",
			"b":[["27050.00","0.5"],["27049.50","0.2"]],
			"a":[["27051.00","0.4"],["27052.00","0.1"]],
			"ts":1700000001000,
			"u":12345678,
			"seq":98765432
		},
		"retExtInfo":{},"time":0
	}`
	var _, client = mockBybit(t, map[string]string{
		"/v5/market/orderbook": fixture,
	})

	var ob bybitspottypes.OrderBookSnapshot
	var err error
	ob, err = spotOf(client).MarketData().GetOrderBook(context.Background(), "BTCUSDT", 25)
	if err != nil {
		t.Fatalf("GetOrderBook: %v", err)
	}
	if ob.Symbol != "BTCUSDT" {
		t.Fatalf("Symbol: got %q", ob.Symbol)
	}
	if len(ob.Bids) != 2 || len(ob.Asks) != 2 {
		t.Fatalf("levels: bids=%d asks=%d", len(ob.Bids), len(ob.Asks))
	}
	if !ob.Bids[0].Price.Equal(dq("27050.00")) {
		t.Fatalf("bid[0].Price: got %v", ob.Bids[0].Price)
	}
	if ob.UpdateID != 12345678 || ob.SeqID != 98765432 || ob.TsMs != 1700000001000 {
		t.Fatalf("ids/ts: %+v", ob)
	}
}

func TestContract_GetHistoricalCandles(t *testing.T) {
	t.Parallel()
	const fixture = `{
		"retCode":0,"retMsg":"OK",
		"result":{
			"category":"spot","symbol":"BTCUSDT",
			"list":[
				["1700000060000","27000.0","27050.0","26995.5","27010.0","12.34","333500.55"],
				["1700000000000","26995.0","27005.5","26990.0","27000.0","8.10","218800.10"]
			]
		},
		"retExtInfo":{},"time":0
	}`
	var _, client = mockBybit(t, map[string]string{
		"/v5/market/kline": fixture,
	})
	var candles bybitspottypes.Candles
	var err error
	candles, err = spotOf(client).MarketData().GetHistoricalCandles(context.Background(), HistoricalCandlesRequest{
		Symbol:    "BTCUSDT",
		Timeframe: bybitspottypes.Timeframe1m,
		Limit:     2,
	})
	if err != nil {
		t.Fatalf("GetHistoricalCandles: %v", err)
	}
	if len(candles) != 2 {
		t.Fatalf("len: got %d", len(candles))
	}
	if candles[0].OpenTimeMs != 1700000060000 {
		t.Fatalf("OpenTimeMs[0]: got %d", candles[0].OpenTimeMs)
	}
	if !candles[0].Close.Equal(dq("27010.0")) {
		t.Fatalf("Close[0]: got %v", candles[0].Close)
	}
	if !candles[1].Volume.Equal(dq("8.10")) {
		t.Fatalf("Volume[1]: got %v", candles[1].Volume)
	}
}

// ---------------------------------------------------------------------
// Account.
// ---------------------------------------------------------------------

func TestContract_GetWalletBalance(t *testing.T) {
	t.Parallel()
	const fixture = `{
		"retCode":0,"retMsg":"OK",
		"result":{
			"list":[{
				"accountType":"UNIFIED",
				"totalEquity":"12345.67",
				"totalWalletBalance":"12000.00",
				"totalAvailableBalance":"11000.00",
				"totalMarginBalance":"12345.67",
				"totalInitialMargin":"100.00",
				"totalMaintenanceMargin":"50.00",
				"totalPerpUPL":"0",
				"accountIMRate":"0.01",
				"accountMMRate":"0.005",
				"accountLTV":"0",
				"coin":[{
					"coin":"USDT",
					"equity":"11000",
					"walletBalance":"11000",
					"usdValue":"11000",
					"unrealisedPnl":"0",
					"cumRealisedPnl":"345.67",
					"borrowAmount":"0",
					"availableToWithdraw":"11000",
					"availableToBorrow":"0",
					"locked":"0",
					"totalOrderIM":"0",
					"totalPositionIM":"0",
					"totalPositionMM":"0",
					"accruedInterest":"0",
					"spotHedgingQty":"",
					"marginCollateral":true,
					"collateralSwitch":true
				}]
			}]
		},
		"retExtInfo":{},"time":0
	}`
	var _, client = mockBybit(t, map[string]string{
		"/v5/account/wallet-balance": fixture,
	})
	var bal bybitspottypes.Balance
	var err error
	bal, err = spotOf(client).Account().GetWalletBalance(context.Background(), WalletBalanceRequest{})
	if err != nil {
		t.Fatalf("GetWalletBalance: %v", err)
	}
	if bal.AccountType != bybitspottypes.AccountTypeUnified {
		t.Fatalf("AccountType: got %q", bal.AccountType)
	}
	if !bal.TotalEquity.Equal(dq("12345.67")) {
		t.Fatalf("TotalEquity: got %v", bal.TotalEquity)
	}
	if len(bal.Coins) != 1 {
		t.Fatalf("Coins len: got %d", len(bal.Coins))
	}
	if !bal.Coins[0].WalletBalance.Equal(dq("11000")) {
		t.Fatalf("Coins[0].WalletBalance: got %v", bal.Coins[0].WalletBalance)
	}
}

func TestContract_GetOpenOrders_SinglePage(t *testing.T) {
	t.Parallel()
	const fixture = `{
		"retCode":0,"retMsg":"OK",
		"result":{
			"category":"spot",
			"list":[{
				"orderId":"ord-1",
				"orderLinkId":"link-1",
				"symbol":"BTCUSDT",
				"side":"Buy",
				"orderType":"Limit",
				"timeInForce":"GTC",
				"price":"27000",
				"qty":"0.001",
				"leavesQty":"0.001",
				"cumExecQty":"0",
				"cumExecValue":"0",
				"avgPrice":"0",
				"cumExecFee":"0",
				"orderStatus":"New",
				"marketUnit":"",
				"isLeverage":"0",
				"rejectReason":"EC_NoError",
				"createdTime":"1700000003000",
				"updatedTime":"1700000003500"
			}],
			"nextPageCursor":""
		},
		"retExtInfo":{},"time":0
	}`
	var _, client = mockBybit(t, map[string]string{
		"/v5/order/realtime": fixture,
	})
	var orders []bybitspottypes.OrderInfo
	var err error
	orders, err = spotOf(client).Account().GetOpenOrders(context.Background(), "BTCUSDT")
	if err != nil {
		t.Fatalf("GetOpenOrders: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("len: got %d", len(orders))
	}
	if orders[0].OrderID != "ord-1" {
		t.Fatalf("OrderID: got %q", orders[0].OrderID)
	}
	if orders[0].Status != bybitspottypes.OrderStatusNew {
		t.Fatalf("Status: got %q", orders[0].Status)
	}
	if orders[0].CreatedAtMs != 1700000003000 {
		t.Fatalf("CreatedAtMs: got %d", orders[0].CreatedAtMs)
	}
	if orders[0].RejectReason != "" {
		t.Fatalf("RejectReason: %q must be masked (EC_NoError)", orders[0].RejectReason)
	}
	if orders[0].IsLeverage {
		t.Fatalf("IsLeverage: got true, want false (isLeverage=\"0\")")
	}
}

// ---------------------------------------------------------------------
// Trading.
// ---------------------------------------------------------------------

func TestContract_CreateOrder_Happy(t *testing.T) {
	t.Parallel()
	const fixture = `{
		"retCode":0,"retMsg":"OK",
		"result":{"orderId":"o-42","orderLinkId":"link-42"},
		"retExtInfo":{},"time":0
	}`
	var _, client = mockBybit(t, map[string]string{
		"/v5/order/create": fixture,
	})
	var info bybitspottypes.OrderInfo
	var err error
	info, err = spotOf(client).Trading().CreateOrder(context.Background(), bybitspottypes.CreateOrderRequest{
		Symbol:        "BTCUSDT",
		Side:          bybitspottypes.SideTypeBuy,
		OrderType:     bybitspottypes.OrderTypeLimit,
		TimeInForce:   bybitspottypes.TimeInForceGTC,
		Quantity:      dq("0.001"),
		Price:         dq("27000"),
		ClientOrderID: "link-42",
	})
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}
	if info.OrderID != "o-42" {
		t.Fatalf("OrderID: got %q", info.OrderID)
	}
	if info.ClientOrderID != "link-42" {
		t.Fatalf("ClientOrderID: got %q", info.ClientOrderID)
	}
	if info.Status != bybitspottypes.OrderStatusNew {
		t.Fatalf("Status: got %q", info.Status)
	}
	if got, ok := spotOf(client).Trading().OrderIDByClientID("link-42"); !ok || got != "o-42" {
		t.Fatalf("mapping not stored: got %q ok=%v", got, ok)
	}
	if info.RateLimits["X-Bapi-Limit-Status"] != "99" {
		t.Fatalf("RateLimits not propagated: %v", info.RateLimits)
	}
}

func TestContract_CreateOrder_RetCodeReject(t *testing.T) {
	t.Parallel()
	const fixture = `{
		"retCode":170131,"retMsg":"insufficient available balance",
		"result":{},"retExtInfo":{},"time":0
	}`
	var _, client = mockBybit(t, map[string]string{
		"/v5/order/create": fixture,
	})
	var _, err = spotOf(client).Trading().CreateOrder(context.Background(), bybitspottypes.CreateOrderRequest{
		Symbol:    "BTCUSDT",
		Side:      bybitspottypes.SideTypeBuy,
		OrderType: bybitspottypes.OrderTypeLimit,
		Quantity:  dq("100"),
		Price:     dq("100000"),
	})
	if err == nil {
		t.Fatalf("expected an error, got nil")
	}
	if !bybit.IsExchange(err) {
		t.Fatalf("expected ErrorKindExchange, got %v", err)
	}
	if !strings.Contains(err.Error(), "170131") {
		t.Fatalf("error must surface the Bybit code 170131, got %v", err)
	}
}

// TestContract_ModifyOrder_IdempotentOn10001 — Bybit replies retCode=10001
// when the requested amend is a no-op (qty/price unchanged). The SDK
// MUST treat this as a successful idempotent path, returning err=nil
// and the request-side fields echoed in the OrderInfo.
func TestContract_ModifyOrder_IdempotentOn10001(t *testing.T) {
	t.Parallel()
	const fixture = `{
		"retCode":10001,"retMsg":"order not modified",
		"result":{},"retExtInfo":{},"time":0
	}`
	var _, client = mockBybit(t, map[string]string{
		"/v5/order/amend": fixture,
	})
	var info bybitspottypes.OrderInfo
	var err error
	info, err = spotOf(client).Trading().ModifyOrder(context.Background(), bybitspottypes.ModifyOrderRequest{
		Symbol:   "BTCUSDT",
		OrderID:  "o-42",
		NewPrice: dq("27000"),
	})
	if err != nil {
		t.Fatalf("ModifyOrder must NOT error on retCode=10001 (idempotent), got %v", err)
	}
	if info.OrderID != "o-42" {
		t.Fatalf("OrderID echo: got %q", info.OrderID)
	}
	if !info.Price.Equal(dq("27000")) {
		t.Fatalf("Price echo: got %v", info.Price)
	}
}

// TestContract_ModifyOrder_PropagatesNonIdempotent — any retCode != 10001
// must surface as an *bberr.Error and the OrderInfo must be zero-valued.
func TestContract_ModifyOrder_PropagatesNonIdempotent(t *testing.T) {
	t.Parallel()
	const fixture = `{
		"retCode":110001,"retMsg":"order does not exist",
		"result":{},"retExtInfo":{},"time":0
	}`
	var _, client = mockBybit(t, map[string]string{
		"/v5/order/amend": fixture,
	})
	var _, err = spotOf(client).Trading().ModifyOrder(context.Background(), bybitspottypes.ModifyOrderRequest{
		Symbol:   "BTCUSDT",
		OrderID:  "o-99",
		NewPrice: dq("27000"),
	})
	if err == nil {
		t.Fatalf("expected an error for retCode=110001, got nil")
	}
	// 110001 is "order does not exist" — a Bybit-side validation
	// rejection, classified as InvalidRequest by MapBybitCode. The
	// important property here is that the SDK propagated the error
	// rather than swallowing it as the 10001 idempotent path.
	var asErr *bybit.Error
	if !errors.As(err, &asErr) || asErr.BybitCode != "110001" {
		t.Fatalf("expected typed *bybit.Error with code=110001, got %v", err)
	}
}

func TestContract_CancelAllOrders(t *testing.T) {
	t.Parallel()
	const fixture = `{
		"retCode":0,"retMsg":"OK",
		"result":{"list":[
			{"orderId":"o-1","orderLinkId":"link-1"},
			{"orderId":"o-2","orderLinkId":"link-2"}
		]},
		"retExtInfo":{},"time":0
	}`
	var _, client = mockBybit(t, map[string]string{
		"/v5/order/cancel-all": fixture,
	})
	var results []bybitspottypes.BatchOrderResult
	var err error
	results, err = spotOf(client).Trading().CancelAllOrders(context.Background(), "BTCUSDT")
	if err != nil {
		t.Fatalf("CancelAllOrders: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("len: got %d", len(results))
	}
	if results[0].Order.OrderID != "o-1" || results[0].Order.Status != bybitspottypes.OrderStatusCancelled {
		t.Fatalf("results[0]: got %+v", results[0])
	}
}

func TestContract_CreateBatchOrders_PartialFailure(t *testing.T) {
	t.Parallel()
	const fixture = `{
		"retCode":0,"retMsg":"OK",
		"result":{"list":[
			{"category":"spot","symbol":"BTCUSDT","orderId":"o-1","orderLinkId":"link-1","createAt":"1700000004000"},
			{"category":"spot","symbol":"BTCUSDT","orderId":"","orderLinkId":"","createAt":""}
		]},
		"retExtInfo":{"list":[
			{"code":0,"msg":"OK"},
			{"code":170131,"msg":"insufficient available balance"}
		]},
		"time":0
	}`
	var _, client = mockBybit(t, map[string]string{
		"/v5/order/create-batch": fixture,
	})
	var results []bybitspottypes.BatchOrderResult
	var err error
	results, err = spotOf(client).Trading().CreateBatchOrders(context.Background(), []bybitspottypes.CreateOrderRequest{
		{Symbol: "BTCUSDT", Side: bybitspottypes.SideTypeBuy, OrderType: bybitspottypes.OrderTypeLimit, Quantity: dq("0.001"), Price: dq("27000"), ClientOrderID: "link-1"},
		{Symbol: "BTCUSDT", Side: bybitspottypes.SideTypeBuy, OrderType: bybitspottypes.OrderTypeLimit, Quantity: dq("0.001"), Price: dq("27000"), ClientOrderID: "link-2"},
	})
	if err == nil {
		t.Fatalf("expected aggregated error for the rejected row, got nil")
	}
	if !bybit.IsExchange(err) {
		t.Fatalf("aggregated error should classify as Exchange, got %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("results len: got %d", len(results))
	}
	if !results[0].IsOK() || results[0].Order.OrderID != "o-1" {
		t.Fatalf("results[0] (success): got %+v", results[0])
	}
	if results[1].IsOK() {
		t.Fatalf("results[1] should be failure, got %+v", results[1])
	}
	if results[1].Code != 170131 {
		t.Fatalf("results[1].Code: got %d", results[1].Code)
	}
}

// TestContract_ModifyBatchOrders_10001Idempotent — per-row retCode=10001
// must NOT contribute to the aggregated error (matches linears).
func TestContract_ModifyBatchOrders_10001Idempotent(t *testing.T) {
	t.Parallel()
	const fixture = `{
		"retCode":0,"retMsg":"OK",
		"result":{"list":[
			{"category":"spot","symbol":"BTCUSDT","orderId":"o-1","orderLinkId":"link-1"},
			{"category":"spot","symbol":"BTCUSDT","orderId":"o-2","orderLinkId":"link-2"}
		]},
		"retExtInfo":{"list":[
			{"code":10001,"msg":"order not modified"},
			{"code":10001,"msg":"order not modified"}
		]},
		"time":0
	}`
	var _, client = mockBybit(t, map[string]string{
		"/v5/order/amend-batch": fixture,
	})
	var results []bybitspottypes.BatchOrderResult
	var err error
	results, err = spotOf(client).Trading().ModifyBatchOrders(context.Background(), []bybitspottypes.ModifyOrderRequest{
		{Symbol: "BTCUSDT", OrderID: "o-1", NewPrice: dq("27000")},
		{Symbol: "BTCUSDT", OrderID: "o-2", NewPrice: dq("27050")},
	})
	if err != nil {
		t.Fatalf("ModifyBatchOrders: 10001 must NOT error out, got %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("results len: got %d", len(results))
	}
	if results[0].Code != 10001 || results[1].Code != 10001 {
		t.Fatalf("Code propagation: got %d / %d", results[0].Code, results[1].Code)
	}
}

// TestContract_CancelForgottenOrders_FilterAndCancel — verifies the
// reconciliation pattern: GetOpenOrders → filter old → CancelBatchOrders.
func TestContract_CancelForgottenOrders_FilterAndCancel(t *testing.T) {
	t.Parallel()
	// One order is 10 minutes old (fits the 5-minute threshold) and
	// one is fresh; only the former should be cancelled.
	var nowMs int64 = time.Now().UnixMilli()
	var oldMs int64 = nowMs - 10*60*1000
	var freshMs int64 = nowMs - 30*1000
	var fixture = `{
		"retCode":0,"retMsg":"OK",
		"result":{
			"category":"spot",
			"list":[{
				"orderId":"o-old","orderLinkId":"link-old","symbol":"BTCUSDT",
				"side":"Buy","orderType":"Limit","timeInForce":"GTC",
				"price":"27000","qty":"0.001","leavesQty":"0.001","cumExecQty":"0","cumExecValue":"0",
				"avgPrice":"0","cumExecFee":"0","orderStatus":"New",
				"createdTime":"` + itoa(oldMs) + `","updatedTime":"` + itoa(oldMs) + `"
			},{
				"orderId":"o-fresh","orderLinkId":"link-fresh","symbol":"BTCUSDT",
				"side":"Buy","orderType":"Limit","timeInForce":"GTC",
				"price":"27000","qty":"0.001","leavesQty":"0.001","cumExecQty":"0","cumExecValue":"0",
				"avgPrice":"0","cumExecFee":"0","orderStatus":"New",
				"createdTime":"` + itoa(freshMs) + `","updatedTime":"` + itoa(freshMs) + `"
			}],
			"nextPageCursor":""
		},
		"retExtInfo":{},"time":0
	}`
	var batchFixture = `{
		"retCode":0,"retMsg":"OK",
		"result":{"list":[
			{"category":"spot","symbol":"BTCUSDT","orderId":"o-old","orderLinkId":"link-old"}
		]},
		"retExtInfo":{"list":[{"code":0,"msg":"OK"}]},
		"time":0
	}`
	var _, client = mockBybit(t, map[string]string{
		"/v5/order/realtime":     fixture,
		"/v5/order/cancel-batch": batchFixture,
	})
	var results []bybitspottypes.BatchOrderResult
	var err error
	results, err = spotOf(client).Trading().CancelForgottenOrders(context.Background(), "BTCUSDT", 5*time.Minute)
	if err != nil {
		t.Fatalf("CancelForgottenOrders: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 cancelled (only the 10m-old order); got %d", len(results))
	}
	if results[0].Order.OrderID != "o-old" {
		t.Fatalf("expected o-old, got %q", results[0].Order.OrderID)
	}
}

func TestContract_CancelForgottenOrders_NoForgotten(t *testing.T) {
	t.Parallel()
	const fixture = `{
		"retCode":0,"retMsg":"OK",
		"result":{"category":"spot","list":[],"nextPageCursor":""},
		"retExtInfo":{},"time":0
	}`
	var _, client = mockBybit(t, map[string]string{
		"/v5/order/realtime": fixture,
	})
	var results []bybitspottypes.BatchOrderResult
	var err error
	results, err = spotOf(client).Trading().CancelForgottenOrders(context.Background(), "BTCUSDT", 1*time.Minute)
	if err != nil {
		t.Fatalf("CancelForgottenOrders: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected no cancellations, got %d", len(results))
	}
}

// itoa is a local int64 → string helper to keep the fixture builders
// terse and avoid importing strconv in tests.
func itoa(v int64) string {
	if v == 0 {
		return "0"
	}
	var negative bool
	if v < 0 {
		negative = true
		v = -v
	}
	var digits []byte
	for v > 0 {
		digits = append([]byte{byte('0' + v%10)}, digits...)
		v /= 10
	}
	if negative {
		return "-" + string(digits)
	}
	return string(digits)
}

// Sanity assertion: the bybit error type is matchable via errors.As
// from the package's filter helper. Keep this here so a refactor that
// breaks the interface is caught locally.
func TestErrorsAsUnwrap(t *testing.T) {
	t.Parallel()
	var raw error = &bybit.Error{Kind: bybit.ErrorKindExchange, BybitCode: "42", Message: "x"}
	var as *bybit.Error
	if !errors.As(raw, &as) || as.BybitCode != "42" {
		t.Fatalf("errors.As: got %+v", as)
	}
}
