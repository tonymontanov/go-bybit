/*
FILE: linears/contract_test.go

DESCRIPTION:
Contract tests for the linears profile. They verify that the parser
correctly maps real Bybit V5 JSON envelopes into domain structs. Each
fixture is hand-derived from the official Bybit V5 documentation
(api.bybit.com /v5/...).

Coverage:
  - GetSymbolInfo:          /v5/market/instruments-info
  - GetSymbolInfo NotFound: empty list returns ErrorKindInvalidRequest
  - GetOrderBook:           /v5/market/orderbook (depth clamping verified)
  - GetHistoricalCandles:   /v5/market/kline
  - GetWalletBalance:       /v5/account/wallet-balance
  - GetPosition:            /v5/position/list
  - GetOpenOrders:          /v5/order/realtime — single page
  - CreateOrder happy:      /v5/order/create — orderId/orderLinkId echoed
  - CreateOrder reject:     retCode != 0 → typed *bberr.Error
  - CancelAllOrders:        /v5/order/cancel-all returns N rows

Tests use a local httptest.Server; no network calls are made.
*/

package linears

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	bybit "github.com/tonymontanov/go-bybit/v2"
	"github.com/tonymontanov/go-bybit/v2/linears/types"
)

// mockBybit starts an httptest.Server that routes requests by path to a
// pre-baked JSON response. Unknown paths return 404 with an envelope so
// tests fail explicitly when the SDK targets an unexpected endpoint.
//
// Bybit's rate-limit headers are set on every reply so the observer
// pipeline is exercised.
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

func linearsOf(c *bybit.Client) *Client { return c.Linears().(*Client) }

// ---------------------------------------------------------------------
// MarketData.
// ---------------------------------------------------------------------

func TestContract_GetSymbolInfo(t *testing.T) {
	t.Parallel()
	const fixture = `{
		"retCode":0,"retMsg":"OK",
		"result":{
			"category":"linear",
			"list":[{
				"symbol":"BTCUSDT",
				"contractType":"LinearPerpetual",
				"status":"Trading",
				"baseCoin":"BTC","quoteCoin":"USDT","settleCoin":"USDT",
				"priceFilter":{"minPrice":"0.10","maxPrice":"1999999.80","tickSize":"0.10"},
				"lotSizeFilter":{
					"maxOrderQty":"1190.000","minOrderQty":"0.001","qtyStep":"0.001",
					"postOnlyMaxOrderQty":"1000.000","minNotionalValue":"5","maxMktOrderQty":"10.000"
				},
				"leverageFilter":{"minLeverage":"1","maxLeverage":"100.00","leverageStep":"0.01"}
			}]
		},
		"retExtInfo":{},"time":1700000000000
	}`
	var _, client = mockBybit(t, map[string]string{
		"/v5/market/instruments-info": fixture,
	})
	var info types.SymbolInfo
	var err error
	info, err = linearsOf(client).MarketData().GetSymbolInfo(context.Background(), "BTCUSDT")
	if err != nil {
		t.Fatalf("GetSymbolInfo: %v", err)
	}
	if info.Symbol != "BTCUSDT" {
		t.Fatalf("Symbol: got %q", info.Symbol)
	}
	if info.SettleCoin != "USDT" {
		t.Fatalf("SettleCoin: got %q", info.SettleCoin)
	}
	if !info.TickSize.Equal(dq("0.10")) {
		t.Fatalf("TickSize: got %v", info.TickSize)
	}
	// Bybit V5 priceFilter содержит minPrice/maxPrice — оба должны
	// прокидываться, иначе downstream-callers не смогут clamp'ить
	// цену ордера и получат zero-price после Round (видели в проде на
	// PARTIUSDT: tickSize=0.00001, maxPrice=199.99998).
	if !info.MinPrice.Equal(dq("0.10")) {
		t.Fatalf("MinPrice: got %v", info.MinPrice)
	}
	if !info.MaxPrice.Equal(dq("1999999.80")) {
		t.Fatalf("MaxPrice: got %v", info.MaxPrice)
	}
	if !info.QtyStep.Equal(dq("0.001")) {
		t.Fatalf("QtyStep: got %v", info.QtyStep)
	}
	if info.PricePrecision != 2 {
		t.Fatalf("PricePrecision: got %d", info.PricePrecision)
	}
	if info.QuantityPrecision != 3 {
		t.Fatalf("QuantityPrecision: got %d", info.QuantityPrecision)
	}
	if !info.MaxLeverage.Equal(dq("100.00")) {
		t.Fatalf("MaxLeverage: got %v", info.MaxLeverage)
	}
}

func TestContract_GetSymbolInfo_NotFound(t *testing.T) {
	t.Parallel()
	const fixture = `{"retCode":0,"retMsg":"OK","result":{"category":"linear","list":[]},"retExtInfo":{},"time":0}`
	var _, client = mockBybit(t, map[string]string{
		"/v5/market/instruments-info": fixture,
	})
	var _, err = linearsOf(client).MarketData().GetSymbolInfo(context.Background(), "FOOUSDT")
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
	var srv, client = mockBybit(t, map[string]string{
		"/v5/market/orderbook": fixture,
	})
	_ = srv

	var ob types.OrderBookSnapshot
	var err error
	ob, err = linearsOf(client).MarketData().GetOrderBook(context.Background(), "BTCUSDT", 25)
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
	if ob.UpdateID != 12345678 {
		t.Fatalf("UpdateID: got %d", ob.UpdateID)
	}
	if ob.SeqID != 98765432 {
		t.Fatalf("SeqID: got %d", ob.SeqID)
	}
	if ob.TsMs != 1700000001000 {
		t.Fatalf("TsMs: got %d", ob.TsMs)
	}
}

func TestContract_GetHistoricalCandles(t *testing.T) {
	t.Parallel()
	const fixture = `{
		"retCode":0,"retMsg":"OK",
		"result":{
			"category":"linear","symbol":"BTCUSDT",
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
	var candles types.Candles
	var err error
	candles, err = linearsOf(client).MarketData().GetHistoricalCandles(context.Background(), HistoricalCandlesRequest{
		Symbol:    "BTCUSDT",
		Timeframe: types.Timeframe1m,
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
				"totalPerpUPL":"345.67",
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
					"totalOrderIM":"50",
					"totalPositionIM":"50",
					"totalPositionMM":"25",
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
	var bal types.Balance
	var err error
	bal, err = linearsOf(client).Account().GetWalletBalance(context.Background(), WalletBalanceRequest{})
	if err != nil {
		t.Fatalf("GetWalletBalance: %v", err)
	}
	if bal.AccountType != types.AccountTypeUnified {
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
	if !bal.Coins[0].MarginCollateral {
		t.Fatalf("Coins[0].MarginCollateral: got false")
	}
}

func TestContract_GetPosition(t *testing.T) {
	t.Parallel()
	const fixture = `{
		"retCode":0,"retMsg":"OK",
		"result":{
			"category":"linear",
			"list":[{
				"positionIdx":0,
				"symbol":"BTCUSDT",
				"side":"Buy",
				"size":"0.020",
				"avgPrice":"27000.50",
				"markPrice":"27100.00",
				"liqPrice":"24000.00",
				"leverage":"10",
				"unrealisedPnl":"1.99",
				"cumRealisedPnl":"5.50",
				"positionValue":"540.01",
				"updatedTime":"1700000002000"
			}],
			"nextPageCursor":""
		},
		"retExtInfo":{},"time":0
	}`
	var _, client = mockBybit(t, map[string]string{
		"/v5/position/list": fixture,
	})
	var positions []types.PositionInfo
	var err error
	positions, err = linearsOf(client).Account().GetPosition(context.Background(), "BTCUSDT")
	if err != nil {
		t.Fatalf("GetPosition: %v", err)
	}
	if len(positions) != 1 {
		t.Fatalf("positions len: got %d", len(positions))
	}
	var p types.PositionInfo = positions[0]
	if p.Side != types.SideTypeBuy {
		t.Fatalf("Side: got %q", p.Side)
	}
	if !p.Quantity.Equal(dq("0.020")) {
		t.Fatalf("Quantity: got %v", p.Quantity)
	}
	if !p.AvgEntryPrice.Equal(dq("27000.50")) {
		t.Fatalf("AvgEntryPrice: got %v", p.AvgEntryPrice)
	}
	if p.UpdatedAtMs != 1700000002000 {
		t.Fatalf("UpdatedAtMs: got %d", p.UpdatedAtMs)
	}
}

func TestContract_GetOpenOrders_SinglePage(t *testing.T) {
	t.Parallel()
	const fixture = `{
		"retCode":0,"retMsg":"OK",
		"result":{
			"category":"linear",
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
				"avgPrice":"0",
				"cumExecFee":"0",
				"orderStatus":"New",
				"positionIdx":0,
				"reduceOnly":false,
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
	var orders []types.OrderInfo
	var err error
	orders, err = linearsOf(client).Account().GetOpenOrders(context.Background(), "BTCUSDT")
	if err != nil {
		t.Fatalf("GetOpenOrders: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("len: got %d", len(orders))
	}
	if orders[0].OrderID != "ord-1" {
		t.Fatalf("OrderID: got %q", orders[0].OrderID)
	}
	if orders[0].Status != types.OrderStatusNew {
		t.Fatalf("Status: got %q", orders[0].Status)
	}
	if orders[0].CreatedAtMs != 1700000003000 {
		t.Fatalf("CreatedAtMs: got %d", orders[0].CreatedAtMs)
	}
	if orders[0].RejectReason != "" {
		t.Fatalf("RejectReason: got %q, want empty (EC_NoError must be masked)", orders[0].RejectReason)
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
	var info types.OrderInfo
	var err error
	info, err = linearsOf(client).Trading().CreateOrder(context.Background(), types.CreateOrderRequest{
		Symbol:        "BTCUSDT",
		Side:          types.SideTypeBuy,
		OrderType:     types.OrderTypeLimit,
		TimeInForce:   types.TimeInForceGTC,
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
	if info.Status != types.OrderStatusNew {
		t.Fatalf("Status: got %q", info.Status)
	}
	if got, ok := linearsOf(client).Trading().OrderIDByClientID("link-42"); !ok || got != "o-42" {
		t.Fatalf("mapping not stored: got %q ok=%v", got, ok)
	}
	if info.RateLimits["X-Bapi-Limit-Status"] != "99" {
		t.Fatalf("RateLimits not propagated: %v", info.RateLimits)
	}
}

func TestContract_CreateOrder_RetCodeReject(t *testing.T) {
	t.Parallel()
	const fixture = `{
		"retCode":110007,"retMsg":"insufficient available balance",
		"result":{},"retExtInfo":{},"time":0
	}`
	var _, client = mockBybit(t, map[string]string{
		"/v5/order/create": fixture,
	})
	var _, err = linearsOf(client).Trading().CreateOrder(context.Background(), types.CreateOrderRequest{
		Symbol:    "BTCUSDT",
		Side:      types.SideTypeBuy,
		OrderType: types.OrderTypeLimit,
		Quantity:  dq("100"),
		Price:     dq("100000"),
	})
	if err == nil {
		t.Fatalf("expected an error, got nil")
	}
	if !bybit.IsExchange(err) {
		t.Fatalf("expected ErrorKindExchange, got %v", err)
	}
	if !strings.Contains(err.Error(), "110007") {
		t.Fatalf("error must surface the Bybit code 110007, got %v", err)
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
	var results []types.BatchOrderResult
	var err error
	results, err = linearsOf(client).Trading().CancelAllOrders(context.Background(), "BTCUSDT")
	if err != nil {
		t.Fatalf("CancelAllOrders: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("len: got %d", len(results))
	}
	if results[0].Order.OrderID != "o-1" || results[0].Order.Status != types.OrderStatusCancelled {
		t.Fatalf("results[0]: got %+v", results[0])
	}
}

func TestContract_CreateBatchOrders_PartialFailure(t *testing.T) {
	t.Parallel()
	const fixture = `{
		"retCode":0,"retMsg":"OK",
		"result":{"list":[
			{"category":"linear","symbol":"BTCUSDT","orderId":"o-1","orderLinkId":"link-1","createAt":"1700000004000"},
			{"category":"linear","symbol":"BTCUSDT","orderId":"","orderLinkId":"","createAt":""}
		]},
		"retExtInfo":{"list":[
			{"code":0,"msg":"OK"},
			{"code":110007,"msg":"insufficient available balance"}
		]},
		"time":0
	}`
	var _, client = mockBybit(t, map[string]string{
		"/v5/order/create-batch": fixture,
	})
	var results []types.BatchOrderResult
	var err error
	results, err = linearsOf(client).Trading().CreateBatchOrders(context.Background(), []types.CreateOrderRequest{
		{Symbol: "BTCUSDT", Side: types.SideTypeBuy, OrderType: types.OrderTypeLimit, Quantity: dq("0.001"), Price: dq("27000"), ClientOrderID: "link-1"},
		{Symbol: "BTCUSDT", Side: types.SideTypeBuy, OrderType: types.OrderTypeLimit, Quantity: dq("0.001"), Price: dq("27000"), ClientOrderID: "link-2"},
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
	if results[1].Code != 110007 {
		t.Fatalf("results[1].Code: got %d", results[1].Code)
	}
}
