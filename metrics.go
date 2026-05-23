/*
FILE: metrics.go

DESCRIPTION:
Public re-export of the metrics interfaces defined in internal/bbmet.
The SDK emits only monotonic counters; histograms/gauges are out of
scope (embedders that want timing distributions should wrap user-facing
calls themselves).

STABLE COUNTER NAMES (the SDK contract — see README):

  bybit_ws_messages_received_total
  bybit_ws_messages_dropped_total
  bybit_ws_reconnects_total
  bybit_ws_subscriptions_total
  bybit_ws_ping_failed_total
  bybit_ws_auth_ok_total
  bybit_ws_auth_failed_total
*/

package bybit

import "github.com/tonymontanov/go-bybit/v2/internal/bbmet"

// Counter is a monotonically increasing metric.
type Counter = bbmet.Counter

// CounterFactory creates named counters. Implementations may attach common
// labels at construction time.
type CounterFactory = bbmet.CounterFactory

// NoopMetrics returns a CounterFactory that discards every increment.
func NoopMetrics() CounterFactory { return bbmet.Noop() }
