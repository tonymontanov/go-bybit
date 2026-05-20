/*
FILE: linears/types/order-book-snapshot.go

DESCRIPTION:
Order book snapshot for the Bybit V5 linear category. Returned by
MarketData.GetOrderBook and pushed by the WebSocket "orderbook" topic
with type=="snapshot".

Bybit synchronization model:
  - Each push (snapshot or delta) carries an "u" sequence number; deltas
    are applied in order and a missing u indicates a gap that requires
    a fresh snapshot.
  - For snapshots Bybit also delivers "seq", a monotonic per-symbol
    counter useful for cross-checking. The SDK preserves both.
  - Bybit does NOT publish CRC32 checksums (unlike OKX) — gap detection
    is purely sequence-based.

FIELDS:
  - Symbol     — e.g. "BTCUSDT".
  - Bids       — buy levels, sorted descending by price.
  - Asks       — sell levels, sorted ascending by price.
  - UpdateID   — Bybit "u" — book sequence number.
  - SeqID      — Bybit "seq" — symbol-wide sequence (snapshots only).
  - TsMs       — Bybit publish timestamp (ms).
*/

package types

// OrderBookSnapshot — order book snapshot for a single symbol.
type OrderBookSnapshot struct {
	Symbol   string
	Bids     []OrderBookLevel
	Asks     []OrderBookLevel
	UpdateID int64
	SeqID    int64
	TsMs     int64
}
