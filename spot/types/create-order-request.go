/*
FILE: spot/types/create-order-request.go

DESCRIPTION:
Order creation request for the Bybit V5 spot category.

DIFFERENCES vs LINEARS:
  - No `PositionIdx` / `ReduceOnly` / `CloseOnTrigger` (spot has no
    positions to reduce or close).
  - For Market BUY orders Bybit V5 spot interprets `Quantity` as the
    QUOTE amount (USDT) by default — see MarketUnit below.
  - Adds `IsLeverage` flag for margin-spot in UTA accounts (0 = spot
    cash, 1 = spot margin). When unset (default) the SDK omits the
    parameter, which Bybit treats as 0.

FIELDS:
  - Symbol         — e.g. "BTCUSDT".
  - Side           — Buy / Sell.
  - OrderType      — Limit / Market.
  - TimeInForce    — GTC / IOC / FOK / PostOnly. Empty defaults: GTC for
                     Limit, IOC for Market.
  - Quantity       — order quantity. Interpretation depends on
                     (OrderType, Side, MarketUnit) — see MarketUnit.
  - Price          — limit price; required for Limit/PostOnly, ignored
                     for Market.
  - ClientOrderID  — Bybit `orderLinkId` (≤ 36 chars, [A-Za-z0-9_.-]).
  - MarketUnit     — only used for Market orders, see MarketUnit constants.
  - IsLeverage     — UTA margin-spot flag (false = cash spot).

INVARIANTS:
  - Quantity > 0 always; the SDK validates locally.
  - For Limit / PostOnly: Price must be > 0.
*/

package types

import "github.com/shopspring/decimal"

// MarketUnit selects how Bybit interprets `Quantity` on Market orders:
//
//   - MarketUnitBaseCoin  ("baseCoin")  — Quantity is in BASE asset.
//   - MarketUnitQuoteCoin ("quoteCoin") — Quantity is in QUOTE asset.
//
// Empty / unset on Market BUY → Bybit defaults to "quoteCoin" (you
// supply how much USDT to spend).
//
// Empty / unset on Market SELL → Bybit defaults to "baseCoin" (you
// supply how much BTC to dump).
//
// Set explicitly when the caller wants the inverse interpretation.
type MarketUnit string

const (
	MarketUnitBaseCoin  MarketUnit = "baseCoin"
	MarketUnitQuoteCoin MarketUnit = "quoteCoin"
)

// CreateOrderRequest — order creation request for the spot category.
type CreateOrderRequest struct {
	Symbol        string
	Side          SideType
	OrderType     OrderType
	TimeInForce   TimeInForceType
	Quantity      decimal.Decimal
	Price         decimal.Decimal
	ClientOrderID string
	MarketUnit    MarketUnit
	IsLeverage    bool
}
