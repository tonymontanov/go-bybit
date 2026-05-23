/*
Package types contains the domain structs used by the linears profile of
the Bybit V5 SDK.

LAYERING (since v2.1.0):

	github.com/tonymontanov/go-bybit/v2/types/        ← layer 1 (protocol-common)
	    │
	    └─── github.com/tonymontanov/go-bybit/v2/linears/types/   ← layer 2 (this package)
	            └─── alias re-exports of layer 1 +
	                 linears-only types

Most types here are Go type aliases (`type X = commontypes.X`) on the
neutral `github.com/tonymontanov/go-bybit/v2/types` package — the wire
format of those types is identical across every Bybit V5 category, so
duplicating the definitions would be parallel copy-paste. Aliases
preserve type identity at the language level: existing code that
references `linears/types.OrderBookLevel{...}` or
`linears/types.SideTypeBuy` continues to compile unchanged after the
refactor.

LINEARS-ONLY TYPES (declared in this package):
  - `PositionIdx`, `PositionMode`              — derivatives concepts.
  - `OrderStatus` constants `Untriggered` /
    `Triggered`                                — trigger-order states.
  - `SymbolInfo`                               — leverage / qtyStep filter.
  - `CreateOrderRequest`, `ModifyOrderRequest` — PositionIdx /
                                                 ReduceOnly /
                                                 CloseOnTrigger fields.
  - `OrderInfo`, `ExecutionInfo`               — mirror request fields.
  - `TickerUpdate`                             — MarkPrice / FundingRate /
                                                 OpenInterest.
  - `PositionInfo`                             — derivatives positions.
  - `BatchOrderResult`                         — embeds linears OrderInfo.

NUMERIC POLICY:
All numeric fields are typed with shopspring/decimal where precision
is critical (prices, quantities, leverage, fees) and int64 where ms
timestamps are stored. The SDK never converts a Bybit numeric string
into a float64 internally — the desk adapter does that conversion at
the boundary.
*/
package types
