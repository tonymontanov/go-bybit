/*
Package types contains the domain structs used by the spot profile of
the Bybit V5 SDK.

DESIGN MIRROR: it is intentionally separated from the spot package so
that callers can `import bybitspottypes "github.com/tonymontanov/go-bybit/spot/types"`
without dragging the implementation. The structure parallels
`linears/types` but with spot-specific differences:

  - No `PositionIdx`, no `ReduceOnly` / `CloseOnTrigger` on order
    requests (spot orders cannot reduce or close a position).
  - `SymbolInfo` carries `MarginTrading` (string enum) instead of
    leverage filters.
  - Trade/ticker/kline/orderbook update payloads share the same
    on-the-wire shape as linear, so the structs are byte-identical
    to their `linears/types` counterparts. They are duplicated here
    rather than imported because Go interfaces over those shapes are
    the wrong abstraction (we don't want callers to write code that
    is generic across category — each category has different trading
    semantics that show up only at the call-site level).
*/
package types
