/*
FILE: spot/parse-helpers.go

DESCRIPTION:
Tiny lossy converters used by REST/WS payload mappers in this package.
They are thin aliases over `internal/v5common` so the spot package can
keep concise call sites (`dec(...)`, `ms(...)`) without each file
importing v5common directly.

The lossy semantic ("default to zero on parse failure") matches the
linears profile: per-field parse errors here are upstream contract
violations and surfacing them at every call site would only add noise.
For strict parsing (e.g. orderbook deltas) call into
`internal/codec.ParseDecimal` / `ParseInt64` directly.
*/

package spot

import (
	"github.com/shopspring/decimal"

	"github.com/tonymontanov/go-bybit/v2/internal/v5common"
)

// dec parses s into decimal.Decimal; Zero on empty / invalid input.
func dec(s string) decimal.Decimal {
	return v5common.Dec(s)
}

// ms parses s into an int64 (used for ms timestamps); 0 on empty /
// invalid input.
func ms(s string) int64 {
	return v5common.Ms(s)
}

// normalizeRejectReason masks Bybit's "EC_NoError" sentinel — see
// internal/v5common.NormalizeRejectReason for the rationale.
func normalizeRejectReason(s string) string {
	return v5common.NormalizeRejectReason(s)
}
