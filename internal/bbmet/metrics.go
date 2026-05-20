/*
FILE: internal/bbmet/metrics.go

DESCRIPTION:
Minimalistic metrics interface used by the SDK to expose internal counters
without taking a hard dependency on any specific metrics library
(prometheus, opencensus, opentelemetry).

The SDK only emits monotonic counters. Histograms/gauges are out of scope —
embedders that want timing distributions should wrap user-facing calls
themselves.

NAMES used by the SDK (stable contract — see README):
  bybit_ws_messages_received_total
  bybit_ws_messages_dropped_total
  bybit_ws_reconnects_total
  bybit_ws_subscriptions_total
  bybit_ws_ping_failed_total

The factory pattern (CounterFactory) lets embedders attach labels (e.g.
exchange="bybit", profile="linears") at construction time without the SDK
knowing about labels at all.
*/

package bbmet

// Counter is a monotonically increasing metric.
type Counter interface {
	// Inc increments the counter by 1. Must be safe for concurrent use.
	Inc()
	// Add increments the counter by delta. delta must be non-negative —
	// negative values are silently dropped (counters are monotonic by
	// definition).
	Add(delta uint64)
}

// CounterFactory creates named counters. Implementations may apply common
// label sets, prefix the name, etc. Must be safe for concurrent use.
type CounterFactory interface {
	// Counter returns a Counter for the given name. Must be idempotent: the
	// same name always returns the same underlying counter. Implementations
	// are free to cache.
	Counter(name string) Counter
}

// noopCounter discards all increments.
type noopCounter struct{}

func (noopCounter) Inc()       {}
func (noopCounter) Add(uint64) {}

// noopFactory always returns the same noopCounter.
type noopFactory struct{}

func (noopFactory) Counter(string) Counter { return noopCounter{} }

// Noop returns a CounterFactory that discards every increment. Default
// when Config.Metrics is nil.
func Noop() CounterFactory { return noopFactory{} }
