/*
FILE: errors.go

DESCRIPTION:
Public re-export of the SDK error type and category predicates from
internal/bberr. Importers work through the root package only:

	import bybit "github.com/tonymontanov/go-bybit"

	if bybit.IsRateLimit(err) { ... }

The aliases below preserve the typed identity (`type X = internal.X`),
so users can also do `errors.As(err, &bybit.Error{})`.
*/

package bybit

import "github.com/tonymontanov/go-bybit/internal/bberr"

// Error is the SDK error type (alias). All SDK methods return *Error
// (sometimes wrapped). errors.As / errors.Is work normally.
type Error = bberr.Error

// ErrorKind is the error category enum (alias).
type ErrorKind = bberr.ErrorKind

// Error categories. See internal/bberr for full semantics of each kind.
const (
	// ErrorKindUnknown — the SDK could not classify the failure.
	ErrorKindUnknown = bberr.ErrorKindUnknown
	// ErrorKindNetwork — transport-level failure (timeout, conn reset, ...).
	ErrorKindNetwork = bberr.ErrorKindNetwork
	// ErrorKindRateLimit — Bybit told us we hit a rate limit.
	ErrorKindRateLimit = bberr.ErrorKindRateLimit
	// ErrorKindAuth — credentials missing/invalid or signature rejected.
	ErrorKindAuth = bberr.ErrorKindAuth
	// ErrorKindInvalidRequest — malformed request, validation rejection.
	ErrorKindInvalidRequest = bberr.ErrorKindInvalidRequest
	// ErrorKindExchange — exchange rejected the request for business reasons.
	ErrorKindExchange = bberr.ErrorKindExchange
)

// NewError constructs an *Error. Mostly used by SDK internals; user code
// rarely needs this.
func NewError(kind ErrorKind, code, msg string, cause error) *Error {
	return bberr.New(kind, code, msg, cause)
}

// IsNetwork reports whether err is a network-class error.
func IsNetwork(err error) bool { return bberr.IsNetwork(err) }

// IsRateLimit reports whether err is a rate-limit error.
func IsRateLimit(err error) bool { return bberr.IsRateLimit(err) }

// IsAuth reports whether err is an auth/permission error.
func IsAuth(err error) bool { return bberr.IsAuth(err) }

// IsInvalidRequest reports whether err is a validation/build-time error.
func IsInvalidRequest(err error) bool { return bberr.IsInvalidRequest(err) }

// IsExchange reports whether err is an exchange-level rejection.
func IsExchange(err error) bool { return bberr.IsExchange(err) }

// MapBybitCode returns the SDK ErrorKind for a Bybit V5 retCode (string).
func MapBybitCode(code, msg string) ErrorKind { return bberr.MapBybitCode(code, msg) }

// MapHTTPStatus returns the SDK ErrorKind for an HTTP status code.
func MapHTTPStatus(status int) ErrorKind { return bberr.MapHTTPStatus(status) }
