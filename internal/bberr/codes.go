/*
FILE: internal/bberr/codes.go

DESCRIPTION:
Mapping from raw transport-level signals (HTTP status, Bybit V5 retCode) to
ErrorKind. Centralised here so the rest of the SDK never sprinkles
"if code == ..." chains.

SOURCES:
  - HTTP status mapping is the standard 4xx/5xx convention, with 401/403 as
    Auth and 429 as RateLimit.
  - retCode tables are derived from the public Bybit V5 documentation
    (see https://bybit-exchange.github.io/docs/v5/error). The list is NOT
    exhaustive — only codes the SDK can usefully classify. Anything not
    explicitly listed falls back to ErrorKindExchange (so the caller still
    gets an Exchange-class error, just without the SDK pre-classifying it).

UPDATE PROCEDURE:
When Bybit publishes a new error code that the SDK should react to:
  1. Add a `case` to MapBybitCode below.
  2. Add a covering test in error_test.go.
  3. If it changes Kind for an already-listed code, bump the SDK minor
     version — this is a behaviour change for callers of IsRateLimit etc.
*/

package bberr

// MapHTTPStatus maps an HTTP status code to an ErrorKind.
//
// 2xx is a success and is not expected to be passed here.
// 401/403 → Auth (key/IP/permission).
// 429     → RateLimit.
// 4xx     → InvalidRequest (the SDK or caller built a bad request).
// 5xx     → Network (transient at the network/exchange edge — retryable).
// other   → Unknown.
func MapHTTPStatus(status int) ErrorKind {
	switch {
	case status == 401, status == 403:
		return ErrorKindAuth
	case status == 429:
		return ErrorKindRateLimit
	case status >= 400 && status < 500:
		return ErrorKindInvalidRequest
	case status >= 500 && status < 600:
		return ErrorKindNetwork
	default:
		return ErrorKindUnknown
	}
}

// MapBybitCode maps a Bybit V5 retCode to an ErrorKind. msg is currently
// unused but kept in the signature so future heuristics (e.g. parsing
// "Too many visits" out of retMsg) do not break callers.
//
// The function operates on the string form of retCode (Bybit returns it
// as an integer in JSON; the REST client converts it to string before
// calling this).
//
// Codes are grouped by family:
//
//   - 10xxx — generic API/auth/rate-limit errors (apply to all categories).
//   - 110xxx — derivatives (linear/inverse) trading errors.
//   - 130xxx — risk-limit / cancel-all family (linear/inverse).
//   - 170xxx — spot trading errors.
//
// Anything outside the explicitly listed codes maps to ErrorKindExchange:
// the SDK saw a non-zero retCode, but does not pre-classify it.
func MapBybitCode(code, _ string) ErrorKind {
	switch code {
	// ----- 10xxx generic auth / signature / rate limit -----
	case "10001":
		// "params error" / "request parameter error"
		return ErrorKindInvalidRequest
	case "10002":
		// "request not authorized" — recv-window expired / clock skew
		return ErrorKindAuth
	case "10003":
		// "API key invalid"
		return ErrorKindAuth
	case "10004":
		// "sign auth fail" — wrong signature
		return ErrorKindAuth
	case "10005":
		// "permission denied"
		return ErrorKindAuth
	case "10006":
		// "too many visits in the window" — IP-level rate limit
		return ErrorKindRateLimit
	case "10007":
		// "user authorization failed"
		return ErrorKindAuth
	case "10009":
		// "IP banned" / "IP not in whitelist"
		return ErrorKindAuth
	case "10010":
		// "unmatched IP"
		return ErrorKindAuth
	case "10016":
		// "server error" — typically 5xx; retryable
		return ErrorKindNetwork
	case "10017":
		// "route not found" — SDK targeted a path Bybit does not expose;
		// caller built an invalid request.
		return ErrorKindInvalidRequest
	case "10018":
		// "exceeded UID rate limit" — endpoint-level rate limit
		return ErrorKindRateLimit
	case "10029":
		// "request frequency exceeds the limit" (system-wide).
		return ErrorKindRateLimit

	// ----- 110xxx derivatives trading -----
	case "110001":
		// "order does not exist"
		return ErrorKindInvalidRequest
	case "110003":
		// "order price out of permissible range"
		return ErrorKindInvalidRequest
	case "110004":
		// "wallet balance insufficient" — distinct from 110007
		return ErrorKindExchange
	case "110007":
		// "insufficient available balance"
		return ErrorKindExchange
	case "110008":
		// "order has been finished or cancelled"
		return ErrorKindInvalidRequest
	case "110009":
		// "too many active stop orders for the symbol"
		return ErrorKindInvalidRequest
	case "110012":
		// "insufficient available balance for order cost"
		return ErrorKindExchange
	case "110017":
		// "qty does not meet min/max" — caller built an invalid request
		return ErrorKindInvalidRequest
	case "110020":
		// "too many active orders for the symbol"
		return ErrorKindInvalidRequest
	case "110025":
		// "position-mode mismatch" — cross/isolated or hedge/one-way
		// switch attempted while positions are open.
		return ErrorKindInvalidRequest
	case "110043":
		// "set leverage not modified" / position-mode mismatch
		return ErrorKindInvalidRequest
	case "110052":
		// "leverage out of allowed range for the symbol"
		return ErrorKindInvalidRequest

	// ----- 130xxx cancel-all / risk-limit family -----
	case "130150":
		// "cancel-all rate limit"
		return ErrorKindRateLimit

	// ----- 170xxx spot trading -----
	case "170131":
		// "balance insufficient"
		return ErrorKindExchange
	case "170135":
		// "qty rounding"
		return ErrorKindInvalidRequest
	case "170140":
		// "order amount too small"
		return ErrorKindInvalidRequest

	default:
		return ErrorKindExchange
	}
}
