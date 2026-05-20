/*
FILE: internal/codec/codec.go

DESCRIPTION:
Thin wrapper around json-iterator/go used everywhere on the SDK hot-path
(WS message dispatch, REST envelope decoding). The standard `encoding/json`
package is intentionally not used directly — jsoniter is 2-3x faster and
allocates less, which matters for ~1k WS messages/sec on a busy market.

Also exposes two small helpers used across REST/WS parsers:

  - ParseDecimal: returns shopspring/decimal.Decimal from a string. Empty
    string maps to decimal.Zero (Bybit very often sends "" for optional
    fields). Returns error on a malformed value.
  - ParseInt64:   returns int64 from a string in the same fashion. Empty
    string maps to 0.

The helpers exist because Bybit V5 sends ALL numbers as strings, and the
typical pattern across domain methods is "take string field, convert,
fallback to zero on missing".
*/

package codec

import (
	"bytes"
	"strconv"

	jsoniter "github.com/json-iterator/go"
	"github.com/shopspring/decimal"
)

// RawJSON is a json.RawMessage equivalent that works correctly with
// jsoniter. Use it inside structs whose fields are forwarded to a
// secondary decoder (typed per topic / per endpoint).
type RawJSON []byte

// MarshalJSON implements json.Marshaler.
func (m RawJSON) MarshalJSON() ([]byte, error) {
	if len(m) == 0 {
		return []byte("null"), nil
	}
	return []byte(m), nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (m *RawJSON) UnmarshalJSON(data []byte) error {
	*m = append((*m)[:0], data...)
	return nil
}

// IsNull reports whether the raw payload is empty or literally "null".
func (m RawJSON) IsNull() bool {
	return len(m) == 0 || bytes.Equal(m, []byte("null"))
}

// jsonAPI is the configured jsoniter instance. ConfigCompatibleWithStandardLibrary
// is used so encoding behavior matches encoding/json (case-insensitive field
// matching disabled, RFC-compliant numeric handling). Same choice as in go-okx.
var jsonAPI = jsoniter.ConfigCompatibleWithStandardLibrary

// Marshal serializes v to JSON.
func Marshal(v any) ([]byte, error) {
	return jsonAPI.Marshal(v)
}

// Unmarshal parses raw into dest.
func Unmarshal(raw []byte, dest any) error {
	return jsonAPI.Unmarshal(raw, dest)
}

// ParseDecimal converts a Bybit numeric string into a decimal.Decimal.
// Empty input → decimal.Zero, no error. Used everywhere because Bybit V5
// returns numbers as strings ("0.00010", "", "1234.5"); the natural Go
// type is shopspring/decimal for HFT-grade precision.
func ParseDecimal(s string) (decimal.Decimal, error) {
	if s == "" {
		return decimal.Zero, nil
	}
	return decimal.NewFromString(s)
}

// ParseInt64 converts a Bybit numeric string to int64.
// Empty input → 0, no error.
func ParseInt64(s string) (int64, error) {
	if s == "" {
		return 0, nil
	}
	return strconv.ParseInt(s, 10, 64)
}

// ParseFloat64 converts a Bybit numeric string to float64.
// Empty input → 0, no error. Used only at boundaries where downstream
// code requires float64 (e.g. desk adapter); inside the SDK prefer
// decimal.Decimal everywhere.
func ParseFloat64(s string) (float64, error) {
	if s == "" {
		return 0, nil
	}
	return strconv.ParseFloat(s, 64)
}
