/*
FILE: spot/types/enums.go

DESCRIPTION:
Closed enums (typed strings) used by the Bybit V5 spot category. Values
match the wire format Bybit accepts and returns — keep them exact, or
the exchange rejects with retCode 10001.

DIFFERENCES vs LINEARS:
  - No `PositionIdx`, no `PositionMode` (spot has no positions).
  - `MarginTrading` enum captures Bybit's instrument-level marginTrading
    string ("none", "both", "utaOnly", "normalSpotOnly").
  - All other enums (Side / OrderType / TimeInForce / OrderStatus /
    Category) are byte-identical to linears so the two profiles speak
    the same wire dialect.
*/

package types

// Category is the Bybit V5 product family identifier. The spot
// profile is hard-pinned to "spot" but the constant set is duplicated
// here so call sites can stay package-local.
type Category string

const (
	// CategoryLinear — USDT/USDC perpetual + USDC futures (declared for
	// cross-reference; spot profile never uses it).
	CategoryLinear Category = "linear"
	// CategorySpot — spot — the value the spot profile pins.
	CategorySpot Category = "spot"
	// CategoryInverse — coin-margined contracts.
	CategoryInverse Category = "inverse"
	// CategoryOption — options.
	CategoryOption Category = "option"
)

// SideType — order direction on the exchange wire.
type SideType string

const (
	// SideTypeBuy — buy.
	SideTypeBuy SideType = "Buy"
	// SideTypeSell — sell.
	SideTypeSell SideType = "Sell"
)

// OrderType — order execution model on the exchange wire.
type OrderType string

const (
	// OrderTypeLimit — limit order.
	OrderTypeLimit OrderType = "Limit"
	// OrderTypeMarket — market order.
	OrderTypeMarket OrderType = "Market"
)

// TimeInForceType — order expiry / queue behaviour. The PostOnly variant
// is recognised by Bybit V5 only when paired with orderType=Limit; the
// SDK applies that mapping in trading.go.
type TimeInForceType string

const (
	// TimeInForceGTC — Good Till Cancel (default).
	TimeInForceGTC TimeInForceType = "GTC"
	// TimeInForceIOC — Immediate or Cancel.
	TimeInForceIOC TimeInForceType = "IOC"
	// TimeInForceFOK — Fill or Kill.
	TimeInForceFOK TimeInForceType = "FOK"
	// TimeInForcePostOnly — post-only (rejected if it would cross the
	// book). Maps to orderType=Limit + timeInForce=PostOnly on the wire.
	TimeInForcePostOnly TimeInForceType = "PostOnly"
)

// OrderStatus is a subset of the Bybit V5 spot order states. Bybit
// occasionally extends the catalogue; values outside this list are
// returned verbatim in OrderInfo.Status — callers that need finer
// discrimination should read the raw status string.
type OrderStatus string

const (
	OrderStatusNew             OrderStatus = "New"
	OrderStatusPartiallyFilled OrderStatus = "PartiallyFilled"
	OrderStatusFilled          OrderStatus = "Filled"
	OrderStatusCancelled       OrderStatus = "Cancelled"
	OrderStatusRejected        OrderStatus = "Rejected"
)

// MarginTrading describes whether a spot symbol can be margin-traded
// and on which account type. Comes from instruments-info.marginTrading.
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
