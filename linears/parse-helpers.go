/*
FILE: linears/parse-helpers.go

DESCRIPTION:
Tiny lossy converters used by REST/WS payload mappers. Bybit V5 sends
all numerics as JSON strings — empty string for "missing", "0" for zero,
otherwise a decimal-encoded value. Most call sites in convertOrderInfo /
convertPosition / convertBalance want a "give me a decimal, default to
zero on parse failure" semantic; surfacing every per-field parse error
would clutter the call sites without a meaningful safety win, since a
malformed numeric on the wire is an upstream contract violation.

These helpers swallow errors silently. The sibling codec.ParseDecimal /
ParseInt64 still expose error returns for cases where strict parsing
matters (e.g. the orderbook engine in M2 needs to refuse a malformed
delta).
*/

package linears

import (
	"github.com/shopspring/decimal"
	"github.com/tonymontanov/go-bybit/internal/codec"
)

// dec parses s into decimal.Decimal; returns Zero on empty/invalid input.
func dec(s string) decimal.Decimal {
	var v decimal.Decimal
	var err error
	v, err = codec.ParseDecimal(s)
	if err != nil {
		return decimal.Zero
	}
	return v
}

// ms parses s into an int64 (used for ms timestamps); 0 on empty/invalid.
func ms(s string) int64 {
	var n int64
	var err error
	n, err = codec.ParseInt64(s)
	if err != nil {
		return 0
	}
	return n
}
