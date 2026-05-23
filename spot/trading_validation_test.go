/*
FILE: spot/trading_validation_test.go

DESCRIPTION:
Pure unit tests for the spot trading body builders. They do NOT touch
the REST transport — they verify only that:

  - SDK-level invariants are enforced before hitting the wire;
  - the wire body shape (keys + values) matches Bybit V5 spot
    expectations.

Coverage:
  - buildCreateOrderBody : invalid inputs reject locally with
                           ErrorKindInvalidRequest; valid inputs
                           produce the correct map.
  - buildModifyOrderBody : same.
  - buildCancelOrderBody : same.
  - resolveOrderType / resolveTimeInForce mappings.
  - orderLinkIDPattern  : alphanumeric + ./_/- only, 1..36 chars.
  - filterOrderNotModifiedFromBatchErr : drops 10001 from joined errors.
*/

package spot

import (
	"errors"
	"testing"

	"github.com/shopspring/decimal"

	bybit "github.com/tonymontanov/go-bybit/v2"
	bybitspottypes "github.com/tonymontanov/go-bybit/v2/spot/types"
)

func dq(s string) decimal.Decimal {
	var d decimal.Decimal
	var err error
	d, err = decimal.NewFromString(s)
	if err != nil {
		panic(err)
	}
	return d
}

func TestBuildCreateOrderBody_Validation(t *testing.T) {
	t.Parallel()
	var trader *TradingClient = newTradingClient(nil)

	type tcase struct {
		name string
		req  bybitspottypes.CreateOrderRequest
	}
	var cases []tcase = []tcase{
		{
			name: "empty symbol",
			req:  bybitspottypes.CreateOrderRequest{Side: bybitspottypes.SideTypeBuy, Quantity: dq("1"), Price: dq("100")},
		},
		{
			name: "missing side",
			req:  bybitspottypes.CreateOrderRequest{Symbol: "BTCUSDT", Quantity: dq("1"), Price: dq("100")},
		},
		{
			name: "zero quantity",
			req:  bybitspottypes.CreateOrderRequest{Symbol: "BTCUSDT", Side: bybitspottypes.SideTypeBuy, Price: dq("100")},
		},
		{
			name: "negative quantity",
			req:  bybitspottypes.CreateOrderRequest{Symbol: "BTCUSDT", Side: bybitspottypes.SideTypeBuy, Quantity: dq("-1"), Price: dq("100")},
		},
		{
			name: "limit without price",
			req:  bybitspottypes.CreateOrderRequest{Symbol: "BTCUSDT", Side: bybitspottypes.SideTypeBuy, Quantity: dq("1"), OrderType: bybitspottypes.OrderTypeLimit},
		},
		{
			name: "bad orderLinkId chars",
			req:  bybitspottypes.CreateOrderRequest{Symbol: "BTCUSDT", Side: bybitspottypes.SideTypeBuy, Quantity: dq("1"), Price: dq("100"), ClientOrderID: "bad space"},
		},
		{
			name: "orderLinkId too long",
			req: bybitspottypes.CreateOrderRequest{
				Symbol: "BTCUSDT", Side: bybitspottypes.SideTypeBuy, Quantity: dq("1"), Price: dq("100"),
				ClientOrderID: "abcdefghijklmnopqrstuvwxyz0123456789abcd",
			},
		},
	}
	var tc tcase
	for _, tc = range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var _, err = trader.buildCreateOrderBody(tc.req)
			if !bybit.IsInvalidRequest(err) {
				t.Fatalf("expected ErrorKindInvalidRequest, got %v", err)
			}
		})
	}
}

func TestBuildCreateOrderBody_HappyPath_Limit(t *testing.T) {
	t.Parallel()
	var trader *TradingClient = newTradingClient(nil)
	var body, err = trader.buildCreateOrderBody(bybitspottypes.CreateOrderRequest{
		Symbol:        "BTCUSDT",
		Side:          bybitspottypes.SideTypeBuy,
		OrderType:     bybitspottypes.OrderTypeLimit,
		TimeInForce:   bybitspottypes.TimeInForcePostOnly,
		Quantity:      dq("0.001"),
		Price:         dq("27000"),
		ClientOrderID: "test_order-1.id",
	})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if body["category"] != "spot" {
		t.Errorf("category: got %v", body["category"])
	}
	if body["symbol"] != "BTCUSDT" {
		t.Errorf("symbol: got %v", body["symbol"])
	}
	if body["side"] != "Buy" {
		t.Errorf("side: got %v", body["side"])
	}
	if body["orderType"] != "Limit" {
		t.Errorf("orderType: got %v", body["orderType"])
	}
	if body["timeInForce"] != "PostOnly" {
		t.Errorf("timeInForce: got %v", body["timeInForce"])
	}
	if body["qty"] != "0.001" {
		t.Errorf("qty: got %v", body["qty"])
	}
	if body["price"] != "27000" {
		t.Errorf("price: got %v", body["price"])
	}
	if body["orderLinkId"] != "test_order-1.id" {
		t.Errorf("orderLinkId: got %v", body["orderLinkId"])
	}
	// PositionIdx / ReduceOnly must NOT appear on spot bodies.
	if _, ok := body["positionIdx"]; ok {
		t.Errorf("positionIdx must not be present on spot")
	}
	if _, ok := body["reduceOnly"]; ok {
		t.Errorf("reduceOnly must not be present on spot")
	}
}

func TestBuildCreateOrderBody_HappyPath_MarketBuyQuoteCoin(t *testing.T) {
	t.Parallel()
	var trader *TradingClient = newTradingClient(nil)
	var body, err = trader.buildCreateOrderBody(bybitspottypes.CreateOrderRequest{
		Symbol:     "BTCUSDT",
		Side:       bybitspottypes.SideTypeBuy,
		OrderType:  bybitspottypes.OrderTypeMarket,
		Quantity:   dq("100"),
		MarketUnit: bybitspottypes.MarketUnitQuoteCoin,
	})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if body["orderType"] != "Market" {
		t.Errorf("orderType: got %v", body["orderType"])
	}
	if _, ok := body["price"]; ok {
		t.Errorf("price must not appear on Market orders, got %v", body["price"])
	}
	if body["marketUnit"] != "quoteCoin" {
		t.Errorf("marketUnit: got %v", body["marketUnit"])
	}
}

func TestBuildCreateOrderBody_IsLeverageMargin(t *testing.T) {
	t.Parallel()
	var trader *TradingClient = newTradingClient(nil)
	var body, err = trader.buildCreateOrderBody(bybitspottypes.CreateOrderRequest{
		Symbol:     "BTCUSDT",
		Side:       bybitspottypes.SideTypeSell,
		OrderType:  bybitspottypes.OrderTypeLimit,
		Quantity:   dq("0.5"),
		Price:      dq("100000"),
		IsLeverage: true,
	})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	// Bybit expects the integer 1 for margin spot in UTA — not bool.
	if body["isLeverage"] != 1 {
		t.Errorf("isLeverage must be int 1 for margin spot, got %T %v", body["isLeverage"], body["isLeverage"])
	}
}

func TestBuildModifyOrderBody_Validation(t *testing.T) {
	t.Parallel()

	type tcase struct {
		name string
		req  bybitspottypes.ModifyOrderRequest
	}
	var cases []tcase = []tcase{
		{name: "empty symbol", req: bybitspottypes.ModifyOrderRequest{OrderID: "x", NewPrice: dq("1")}},
		{name: "no id", req: bybitspottypes.ModifyOrderRequest{Symbol: "BTCUSDT", NewPrice: dq("1")}},
		{name: "both ids", req: bybitspottypes.ModifyOrderRequest{Symbol: "BTCUSDT", OrderID: "x", ClientOrderID: "y", NewPrice: dq("1")}},
		{name: "no fields to amend", req: bybitspottypes.ModifyOrderRequest{Symbol: "BTCUSDT", OrderID: "x"}},
	}
	var tc tcase
	for _, tc = range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var _, err = buildModifyOrderBody(tc.req)
			if !bybit.IsInvalidRequest(err) {
				t.Fatalf("expected ErrorKindInvalidRequest, got %v", err)
			}
		})
	}
}

func TestBuildModifyOrderBody_HappyPath(t *testing.T) {
	t.Parallel()
	var body, err = buildModifyOrderBody(bybitspottypes.ModifyOrderRequest{
		Symbol:      "BTCUSDT",
		OrderID:     "abc",
		NewQuantity: dq("0.1"),
		NewPrice:    dq("27500"),
	})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if body["category"] != "spot" {
		t.Errorf("category: got %v", body["category"])
	}
	if body["orderId"] != "abc" {
		t.Errorf("orderId: got %v", body["orderId"])
	}
	if body["qty"] != "0.1" {
		t.Errorf("qty: got %v", body["qty"])
	}
	if body["price"] != "27500" {
		t.Errorf("price: got %v", body["price"])
	}
}

func TestBuildCancelOrderBody_Validation(t *testing.T) {
	t.Parallel()
	type tcase struct {
		name string
		req  bybitspottypes.CancelOrderRequest
	}
	var cases []tcase = []tcase{
		{name: "empty symbol", req: bybitspottypes.CancelOrderRequest{OrderID: "x"}},
		{name: "no ids", req: bybitspottypes.CancelOrderRequest{Symbol: "BTCUSDT"}},
		{name: "both ids", req: bybitspottypes.CancelOrderRequest{Symbol: "BTCUSDT", OrderID: "x", ClientOrderID: "y"}},
	}
	var tc tcase
	for _, tc = range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var _, err = buildCancelOrderBody(tc.req)
			if !bybit.IsInvalidRequest(err) {
				t.Fatalf("expected ErrorKindInvalidRequest, got %v", err)
			}
		})
	}
}

func TestOrderLinkIDPattern(t *testing.T) {
	t.Parallel()
	var ok = []string{"a", "abc-123_x.y", "ABCDEFG", "1.2.3-4_5"}
	var bad = []string{"", "foo bar", "a/b", "a+b", "abcdefghijklmnopqrstuvwxyz0123456789ABCDE"} // last is 41 chars
	var i int
	for i = 0; i < len(ok); i++ {
		if !orderLinkIDPattern.MatchString(ok[i]) {
			t.Errorf("orderLinkIDPattern rejected valid id %q", ok[i])
		}
	}
	for i = 0; i < len(bad); i++ {
		if orderLinkIDPattern.MatchString(bad[i]) {
			t.Errorf("orderLinkIDPattern accepted invalid id %q", bad[i])
		}
	}
}

func TestFilterOrderNotModifiedFromBatchErr(t *testing.T) {
	t.Parallel()
	var notMod *bybit.Error = &bybit.Error{Kind: bybit.ErrorKindInvalidRequest, BybitCode: "10001", Message: "order not modified"}
	var other *bybit.Error = &bybit.Error{Kind: bybit.ErrorKindInvalidRequest, BybitCode: "10002", Message: "rejected"}
	var nonBybit error = errors.New("local validation failure")

	type tcase struct {
		name    string
		err     error
		wantNil bool
	}
	var cases []tcase = []tcase{
		{name: "single 10001", err: notMod, wantNil: true},
		{name: "single non-10001", err: other, wantNil: false},
		{name: "joined all 10001", err: errors.Join(notMod, notMod), wantNil: true},
		{name: "joined mix drops 10001", err: errors.Join(notMod, other), wantNil: false},
		{name: "joined non-bybit + 10001", err: errors.Join(nonBybit, notMod), wantNil: false},
		{name: "nil", err: nil, wantNil: true},
	}
	var tc tcase
	for _, tc = range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var got = filterOrderNotModifiedFromBatchErr(tc.err)
			if (got == nil) != tc.wantNil {
				t.Fatalf("filterOrderNotModifiedFromBatchErr: got %v, want nil=%v", got, tc.wantNil)
			}
		})
	}
}

func TestClampOrderbookDepth_Spot(t *testing.T) {
	t.Parallel()
	type tc struct {
		in   int
		want int
	}
	var cases []tc = []tc{
		{0, 50},   // default for ≤0
		{-5, 50},  // default for ≤0
		{1, 1},    // exact
		{30, 50},  // clamps up
		{200, 200},
		{500, 200}, // clamps down to 200 (spot max)
	}
	var c tc
	for _, c = range cases {
		var got = clampOrderbookDepth(c.in)
		if got != c.want {
			t.Errorf("clampOrderbookDepth(%d) = %d, want %d", c.in, got, c.want)
		}
	}
}
