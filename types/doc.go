/*
Package types holds the protocol-common domain structs for the Bybit V5
SDK. Every type exported here is byte-identical on the wire across all V5
profiles (linears, spot, and any future inverse/option) — putting them in
one neutral package eliminates the parallel copy-paste between
`linears/types/` and `spot/types/` while keeping the two profile packages
completely decoupled (neither imports the other).

LAYERING:

	github.com/tonymontanov/go-bybit/v2/types/        ← layer 1 (this package)
	    │                                               protocol-common types
	    │
	    ├─── github.com/tonymontanov/go-bybit/v2/linears/types/   ← layer 2 (profile)
	    │       └─── alias re-exports + linears-only types
	    │            (PositionIdx, PositionMode, OrderStatus
	    │             trigger-states, profile-specific
	    │             SymbolInfo / OrderInfo / Create/Modify /
	    │             ExecutionInfo / Ticker / PositionInfo /
	    │             BatchOrderResult)
	    │
	    └─── github.com/tonymontanov/go-bybit/v2/spot/types/      ← layer 2 (profile)
	            └─── alias re-exports + spot-only types
	                 (MarginTrading, profile-specific
	                  SymbolInfo / OrderInfo / Create/Modify /
	                  ExecutionInfo / Ticker / BatchOrderResult)

WHY ALIASES, NOT NEW NAMED TYPES IN PROFILES:
A Go type alias (`type X = commontypes.X`) is the SAME type at the
language level — identity is preserved, no value conversions are needed
at call sites, and constants of the aliased type can be re-exported
verbatim with `const Foo = commontypes.Foo`. This means the v2.0.x →
v2.1.0 transition is non-breaking: existing code that constructs
`linears/types.OrderBookLevel{...}` or compares to
`spot/types.SideTypeBuy` continues to compile without changes.

WHAT BELONGS HERE:
Types whose JSON wire format is the same for every Bybit V5 category
that uses them. Concretely:

  - Closed enums (Side / OrderType / TimeInForce / Category /
    AccountType / OrderStatus base set) — values are wire strings used
    as-is by `/v5/order/*`, `/v5/account/wallet-balance`, etc.
  - Public market-data shapes (`OrderBookLevel`, `OrderBookSnapshot`,
    `Candle`/`Candles`, `Timeframe`, `TradeUpdate`, `KlineUpdate`).
  - Profile-agnostic request shapes (`CancelOrderRequest`).
  - Wallet shapes (`Balance`, `CoinBalance`).

WHAT DOES NOT BELONG HERE:
Anything where the field set legitimately differs across profiles, even
if the type name is the same. Concretely:

  - `SymbolInfo` (linears: leverage / qtyStep filter; spot: marginTrading /
    basePrecision / minOrderAmt).
  - `CreateOrderRequest` / `ModifyOrderRequest` (linears: PositionIdx /
    ReduceOnly / CloseOnTrigger; spot: MarketUnit / IsLeverage).
  - `OrderInfo` / `ExecutionInfo` (mirror the request differences).
  - `TickerUpdate` (linears: MarkPrice / FundingRate / OpenInterest;
    spot: UsdIndexPrice).
  - `BatchOrderResult` (embeds profile-specific OrderInfo).
  - `PositionInfo` (linears-only).

Forcing those into a unified shape would either require empty/unused
fields (noise at every consumer) or a tag-discriminated union (runtime
overhead and lost type safety). The alias-based split keeps both profile
packages typed precisely while reusing 100% of what is genuinely common.
*/
package types
