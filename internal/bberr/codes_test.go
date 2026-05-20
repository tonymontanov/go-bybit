/*
FILE: internal/bberr/codes_test.go

DESCRIPTION:
Table-driven tests for MapHTTPStatus and MapBybitCode. The list intentionally
covers every code we route to a non-default Kind (Auth/RateLimit/InvalidRequest/
Network), plus a couple of "fallback to Exchange" sanity checks.

If you add or change a code in codes.go, add the corresponding row here.
*/

package bberr

import "testing"

func TestMapHTTPStatus(t *testing.T) {
	type tc struct {
		name   string
		status int
		want   ErrorKind
	}
	var tests []tc = []tc{
		{"unauthorized", 401, ErrorKindAuth},
		{"forbidden", 403, ErrorKindAuth},
		{"too-many-requests", 429, ErrorKindRateLimit},
		{"bad-request", 400, ErrorKindInvalidRequest},
		{"not-found", 404, ErrorKindInvalidRequest},
		{"internal-server-error", 500, ErrorKindNetwork},
		{"bad-gateway", 502, ErrorKindNetwork},
		{"service-unavailable", 503, ErrorKindNetwork},
		{"unknown", 100, ErrorKindUnknown},
	}
	var i int
	for i = 0; i < len(tests); i++ {
		var c tc = tests[i]
		t.Run(c.name, func(t *testing.T) {
			var got ErrorKind = MapHTTPStatus(c.status)
			if got != c.want {
				t.Fatalf("MapHTTPStatus(%d): got %v, want %v", c.status, got, c.want)
			}
		})
	}
}

func TestMapBybitCode(t *testing.T) {
	type tc struct {
		name string
		code string
		want ErrorKind
	}
	var tests []tc = []tc{
		// auth
		{"recv-window-expired", "10002", ErrorKindAuth},
		{"invalid-api-key", "10003", ErrorKindAuth},
		{"sign-auth-fail", "10004", ErrorKindAuth},
		{"permission-denied", "10005", ErrorKindAuth},
		{"user-auth-failed", "10007", ErrorKindAuth},
		{"ip-banned", "10009", ErrorKindAuth},
		{"unmatched-ip", "10010", ErrorKindAuth},
		// rate limit
		{"ip-rate-limit", "10006", ErrorKindRateLimit},
		{"uid-rate-limit", "10018", ErrorKindRateLimit},
		{"system-rate-limit", "10029", ErrorKindRateLimit},
		{"cancel-all-rate-limit", "130150", ErrorKindRateLimit},
		// invalid request
		{"params-error", "10001", ErrorKindInvalidRequest},
		{"route-not-found", "10017", ErrorKindInvalidRequest},
		{"order-does-not-exist", "110001", ErrorKindInvalidRequest},
		{"order-price-out-of-range", "110003", ErrorKindInvalidRequest},
		{"order-already-finished", "110008", ErrorKindInvalidRequest},
		{"too-many-stop-orders", "110009", ErrorKindInvalidRequest},
		{"qty-out-of-range", "110017", ErrorKindInvalidRequest},
		{"too-many-active-orders", "110020", ErrorKindInvalidRequest},
		{"position-mode-mismatch", "110025", ErrorKindInvalidRequest},
		{"set-leverage-no-change", "110043", ErrorKindInvalidRequest},
		{"leverage-out-of-range", "110052", ErrorKindInvalidRequest},
		{"qty-rounding-spot", "170135", ErrorKindInvalidRequest},
		{"order-amount-too-small-spot", "170140", ErrorKindInvalidRequest},
		// network (transient)
		{"server-error", "10016", ErrorKindNetwork},
		// exchange
		{"wallet-balance-insufficient", "110004", ErrorKindExchange},
		{"insufficient-balance-derivatives", "110007", ErrorKindExchange},
		{"insufficient-balance-for-cost", "110012", ErrorKindExchange},
		{"insufficient-balance-spot", "170131", ErrorKindExchange},
		// fallback
		{"unknown-falls-back-to-exchange", "999999", ErrorKindExchange},
	}
	var i int
	for i = 0; i < len(tests); i++ {
		var c tc = tests[i]
		t.Run(c.name, func(t *testing.T) {
			var got ErrorKind = MapBybitCode(c.code, "")
			if got != c.want {
				t.Fatalf("MapBybitCode(%q): got %v, want %v", c.code, got, c.want)
			}
		})
	}
}

func TestErrorIsKindAndUnwrap(t *testing.T) {
	var inner error = New(ErrorKindNetwork, "", "conn reset", nil)
	var wrapped *Error = New(ErrorKindNetwork, "", "rest: read body", inner)

	if !IsNetwork(wrapped) {
		t.Fatalf("IsNetwork should be true for wrapped network error")
	}
	if IsAuth(wrapped) {
		t.Fatalf("IsAuth should be false for network error")
	}
	if wrapped.Unwrap() != inner {
		t.Fatalf("Unwrap should return inner cause")
	}
}
