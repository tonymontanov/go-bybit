/*
FILE: types/enums.go

DESCRIPTION:
Closed enums (typed strings / typed ints) shared across every Bybit V5
profile. Values match the wire format Bybit accepts and returns —
keep them exact, or the exchange rejects with retCode 10001.

Profile packages re-export these via type aliases (see linears/types,
spot/types). Profile-specific values that the exchange ALSO encodes as
the same enum type (e.g. linears trigger-only OrderStatus values, swap
OrderType=optimal_limit_ioc on OKX) are added by the profile package as
constants of the aliased type — no separate type is introduced.
*/

package types

// Category — Bybit V5 product family identifier sent on every REST
// call (`category` query/body parameter). Each profile pins one value
// in its REST/WS calls, but the constant set is exposed here so that
// rate-limiter / observer code can address all categories uniformly
// without depending on a profile package.
type Category string

const (
	// CategoryLinear — USDT/USDC perpetual + USDC futures.
	CategoryLinear Category = "linear"
	// CategoryInverse — coin-margined contracts.
	CategoryInverse Category = "inverse"
	// CategorySpot — spot.
	CategorySpot Category = "spot"
	// CategoryOption — options.
	CategoryOption Category = "option"
)

// SideType — order direction on the exchange wire. Bybit V5 uses the
// same string values across all categories.
type SideType string

const (
	// SideTypeBuy — buy / long.
	SideTypeBuy SideType = "Buy"
	// SideTypeSell — sell / short.
	SideTypeSell SideType = "Sell"
)

// OrderType — order execution model on the exchange wire.
//
// Both linears and spot accept the same two base values (`Limit`,
// `Market`); spot does NOT support trigger-only types like
// `StopLimit` or `StopMarket` (those are encoded via the
// `triggerPrice` parameter on a regular Limit/Market order). Profile
// packages MAY add own constants of OrderType for category-specific
// extensions (none today).
type OrderType string

const (
	// OrderTypeLimit — limit order.
	OrderTypeLimit OrderType = "Limit"
	// OrderTypeMarket — market order.
	OrderTypeMarket OrderType = "Market"
)

// TimeInForceType — order expiry / queue behaviour. The PostOnly
// variant is recognised by Bybit V5 only when paired with
// orderType=Limit; the SDK applies that mapping in trading.go of each
// profile.
type TimeInForceType string

const (
	// TimeInForceGTC — Good Till Cancel (default for Limit).
	TimeInForceGTC TimeInForceType = "GTC"
	// TimeInForceIOC — Immediate or Cancel (default for Market).
	TimeInForceIOC TimeInForceType = "IOC"
	// TimeInForceFOK — Fill or Kill.
	TimeInForceFOK TimeInForceType = "FOK"
	// TimeInForcePostOnly — post-only (rejected if it would cross the
	// book). Maps to orderType=Limit + timeInForce=PostOnly on the wire.
	TimeInForcePostOnly TimeInForceType = "PostOnly"
)

// OrderStatus — base order states that every Bybit V5 category emits
// on the same string values. Profiles MAY add own constants of the
// aliased type for category-specific states (linears adds
// `Untriggered` / `Triggered` for trigger orders).
//
// Bybit occasionally extends the catalogue; values outside the well-
// known set are returned verbatim in OrderInfo.Status — callers that
// need finer status discrimination should read the raw status string.
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
)

// AccountType — Bybit V5 wallet partition selector for
// /v5/account/wallet-balance and the wallet WebSocket topic. All four
// values are declared here so the rate-limiter / observer / wallet
// reconciler can address all wallets uniformly. Each profile re-
// exports the subset relevant to it:
//
//   - linears: UNIFIED + CONTRACT
//   - spot:    UNIFIED + SPOT
//
// Sending an inapplicable accountType yields retCode 10001 on the wire.
type AccountType string

const (
	// AccountTypeUnified — Unified Trading Account (UTA), one equity pool.
	AccountTypeUnified AccountType = "UNIFIED"
	// AccountTypeContract — Classic Account derivatives wallet.
	AccountTypeContract AccountType = "CONTRACT"
	// AccountTypeSpot — Classic Account spot wallet.
	AccountTypeSpot AccountType = "SPOT"
)
