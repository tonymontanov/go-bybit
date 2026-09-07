package spot

import "testing"

// Mirror of linears/symbol-info_test.go: quantity precision must come from
// the normalised step so an integer basePrecision ("1", "10") or a step
// with trailing zeros ("0.0010") maps to the grid Bybit actually enforces.
// Price precision keeps Bybit's scale (tickSize "0.100" → 3) unchanged.
func TestConvertSymbolInfo_PrecisionFromNormalisedStep(t *testing.T) {
	cases := []struct {
		name      string
		basePrec  string
		tick      string
		wantQty   int32
		wantPrice int32
	}{
		{"fractional", "0.000001", "0.01", 6, 2},
		{"whole units", "1", "0.0001", 0, 4},
		{"tens", "10", "0.001", -1, 3},
		{"trailing zeros", "0.0010", "0.100", 3, 3}, // price keeps Bybit scale
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			info := convertSymbolInfo(spotInstrumentsEntry{
				Symbol:        "XUSDT",
				PriceFilter:   spotInstrumentsPriceFilter{TickSize: tc.tick},
				LotSizeFilter: spotInstrumentsLotFilter{BasePrecision: tc.basePrec},
			})
			if info.QuantityPrecision != tc.wantQty || info.PricePrecision != tc.wantPrice {
				t.Errorf("got qty=%d price=%d, want %d/%d",
					info.QuantityPrecision, info.PricePrecision, tc.wantQty, tc.wantPrice)
			}
		})
	}
}
