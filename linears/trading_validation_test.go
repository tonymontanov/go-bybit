/*
FILE: linears/trading_validation_test.go

DESCRIPTION:
Pure unit tests for the request-body builders in trading.go. These tests
do NOT touch the REST transport — they verify only that:
  - SDK-level invariants are enforced before hitting the wire;
  - the wire body shape (keys + values) matches Bybit V5 expectations.

Coverage:
  - buildCreateOrderBody : invalid inputs reject locally with
                           ErrorKindInvalidRequest; valid inputs produce
                           the correct map.
  - buildModifyOrderBody : same.
  - buildCancelOrderBody : same.
  - resolveOrderType / resolveTimeInForce mappings.
  - orderLinkIDPattern   : alphanumeric + ./_/- only, 1..36 chars.
*/

package linears

import (
	"testing"

	"github.com/shopspring/decimal"
	bybit "github.com/tonymontanov/go-bybit"
	"github.com/tonymontanov/go-bybit/linears/types"
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
		req  types.CreateOrderRequest
	}
	var cases []tcase = []tcase{
		{
			name: "empty symbol",
			req:  types.CreateOrderRequest{Side: types.SideTypeBuy, Quantity: dq("1"), Price: dq("100")},
		},
		{
			name: "missing side",
			req:  types.CreateOrderRequest{Symbol: "BTCUSDT", Quantity: dq("1"), Price: dq("100")},
		},
		{
			name: "zero quantity",
			req:  types.CreateOrderRequest{Symbol: "BTCUSDT", Side: types.SideTypeBuy, Price: dq("100")},
		},
		{
			name: "limit without price",
			req:  types.CreateOrderRequest{Symbol: "BTCUSDT", Side: types.SideTypeBuy, Quantity: dq("1"), OrderType: types.OrderTypeLimit},
		},
		{
			name: "bad orderLinkId chars",
			req:  types.CreateOrderRequest{Symbol: "BTCUSDT", Side: types.SideTypeBuy, Quantity: dq("1"), Price: dq("100"), ClientOrderID: "bad space"},
		},
		{
			name: "orderLinkId too long",
			req: types.CreateOrderRequest{
				Symbol: "BTCUSDT", Side: types.SideTypeBuy, Quantity: dq("1"), Price: dq("100"),
				ClientOrderID: "abcdefghijklmnopqrstuvwxyz0123456789abcd", // 40 chars
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
	var body, err = trader.buildCreateOrderBody(types.CreateOrderRequest{
		Symbol:        "BTCUSDT",
		Side:          types.SideTypeBuy,
		OrderType:     types.OrderTypeLimit,
		TimeInForce:   types.TimeInForcePostOnly,
		Quantity:      dq("0.001"),
		Price:         dq("27000"),
		ClientOrderID: "test_order-1.id",
		ReduceOnly:    true,
		PositionIdx:   types.PositionIdxHedgeBuy,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := body["category"], string(types.CategoryLinear); got != want {
		t.Fatalf("category: got %v, want %v", got, want)
	}
	if got, want := body["symbol"], "BTCUSDT"; got != want {
		t.Fatalf("symbol: got %v, want %v", got, want)
	}
	if got, want := body["side"], "Buy"; got != want {
		t.Fatalf("side: got %v, want %v", got, want)
	}
	if got, want := body["orderType"], "Limit"; got != want {
		t.Fatalf("orderType: got %v, want %v", got, want)
	}
	if got, want := body["qty"], "0.001"; got != want {
		t.Fatalf("qty: got %v, want %v", got, want)
	}
	if got, want := body["price"], "27000"; got != want {
		t.Fatalf("price: got %v, want %v", got, want)
	}
	if got, want := body["timeInForce"], "PostOnly"; got != want {
		t.Fatalf("timeInForce: got %v, want %v", got, want)
	}
	if got, want := body["orderLinkId"], "test_order-1.id"; got != want {
		t.Fatalf("orderLinkId: got %v, want %v", got, want)
	}
	if got := body["reduceOnly"]; got != true {
		t.Fatalf("reduceOnly: got %v", got)
	}
	if got := body["positionIdx"]; got != int(types.PositionIdxHedgeBuy) {
		t.Fatalf("positionIdx: got %v", got)
	}
}

func TestBuildCreateOrderBody_Market_NoPriceKey(t *testing.T) {
	t.Parallel()
	var trader *TradingClient = newTradingClient(nil)
	var body, err = trader.buildCreateOrderBody(types.CreateOrderRequest{
		Symbol:    "BTCUSDT",
		Side:      types.SideTypeSell,
		OrderType: types.OrderTypeMarket,
		Quantity:  dq("0.5"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := body["price"]; ok {
		t.Fatalf("market order body must NOT carry a price key, got %v", body)
	}
	if _, ok := body["timeInForce"]; ok {
		t.Fatalf("market order without TIF must NOT carry timeInForce; let Bybit default apply, got %v", body)
	}
}

func TestBuildModifyOrderBody_Validation(t *testing.T) {
	t.Parallel()
	type tcase struct {
		name string
		req  types.ModifyOrderRequest
	}
	var cases []tcase = []tcase{
		{name: "empty symbol", req: types.ModifyOrderRequest{NewQuantity: dq("1")}},
		{name: "no identifier", req: types.ModifyOrderRequest{Symbol: "BTCUSDT", NewQuantity: dq("1")}},
		{
			name: "both identifiers",
			req:  types.ModifyOrderRequest{Symbol: "BTCUSDT", OrderID: "x", ClientOrderID: "y", NewQuantity: dq("1")},
		},
		{name: "no fields to change", req: types.ModifyOrderRequest{Symbol: "BTCUSDT", OrderID: "x"}},
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
	var body, err = buildModifyOrderBody(types.ModifyOrderRequest{
		Symbol:      "BTCUSDT",
		OrderID:     "abc123",
		NewPrice:    dq("28100.5"),
		NewQuantity: dq("0.002"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := body["orderId"]; got != "abc123" {
		t.Fatalf("orderId: got %v", got)
	}
	if got := body["price"]; got != "28100.5" {
		t.Fatalf("price: got %v", got)
	}
	if got := body["qty"]; got != "0.002" {
		t.Fatalf("qty: got %v", got)
	}
	if _, ok := body["orderLinkId"]; ok {
		t.Fatalf("orderLinkId must be absent when only OrderID set, got %v", body)
	}
}

func TestBuildCancelOrderBody_Validation(t *testing.T) {
	t.Parallel()
	type tcase struct {
		name string
		req  types.CancelOrderRequest
	}
	var cases []tcase = []tcase{
		{name: "empty symbol", req: types.CancelOrderRequest{OrderID: "x"}},
		{name: "no identifier", req: types.CancelOrderRequest{Symbol: "BTCUSDT"}},
		{name: "both identifiers", req: types.CancelOrderRequest{Symbol: "BTCUSDT", OrderID: "x", ClientOrderID: "y"}},
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
	type tcase struct {
		v    string
		want bool
	}
	var cases []tcase = []tcase{
		{v: "", want: false},
		{v: "abc", want: true},
		{v: "abc-1.2_3", want: true},
		{v: "abc def", want: false},
		{v: "abc/def", want: false},
		{v: "abcdefghijklmnopqrstuvwxyz0123456789", want: true},   // 36 chars
		{v: "abcdefghijklmnopqrstuvwxyz0123456789X", want: false}, // 37
	}
	var tc tcase
	for _, tc = range cases {
		var got = orderLinkIDPattern.MatchString(tc.v)
		if got != tc.want {
			t.Errorf("orderLinkIDPattern.MatchString(%q): got %v, want %v", tc.v, got, tc.want)
		}
	}
}

func TestClampOrderbookDepth(t *testing.T) {
	t.Parallel()
	type tcase struct {
		in   int
		want int
	}
	var cases []tcase = []tcase{
		{in: 0, want: 50},
		{in: -10, want: 50},
		{in: 1, want: 1},
		{in: 25, want: 50},
		{in: 50, want: 50},
		{in: 75, want: 200},
		{in: 200, want: 200},
		{in: 300, want: 500},
		{in: 500, want: 500},
		{in: 1000, want: 500},
	}
	var tc tcase
	for _, tc = range cases {
		var got = clampOrderbookDepth(tc.in)
		if got != tc.want {
			t.Errorf("clampOrderbookDepth(%d) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

// joinUpper is in account.go but trivial enough to test alongside the
// trading helpers — it lives in the linears package and is pure.
func TestJoinUpper(t *testing.T) {
	t.Parallel()
	if got := joinUpper(nil); got != "" {
		t.Fatalf("nil input: got %q", got)
	}
	if got := joinUpper([]string{"btc", "USDT"}); got != "BTC,USDT" {
		t.Fatalf("got %q", got)
	}
}

func TestNormalizeRejectReason(t *testing.T) {
	t.Parallel()

	type tc struct {
		in   string
		want string
	}
	var cases []tc = []tc{
		{"", ""},
		{"EC_NoError", ""},
		{"EC_PostOnlyWillTakeLiquidity", "EC_PostOnlyWillTakeLiquidity"},
		{"EC_PerCancelRequest", "EC_PerCancelRequest"},
		{"EC_NoErrorish", "EC_NoErrorish"},
	}
	var i int
	for i = 0; i < len(cases); i++ {
		var c tc = cases[i]
		var got string = normalizeRejectReason(c.in)
		if got != c.want {
			t.Fatalf("normalizeRejectReason(%q): got %q, want %q", c.in, got, c.want)
		}
	}
}

