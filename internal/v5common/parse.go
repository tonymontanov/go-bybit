/*
FILE: internal/v5common/parse.go

DESCRIPTION:
Numeric and string parsing helpers shared by the Bybit V5 profile
packages. Bybit ships every numeric as a JSON string ("0.001", "0",
empty for missing) and the parse-or-default-to-zero semantic is
identical across linear, spot and every other category.

These helpers swallow errors silently. The sibling
`internal/codec.ParseDecimal` / `ParseInt64` still expose strict error
returns for cases where strict parsing matters (e.g. the orderbook
engine refuses a malformed delta).
*/

package v5common

import (
	"github.com/shopspring/decimal"

	"github.com/tonymontanov/go-bybit/v2/internal/codec"
)

// Dec parses s into a decimal.Decimal; returns Zero on empty or
// invalid input.
func Dec(s string) decimal.Decimal {
	var v decimal.Decimal
	var err error
	v, err = codec.ParseDecimal(s)
	if err != nil {
		return decimal.Zero
	}
	return v
}

// Ms parses s into an int64 (used for ms timestamps); returns 0 on
// empty / invalid input.
func Ms(s string) int64 {
	var n int64
	var err error
	n, err = codec.ParseInt64(s)
	if err != nil {
		return 0
	}
	return n
}

// NormalizeRejectReason maps Bybit V5's "no rejection" sentinel
// "EC_NoError" to an empty string so OrderInfo.RejectReason is empty
// when the order was NOT rejected. Bybit ships "EC_NoError" on every
// non-rejected order in REST and WS payloads — surfacing it verbatim
// confuses downstream consumers that branch on `RejectReason != ""`.
//
// Other reason codes (e.g. "EC_PostOnlyWillTakeLiquidity") are
// returned unchanged.
func NormalizeRejectReason(s string) string {
	if s == "EC_NoError" {
		return ""
	}
	return s
}
