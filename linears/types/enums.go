/*
FILE: linears/types/enums.go

DESCRIPTION:
Closed enums (typed strings / typed ints) used by the Bybit V5 linear
profile. Most values are PROTOCOL-COMMON and re-exported from the
neutral `github.com/tonymontanov/go-bybit/v2/types` package via Go type
aliases (`type X = commontypes.X`) — this preserves type identity and
keeps `linears/types.SideTypeBuy` byte-for-byte compatible with
`spot/types.SideTypeBuy` at the type-system level.

Linears-only enums declared HERE:
  - PositionIdx (one-way / hedge-buy / hedge-sell)  — derivatives concept.
  - PositionMode (one-way / hedge)                  — derivatives concept.
  - OrderStatus extra constants: Untriggered / Triggered. Both are
    declared as constants of the aliased OrderStatus type so they are
    interchangeable with the common values.
*/

package types

import commontypes "github.com/tonymontanov/go-bybit/v2/types"

// Category is the Bybit V5 product family identifier sent on every
// REST call. The linears profile pins `linear`; the constant set is
// re-exported so call sites can stay package-local.
type Category = commontypes.Category

const (
	// CategoryLinear — USDT/USDC perpetual + USDC futures.
	CategoryLinear = commontypes.CategoryLinear
	// CategoryInverse — coin-margined contracts.
	CategoryInverse = commontypes.CategoryInverse
	// CategorySpot — spot.
	CategorySpot = commontypes.CategorySpot
	// CategoryOption — options.
	CategoryOption = commontypes.CategoryOption
)

// SideType — order direction on the exchange wire.
type SideType = commontypes.SideType

const (
	// SideTypeBuy — buy / long.
	SideTypeBuy = commontypes.SideTypeBuy
	// SideTypeSell — sell / short.
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

// PositionIdx — Bybit V5 uses a numeric position index that matters
// only in hedge mode. In one-way mode it is always 0; in hedge mode
// 1 = long position, 2 = short position.
//
// LINEARS-ONLY: the spot category has no positions and therefore no
// position index — the type lives only in this package.
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

// PositionMode — account-wide position mode. Wire codes are integers
// per Bybit V5; the typed constants make call sites self-documenting.
//
// LINEARS-ONLY.
type PositionMode int

const (
	// PositionModeOneWay — single position per symbol; long/short is
	// inferred from the sign of the size.
	PositionModeOneWay PositionMode = 0
	// PositionModeHedge — separate long/short positions per symbol.
	PositionModeHedge PositionMode = 3
)

// OrderStatus is the Bybit V5 order state. The base catalogue lives
// in commontypes; the linears profile additionally accepts the
// trigger-order specific values (Untriggered, Triggered) — declared
// here as constants of the aliased type so they interoperate with the
// common values.
//
// The SDK does NOT expose a closed enum of every state (the Bybit
// list is long and occasionally extended); strings outside this list
// are returned verbatim in OrderInfo.Status. Callers that need finer
// status discrimination should read the raw status string.
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
	// OrderStatusUntriggered — conditional/trigger order waiting for
	// trigger. LINEARS-ONLY.
	OrderStatusUntriggered OrderStatus = "Untriggered"
	// OrderStatusTriggered — conditional order triggered, awaiting
	// fill. LINEARS-ONLY.
	OrderStatusTriggered OrderStatus = "Triggered"
)
