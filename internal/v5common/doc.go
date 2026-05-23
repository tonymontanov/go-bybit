/*
Package v5common holds tiny utilities shared across all V5 profile
sub-packages of the Bybit SDK (linears, spot, ...). It is intentionally
"internal" — consumers must not import it directly; profile packages
(`linears/`, `spot/`) re-export the bits they need.

Scope is deliberately narrow:

  - Numeric parsing helpers (`Dec`, `Ms`) — Bybit V5 wire format ships
    every numeric as a JSON string and the parsing semantics ("default
    to zero on empty / parse error") are identical for spot and linears.
  - `NormalizeRejectReason` — masks Bybit's "EC_NoError" sentinel that
    every non-rejected order carries. Shared because the order schema
    is identical across V5 categories.
  - `ClampOrderbookDepth` — Bybit's `/v5/market/orderbook?limit=` only
    accepts a fixed set of values. Linears allows {1, 50, 200, 500},
    Spot allows {1, 50, 200}. The helper takes the allowed set as an
    argument rather than hard-coding the linears table.
  - `ConvertOrderBookLevels` — generic over any struct that has Price /
    Size `decimal.Decimal` fields, via a tiny constructor closure.

Larger orchestration (page-iterators, instrument-cache builders,
stream-cores) intentionally lives in profile packages until we see at
least two consumers — premature shared abstraction has historically
made review harder, not easier, on this codebase.
*/
package v5common
