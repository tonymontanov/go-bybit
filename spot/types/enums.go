/*
FILE: spot/types/enums.go

DESCRIPTION:
Closed enums (typed strings) used by the Bybit V5 spot profile. Most
values are PROTOCOL-COMMON and re-exported from the neutral
`github.com/tonymontanov/go-bybit/v2/types` package via Go type
aliases (`type X = commontypes.X`) — this preserves type identity, so
`spot/types.SideTypeBuy` and `linears/types.SideTypeBuy` are the same
type at the language level (both are `commontypes.SideType`).

Spot-only enum declared HERE:
  - MarginTrading — instrument-level margin-trading flag (Bybit's
    `marginTrading` from /v5/market/instruments-info, values
    "none" / "both" / "utaOnly" / "normalSpotOnly"). Linears does not
    surface this field.

Spot does NOT have positions, so `PositionIdx` / `PositionMode` from
linears/types are intentionally not present here.
*/

package types

import commontypes "github.com/tonymontanov/go-bybit/v2/types"

// Category is the Bybit V5 product family identifier. The spot
// profile pins `spot`; the constant set is re-exported so call sites
// can stay package-local.
type Category = commontypes.Category

const (
	// CategoryLinear — USDT/USDC perpetual + USDC futures (declared
	// for cross-reference; spot profile never uses it).
	CategoryLinear = commontypes.CategoryLinear
	// CategorySpot — spot — the value the spot profile pins.
	CategorySpot = commontypes.CategorySpot
	// CategoryInverse — coin-margined contracts.
	CategoryInverse = commontypes.CategoryInverse
	// CategoryOption — options.
	CategoryOption = commontypes.CategoryOption
)

// SideType — order direction on the exchange wire.
type SideType = commontypes.SideType

const (
	// SideTypeBuy — buy.
	SideTypeBuy = commontypes.SideTypeBuy
	// SideTypeSell — sell.
	SideTypeSell = commontypes.SideTypeSell
)

// OrderType — order execution model on the exchange wire.
type OrderType = commontypes.OrderType

const (
	// OrderTypeLimit — limit order.
	OrderTypeLimit = commontypes.OrderTypeLimit
	// OrderTypeMarket — market order.
	OrderTypeMarket = commontypes.OrderTypeMarket
)

// TimeInForceType — order expiry / queue behaviour.
type TimeInForceType = commontypes.TimeInForceType

const (
	// TimeInForceGTC — Good Till Cancel (default).
	TimeInForceGTC = commontypes.TimeInForceGTC
	// TimeInForceIOC — Immediate or Cancel.
	TimeInForceIOC = commontypes.TimeInForceIOC
	// TimeInForceFOK — Fill or Kill.
	TimeInForceFOK = commontypes.TimeInForceFOK
	// TimeInForcePostOnly — post-only (rejected if it would cross the
	// book). Maps to orderType=Limit + timeInForce=PostOnly on the wire.
	TimeInForcePostOnly = commontypes.TimeInForcePostOnly
)

// OrderStatus — Bybit V5 spot order state. Re-exports the common
// base catalogue; the trigger-order specific values
// (Untriggered / Triggered) are not applicable to spot and are NOT
// declared here.
//
// Bybit occasionally extends the catalogue; values outside this list
// are returned verbatim in OrderInfo.Status.
type OrderStatus = commontypes.OrderStatus

const (
	// OrderStatusNew — accepted by the matcher, untriggered.
	OrderStatusNew = commontypes.OrderStatusNew
	// OrderStatusPartiallyFilled — partially filled, remainder live.
	OrderStatusPartiallyFilled = commontypes.OrderStatusPartiallyFilled
	// OrderStatusFilled — fully filled.
	OrderStatusFilled = commontypes.OrderStatusFilled
	// OrderStatusCancelled — cancelled.
	OrderStatusCancelled = commontypes.OrderStatusCancelled
	// OrderStatusRejected — rejected by the exchange before reaching the book.
	OrderStatusRejected = commontypes.OrderStatusRejected
)

// MarginTrading describes whether a spot symbol can be margin-traded
// and on which account type. Comes from instruments-info.marginTrading.
//
// SPOT-ONLY: the linears profile does not surface this field.
//
// Spot trading on Bybit V5 supports two account models:
//
//   - "normalSpotOnly" — classic non-margin spot.
//   - "utaOnly"        — Unified Trading Account margin spot.
//   - "both"           — supported on classic and UTA.
//   - "none"           — symbol does not support margin trading at all
//     (still tradeable as plain spot when the instrument is "Trading").
type MarginTrading string

const (
	MarginTradingNone           MarginTrading = "none"
	MarginTradingBoth           MarginTrading = "both"
	MarginTradingUTAOnly        MarginTrading = "utaOnly"
	MarginTradingNormalSpotOnly MarginTrading = "normalSpotOnly"
)
