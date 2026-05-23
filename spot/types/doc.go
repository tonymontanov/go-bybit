/*
Package types contains the domain structs used by the spot profile of
the Bybit V5 SDK.

LAYERING (since v2.1.0):

	github.com/tonymontanov/go-bybit/v2/types/        ← layer 1 (protocol-common)
	    │
	    └─── github.com/tonymontanov/go-bybit/v2/spot/types/      ← layer 2 (this package)
	            └─── alias re-exports of layer 1 +
	                 spot-only types

Most types here are Go type aliases (`type X = commontypes.X`) on the
neutral `github.com/tonymontanov/go-bybit/v2/types` package — the wire
format of those types is identical across every Bybit V5 category, so
duplicating the definitions would be parallel copy-paste with the
linears profile. Aliases preserve type identity at the language level:
existing code that references `spot/types.OrderBookLevel{...}` or
`spot/types.SideTypeBuy` continues to compile unchanged after the
refactor.

NO IMPORT CYCLE BETWEEN PROFILES:
The spot profile does NOT import linears/types and vice versa — both
profile packages depend only on the neutral commontypes package. This
keeps spot/derivatives orthogonal: using a derivatives method from the
spot connector (or vice versa) is impossible by construction.

SPOT-ONLY TYPES (declared in this package):
  - `MarginTrading`                            — instrument-level margin flag.
  - `MarketUnit`                               — Market BUY interpretation
                                                 (baseCoin / quoteCoin).
  - `SymbolInfo`                               — marginTrading / basePrecision /
                                                 minOrderAmt fields.
  - `CreateOrderRequest`, `ModifyOrderRequest` — MarketUnit / IsLeverage fields.
  - `OrderInfo`, `ExecutionInfo`               — mirror request fields.
  - `TickerUpdate`                             — UsdIndexPrice (no MarkPrice /
                                                 FundingRate / OpenInterest).
  - `BatchOrderResult`                         — embeds spot OrderInfo.

Spot has no positions, so `PositionIdx` / `PositionMode` from
linears/types are intentionally not present here.

NUMERIC POLICY:
All numeric fields are typed with shopspring/decimal where precision
is critical and int64 where ms timestamps are stored. The SDK never
converts a Bybit numeric string into a float64 internally — the desk
adapter does that conversion at the boundary.
*/
package types
