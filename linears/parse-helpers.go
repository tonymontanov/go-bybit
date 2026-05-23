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

	"github.com/tonymontanov/go-bybit/v2/internal/v5common"
)

// dec is a thin alias over v5common.Dec, retained because the rest of
// this profile uses lowercase, single-letter helpers and switching the
// existing call sites (~150 occurrences) to v5common.Dec would make
// review noisy without a behaviour change.
func dec(s string) decimal.Decimal {
	return v5common.Dec(s)
}

// ms is a thin alias over v5common.Ms (see `dec` rationale).
func ms(s string) int64 {
	return v5common.Ms(s)
}

// normalizeRejectReason is a thin alias over v5common.NormalizeRejectReason.
func normalizeRejectReason(s string) string {
	return v5common.NormalizeRejectReason(s)
}
