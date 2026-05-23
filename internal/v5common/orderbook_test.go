package v5common

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestClampOrderbookDepth(t *testing.T) {
	t.Parallel()
	var linears = []int{1, 50, 200, 500}
	var spot = []int{1, 50, 200}
	type tc struct {
		name    string
		d       int
		allowed []int
		want    int
	}
	var cases []tc = []tc{
		{"linears: zero → smallest", 0, linears, 1},
		{"linears: negative → smallest", -10, linears, 1},
		{"linears: exact match 50", 50, linears, 50},
		{"linears: between 50 and 200", 100, linears, 200},
		{"linears: above max → max", 5000, linears, 500},
		{"spot: 300 → 200 (capped)", 300, spot, 200},
		{"spot: 1 → 1", 1, spot, 1},
		{"empty allowed → passthrough", 42, []int{}, 42},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var got = ClampOrderbookDepth(c.d, c.allowed)
			if got != c.want {
				t.Errorf("ClampOrderbookDepth(%d, %v) = %d, want %d", c.d, c.allowed, got, c.want)
			}
		})
	}
}

type testLevel struct {
	Price decimal.Decimal
	Size  decimal.Decimal
}

func TestConvertOrderBookLevels(t *testing.T) {
	t.Parallel()
	var rows = [][]string{
		{"100.5", "0.001"},
		{"99.9", "2.5"},
		{"bad-row"}, // skipped: < 2 columns
		{"50", "10"},
	}
	var got = ConvertOrderBookLevels(rows, func(p, s decimal.Decimal) testLevel {
		return testLevel{Price: p, Size: s}
	})
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3 (malformed row should be skipped)", len(got))
	}
	if got[0].Price.String() != "100.5" || got[0].Size.String() != "0.001" {
		t.Errorf("row 0: %+v", got[0])
	}
	if got[2].Price.String() != "50" || got[2].Size.String() != "10" {
		t.Errorf("row 2: %+v", got[2])
	}
}

func TestConvertOrderBookLevels_Empty(t *testing.T) {
	t.Parallel()
	var got = ConvertOrderBookLevels[testLevel](nil, func(p, s decimal.Decimal) testLevel {
		return testLevel{p, s}
	})
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}
