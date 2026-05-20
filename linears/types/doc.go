/*
Package types contains the domain structs used by the linears profile of
the Bybit V5 SDK. They are intentionally separated from the linears
package so that:

  - external code (e.g. a desk-side adapter) can depend on the types
    without pulling in the REST/WS clients;
  - the same struct definitions can be shared between trading methods and
    stream handlers without import cycles.

Naming is kept symmetric with the SDK's overall public surface — e.g.
CreateOrderRequest / OrderInfo / PositionInfo mirror the desk's typing
to make wrapping by the desk-side adapter trivial.

All numeric fields are typed with shopspring/decimal where precision is
critical (prices, quantities, leverage, fees) and int64 where ms
timestamps are stored. The SDK never converts a Bybit numeric string
into a float64 internally — the desk adapter does that conversion at the
boundary.
*/
package types
