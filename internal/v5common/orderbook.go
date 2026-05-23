/*
FILE: internal/v5common/orderbook.go

DESCRIPTION:
Order-book wire-payload helpers shared by the Bybit V5 profile packages.

`ClampOrderbookDepth` — Bybit's `/v5/market/orderbook?limit=` accepts a
fixed set of integer values that depends on the category:

  - linear / inverse: {1, 50, 200, 500}
  - spot:             {1, 50, 200}
  - option:           {25, 100}

Any other value yields retCode=10001. The SDK clamps the caller's
requested depth to the nearest allowed value rather than refusing,
because callers typically pass "give me top-K" and don't memorise the
exchange's filter table. The allowed set is passed as a parameter
rather than hard-coded so each profile can pick its own.

`ConvertOrderBookLevels` is generic over any struct that exposes
Price / Size `decimal.Decimal` fields. Profile types satisfy this via
a small constructor closure rather than implementing a Go interface
(zero-cost: every Bybit V5 orderbook level is exactly two numerics on
the wire: `[price, size]`).
*/

package v5common

import "github.com/shopspring/decimal"

// ClampOrderbookDepth returns the smallest value in `allowed` that is
// ≥ d. d ≤ 0 resolves to allowed[0] (the first / smallest entry).
// d > max resolves to the maximum.
//
// Behaviour assumes `allowed` is sorted ascending (the SDK call sites
// hard-code the values that match Bybit docs); the helper does NOT
// re-sort to keep the hot path branch-free.
func ClampOrderbookDepth(d int, allowed []int) int {
	if len(allowed) == 0 {
		return d
	}
	if d <= 0 {
		return allowed[0]
	}
	var i int
	for i = 0; i < len(allowed); i++ {
		if d <= allowed[i] {
			return allowed[i]
		}
	}
	return allowed[len(allowed)-1]
}

// ConvertOrderBookLevels converts Bybit's `[][]string` wire payload
// (each row = `[price, size]`) into a slice of caller-defined level
// type T. The constructor `mk` builds T from parsed price / size.
//
// Empty or malformed rows (< 2 columns) are silently skipped — Bybit's
// snapshot format is positional and we should not let a stray entry
// crash a caller who is just trying to display top-of-book.
func ConvertOrderBookLevels[T any](rows [][]string, mk func(price, size decimal.Decimal) T) []T {
	if len(rows) == 0 {
		return nil
	}
	var out []T = make([]T, 0, len(rows))
	var i int
	for i = 0; i < len(rows); i++ {
		if len(rows[i]) < 2 {
			continue
		}
		out = append(out, mk(Dec(rows[i][0]), Dec(rows[i][1])))
	}
	return out
}
