/*
FILE: internal/bberr/error.go

DESCRIPTION:
Single error type used everywhere in the SDK. Has three orthogonal axes:

  - Kind:       category from the SDK's perspective. Used by callers to
                react to a class of failures without parsing strings:
                IsRateLimit / IsAuth / IsNetwork / IsInvalidRequest / IsExchange.
  - HTTPStatus: HTTP status as returned by the transport layer. 0 for
                non-HTTP errors (WS, parse, validation).
  - BybitCode:  numeric Bybit V5 retCode as a string (e.g. "10006",
                "110007"). 0 for non-exchange errors.

The Error type wraps an inner cause so errors.Is / errors.As keep working
across the SDK boundary.

The mapping from raw HTTP status / Bybit retCode to ErrorKind is in
codes.go.
*/

package bberr

import (
	"errors"
	"fmt"
	"strconv"
)

// ErrorKind enumerates SDK-level error categories. Keep this set small and
// stable: every value is part of the public Is*() API.
type ErrorKind uint8

const (
	// ErrorKindUnknown — fallback when the SDK could not classify the
	// failure. Should be rare; investigate any occurrence.
	ErrorKindUnknown ErrorKind = iota
	// ErrorKindNetwork — transport-level failures (timeout, conn reset, DNS,
	// ctx cancelled, EOF on WS, etc.). Caller may retry with backoff.
	ErrorKindNetwork
	// ErrorKindRateLimit — Bybit told us we exceeded a rate limit (HTTP 429,
	// retCode 10006/10018, 130150 family for cancel-all). Caller MUST back
	// off; see exchange-specific headers (X-Bapi-Limit-*) for guidance.
	ErrorKindRateLimit
	// ErrorKindAuth — credentials missing/invalid, signature check failed,
	// recv-window expired (10002), permission denied (10005). NOT retryable.
	ErrorKindAuth
	// ErrorKindInvalidRequest — request malformed at SDK build time, or
	// rejected by exchange validation (10001, params errors, qty/price
	// step violations, leverage too low/high, …). NOT retryable without
	// fixing the input.
	ErrorKindInvalidRequest
	// ErrorKindExchange — request reached exchange and was rejected for
	// business reasons (insufficient balance, position cap, etc.). NOT
	// retryable in the general case.
	ErrorKindExchange
)

// String returns the human-readable name. Used in error messages and logs.
func (k ErrorKind) String() string {
	switch k {
	case ErrorKindNetwork:
		return "network"
	case ErrorKindRateLimit:
		return "rate_limit"
	case ErrorKindAuth:
		return "auth"
	case ErrorKindInvalidRequest:
		return "invalid_request"
	case ErrorKindExchange:
		return "exchange"
	case ErrorKindUnknown:
		fallthrough
	default:
		return "unknown"
	}
}

// Error is the single SDK error type. Implements the standard error
// interface plus Unwrap so callers can use errors.Is / errors.As.
type Error struct {
	// Kind — SDK category. Set by the SDK at the point of error creation.
	Kind ErrorKind
	// HTTPStatus — HTTP status code from the response, or 0 if the error
	// did not originate from an HTTP exchange (network, validation, ws).
	HTTPStatus int
	// BybitCode — Bybit V5 retCode as a string. Empty if the error did not
	// come with a retCode (network, validation, parse).
	BybitCode string
	// Message — human-readable description. Includes Bybit retMsg when
	// available. Safe to show to end-users.
	Message string
	// Cause — wrapped underlying error. May be nil.
	Cause error
}

// New constructs an Error.
func New(kind ErrorKind, code, msg string, cause error) *Error {
	return &Error{Kind: kind, BybitCode: code, Message: msg, Cause: cause}
}

// Error implements the error interface. Format is stable enough to be parsed
// by simple tooling but not by other code (consumers should use the typed
// fields).
func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	var prefix string = "bybit: " + e.Kind.String()
	if e.BybitCode != "" {
		prefix = prefix + " (retCode=" + e.BybitCode + ")"
	} else if e.HTTPStatus != 0 {
		prefix = prefix + " (status=" + strconv.Itoa(e.HTTPStatus) + ")"
	}
	if e.Message == "" && e.Cause == nil {
		return prefix
	}
	if e.Cause == nil {
		return prefix + ": " + e.Message
	}
	if e.Message == "" {
		return prefix + ": " + e.Cause.Error()
	}
	return fmt.Sprintf("%s: %s: %s", prefix, e.Message, e.Cause.Error())
}

// Unwrap returns the wrapped cause for use with errors.Is / errors.As.
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// Is implements errors.Is by comparing on Kind for sibling *Error targets.
// Other targets fall back to standard equality via Unwrap.
func (e *Error) Is(target error) bool {
	if e == nil || target == nil {
		return false
	}
	var other *Error
	if errors.As(target, &other) {
		return e.Kind == other.Kind
	}
	return false
}

// IsKind returns true if err is a *Error with the given kind. Convenience
// wrapper used by the public Is* helpers in errors.go.
func IsKind(err error, kind ErrorKind) bool {
	if err == nil {
		return false
	}
	var e *Error
	if !errors.As(err, &e) {
		return false
	}
	return e.Kind == kind
}

// IsNetwork reports whether err is a network-class error.
func IsNetwork(err error) bool { return IsKind(err, ErrorKindNetwork) }

// IsRateLimit reports whether err is a rate-limit error.
func IsRateLimit(err error) bool { return IsKind(err, ErrorKindRateLimit) }

// IsAuth reports whether err is an auth/permission error.
func IsAuth(err error) bool { return IsKind(err, ErrorKindAuth) }

// IsInvalidRequest reports whether err is a validation error.
func IsInvalidRequest(err error) bool { return IsKind(err, ErrorKindInvalidRequest) }

// IsExchange reports whether err is an exchange-level rejection.
func IsExchange(err error) bool { return IsKind(err, ErrorKindExchange) }
