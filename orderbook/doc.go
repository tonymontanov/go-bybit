/*
FILE: orderbook/doc.go

DESCRIPTION:
Package orderbook implements a local L2 order book engine for Bybit V5.
The source of truth is the WebSocket orderbook.{depth}.{symbol} topic
(snapshot + delta). Bybit publishes:

  - "u" — a monotonic per-symbol update id; deltas must increase by 1.
  - "seq" — a service-side sequence (informational).

UNLIKE OKX, Bybit does NOT publish a CRC32 checksum, so the only gap
signal is the "u" sequence. When a gap is detected the engine returns
GapSequence and stops applying further deltas until ApplySnapshot or
MarkResynced is called by the dispatcher (M3 wires this to the
StreamClient resubscription path).

ENTRY POINTS:
  - NewEngine(symbol, maxDepth) *Engine
  - (*Engine).ApplySnapshot(Snapshot) ApplyResult
  - (*Engine).ApplyDelta(Delta) ApplyResult
  - (*Engine).TopLevels(n)            ([]OrderBookLevel, []OrderBookLevel)
  - (*Engine).BestBidAsk()            (px,sz,px,sz)
  - (*Engine).LastUpdateID()          int64
  - (*Engine).IsDirty()               bool
  - (*Engine).MarkResynced(u, seq, ts) — clears dirty after a manual resync.

THREAD SAFETY:
The engine is safe for concurrent ApplyDelta + readers. Readers acquire a
read lock and copy out the requested slice; writers use a single write
lock that covers the entire delta application.

PERFORMANCE:
Levels are stored in two sorted slices with O(log n) binary-search
insertion / deletion. For depth ≤ 200 (Bybit's typical "fast" topic) one
delta application costs a few hundred ns on modern hardware, dominated by
the decimal.Decimal compares.
*/
package orderbook
