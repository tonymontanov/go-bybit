/*
FILE: rate-limit-event.go

DESCRIPTION:
Public RateLimitEvent type that the SDK delivers to subscribers via
Config.RateLimitEventObserver. The observer pattern is identical to
go-okx: the SDK writes once per completed REST call, the desk
rate-limiter consumes events to update its model.

WHY HEADERS ARE POPULATED HERE (UNLIKE go-okx):
Bybit V5 returns rate-limit headers on every signed REST response
(X-Bapi-Limit / X-Bapi-Limit-Status / X-Bapi-Limit-Reset-Timestamp /
Retry-After). They are forwarded as-is so an external rate-limiter at
the desk level can reconcile its model with the live remaining budget.

THE THREE METADATA AXES:

  1. OrderCount: 1 for single, len(orders) for batch endpoints
     (/v5/order/create-batch, /v5/order/cancel-batch,
      /v5/order/amend-batch). Bybit accounts for batches in
     orders, not requests.
  2. Symbols:    sorted unique list of symbols affected. Bybit V5
     trading limits on derivatives are per (UID + Symbol); the
     subscriber must debit usage to the symbols actually consumed,
     not aggregate by endpoint.
  3. Category:   "place" | "amend" | "cancel" | "query" | "market" | "".
     Used by the rate-limiter to model the sub-account-level
     NEW+AMEND budget separately from cancellations and queries.
*/

package bybit

// RateLimitCategory classifies a REST call from the rate-limit model
// perspective. Used by external rate-limiters to distribute usage across
// different limit planes.
type RateLimitCategory string

const (
	// RateLimitCategoryPlace — order creation. Endpoints:
	// /v5/order/create, /v5/order/create-batch.
	RateLimitCategoryPlace RateLimitCategory = "place"

	// RateLimitCategoryAmend — order modification. Endpoints:
	// /v5/order/amend, /v5/order/amend-batch.
	RateLimitCategoryAmend RateLimitCategory = "amend"

	// RateLimitCategoryCancel — order cancellation. Endpoints:
	// /v5/order/cancel, /v5/order/cancel-batch, /v5/order/cancel-all.
	RateLimitCategoryCancel RateLimitCategory = "cancel"

	// RateLimitCategoryQuery — private GET / non-trading POST. Endpoints:
	// /v5/order/realtime, /v5/position/list, /v5/account/wallet-balance,
	// /v5/account/info, /v5/position/set-leverage, etc.
	RateLimitCategoryQuery RateLimitCategory = "query"

	// RateLimitCategoryMarketData — public GET (per-IP limits). Endpoints:
	// /v5/market/*, /v5/market/instruments-info, /v5/market/orderbook, ...
	RateLimitCategoryMarketData RateLimitCategory = "market"

	// RateLimitCategoryUnknown — fallback for requests not covered by any
	// explicit category. Treat as Query for safety in subscribers.
	RateLimitCategoryUnknown RateLimitCategory = ""
)

// String returns the string representation.
func (c RateLimitCategory) String() string { return string(c) }

// RateLimitEvent is the structured event delivered to
// Config.RateLimitEventObserver. The SDK emits exactly one event per
// completed REST call (whether successful or rejected at the application
// layer). Network-only failures (timeout before any HTTP response) do
// NOT trigger the observer.
type RateLimitEvent struct {
	// Endpoint — request path in canonical form (e.g. "/v5/order/create").
	// Never empty.
	Endpoint string

	// Method — HTTP method in upper-case ("GET", "POST", ...).
	Method string

	// Headers — selected rate-limit headers from the response. Populated
	// from the X-Bapi-Limit-* family when present:
	//
	//   X-Bapi-Limit                 (max requests in current window)
	//   X-Bapi-Limit-Status          (current usage counter)
	//   X-Bapi-Limit-Reset-Timestamp (window reset timestamp, ms)
	//   X-Bapi-Recv-Window-Status    (set on recv-window related rejects)
	//   Retry-After                  (set on 429 responses)
	//
	// May be empty for public endpoints that do not advertise per-UID
	// limits. Always non-nil.
	Headers map[string]string

	// OrderCount — number of orders this request creates / amends /
	// cancels:
	//   - 1 for /v5/order/{create,amend,cancel};
	//   - len(orders) for batch endpoints;
	//   - 0 for non-trading queries.
	OrderCount int

	// Symbols — sorted unique list of symbols affected. Empty for
	// account-level queries (wallet balance, fee rates, etc.).
	Symbols []string

	// Category — see RateLimitCategory.
	Category RateLimitCategory
}
