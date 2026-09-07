package v5common

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestPrecision(t *testing.T) {
	cases := []struct {
		in   string
		want int32
	}{
		// Fractional steps — the common case, unchanged behaviour.
		{"0.001", 3},
		{"0.01", 2},
		{"0.1", 1},
		{"1", 0},
		// Integer power-of-ten steps: qtyStep=10 on IRYSUSDT linear
		// (2026-09-07) must round to tens, not to whole units.
		{"10", -1},
		{"100", -2},
		{"1000", -3},
		// Trailing zeros must not inflate precision.
		{"0.0010", 3},
		{"0.100", 1},
		{"10.0", -1},
		// Scientific notation parses to a normalised exponent already.
		{"1E1", -1},
		{"1e-3", 3},
		// Non-power-of-ten steps: finest decimal grid containing the step.
		{"5", 0},
		{"0.25", 2},
		{"0.5", 1},
		// Degenerate inputs.
		{"0", 0},
		{"", 0},
		{"-10", 0},
	}
	for _, tc := range cases {
		var d decimal.Decimal
		if tc.in != "" {
			var err error
			d, err = decimal.NewFromString(tc.in)
			if err != nil {
				t.Fatalf("%q: %v", tc.in, err)
			}
		}
		if got := Precision(d); got != tc.want {
			t.Errorf("Precision(%q) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestPrecision_MatchesExponentForFractionalSteps(t *testing.T) {
	// For steps without trailing zeros and with a fractional part,
	// Precision must agree with the previous -Exponent() derivation so the
	// change is invisible to every symbol that was already correct.
	for _, s := range []string{"0.00001", "0.0001", "0.001", "0.01", "0.1", "0.05", "0.25"} {
		d := decimal.RequireFromString(s)
		if got, want := Precision(d), -d.Exponent(); got != want {
			t.Errorf("%s: Precision=%d, -Exponent=%d", s, got, want)
		}
	}
}
