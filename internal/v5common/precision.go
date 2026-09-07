/*
FILE: internal/v5common/precision.go

DESCRIPTION:
Derives a "precision" (number of decimal places, possibly negative) from a
Bybit filter step such as tickSize or qtyStep. Shared by the linear and
spot converters so both profiles agree on what a step means.

WHY NOT decimal.Exponent() DIRECTLY:
shopspring/decimal keeps the exponent exactly as parsed. "0.001" parses to
coefficient 1, exponent -3 and yields precision 3 — fine. But an integer
step such as "10" parses to coefficient 10, exponent 0, so -Exponent()
reports precision 0 ("whole units") when the exchange actually demands
multiples of ten. A caller rounding to that precision sends 332 where
Bybit accepts only 330/340; Bybit silently truncates 332 → 330 and then
applies its min-notional check to the truncated quantity — which is how a
5.00 USDT order comes back as "Order does not meet minimum order value
5USDT" (IRYSUSDT linear, qtyStep=10, 2026-09-07). Trailing zeros in a
fractional step ("0.0010") would likewise over-report precision.

Precision normalises the step first: strips trailing zeros from the
coefficient, adjusting the exponent, and returns -exponent. So "10" → -1,
"100" → -2, "1" → 0, "0.001" → 3, "0.0010" → 3, "1E1" → -1.

LIMITATION:
A step that is not a power of ten (e.g. "5", "0.25") has no exact
decimal-places representation; Precision returns the finest decimal
grid that contains it (0 and 2 respectively). Callers that need exact
step compliance must round to QtyStep/TickSize themselves.
*/

package v5common

import (
	"math/big"

	"github.com/shopspring/decimal"
)

var bigTen = big.NewInt(10)

// Precision returns the number of decimal places implied by a step
// value, after stripping trailing zeros. Negative results mean the step
// is a power of ten greater than one ("10" → -1, "100" → -2). Zero and
// negative steps yield 0.
func Precision(step decimal.Decimal) int32 {
	if step.Sign() <= 0 {
		return 0
	}
	var coef *big.Int = new(big.Int).Set(step.Coefficient())
	var exp int32 = step.Exponent()
	var rem = new(big.Int)
	for {
		var q = new(big.Int)
		q.QuoRem(coef, bigTen, rem)
		if rem.Sign() != 0 {
			break
		}
		coef = q
		exp++
	}
	return -exp
}
