/*
FILE: linears/types/enums.go

DESCRIPTION:
Closed enums (typed strings) used by the Bybit V5 linear category.
The values match the wire format Bybit accepts and returns — keep them
exact, or the exchange will reject the request with retCode 10001.

GROUPING:
  - Side (Buy/Sell)
  - OrderType (Market/Limit)
  - TimeInForce (GTC/IOC/FOK/PostOnly)
  - PositionIdx (one-way / hedge-buy / hedge-sell)
  - PositionMode (one-way / hedge)
  - OrderStatus (subset, see file body for full Bybit list)
  - Category — fixed to "linear" in this package; declared here so that
    the rate-limiter / observers can use the same constant.
  - TriggerBy / TpslMode — not in v1 scope; intentionally omitted.

Bybit V5 also publishes orderType=PostOnly for spot, and timeInForce=
PostOnly for derivatives — for symmetry with the desk-side enum
("GTX = post-only"), we expose TimeInForcePostOnly. CreateOrder maps
TimeInForcePostOnly → orderType=Limit + timeInForce=PostOnly when sending.
*/

package types

// Category is the Bybit V5 product family identifier sent on every REST
// call. The linears profile is hard-pinned to "linear".
type Category string

const (
	// CategoryLinear — USDT/USDC perpetual + USDC futures.
	CategoryLinear Category = "linear"
	// CategoryInverse — coin-margined contracts. Reserved for a later profile.
	CategoryInverse Category = "inverse"
	// CategorySpot — spot. Reserved for a later profile.
	CategorySpot Category = "spot"
	// CategoryOption — options. Reserved for a later profile.
	CategoryOption Category = "option"
)

// SideType — order direction on the exchange wire.
type SideType string

const (
	// SideTypeBuy — buy / long.
	SideTypeBuy SideType = "Buy"
	// SideTypeSell — sell / short.
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

// TimeInForceType — order expiry / queue behavior. The PostOnly variant
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

// PositionIdx — Bybit V5 uses a numeric position index that matters only
// in hedge mode. In one-way mode it is always 0; in hedge mode 1 = long
// position, 2 = short position.
type PositionIdx int

const (
	// PositionIdxOneWay — one-way mode (the only mode available unless
	// SetPositionMode is changed).
	PositionIdxOneWay PositionIdx = 0
	// PositionIdxHedgeBuy — hedge-mode long position.
	PositionIdxHedgeBuy PositionIdx = 1
	// PositionIdxHedgeSell — hedge-mode short position.
	PositionIdxHedgeSell PositionIdx = 2
)

// PositionMode — account-wide position mode. Wire codes are integers per
// Bybit V5; the typed constants make call sites self-documenting.
type PositionMode int

const (
	// PositionModeOneWay — single position per symbol; long/short is
	// inferred from the sign of the size.
	PositionModeOneWay PositionMode = 0
	// PositionModeHedge — separate long/short positions per symbol.
	PositionModeHedge PositionMode = 3
)

// OrderStatus is a subset of the Bybit V5 order states. The SDK does NOT
// expose a closed enum of every state (the Bybit list is long and
// occasionally extended); strings outside this list are returned verbatim
// in OrderInfo.Status. Callers that need finer status discrimination
// should read the raw status string.
type OrderStatus string

const (
	// OrderStatusNew — accepted by the matcher, untriggered (alias: Created/New).
	OrderStatusNew OrderStatus = "New"
	// OrderStatusPartiallyFilled — partially filled, remainder live.
	OrderStatusPartiallyFilled OrderStatus = "PartiallyFilled"
	// OrderStatusFilled — fully filled.
	OrderStatusFilled OrderStatus = "Filled"
	// OrderStatusCancelled — cancelled.
	OrderStatusCancelled OrderStatus = "Cancelled"
	// OrderStatusRejected — rejected by the exchange before reaching the book.
	OrderStatusRejected OrderStatus = "Rejected"
	// OrderStatusUntriggered — conditional/trigger order waiting for trigger.
	OrderStatusUntriggered OrderStatus = "Untriggered"
	// OrderStatusTriggered — conditional order triggered, awaiting fill.
	OrderStatusTriggered OrderStatus = "Triggered"
)
