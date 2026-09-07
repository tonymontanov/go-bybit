package linears

import "testing"

// Regression for 2026-09-07: IRYSUSDT linear ships qtyStep="10". The old
// derivation (-Exponent()) reported QuantityPrecision 0, callers rounded to
// whole units (332), Bybit truncated to the step (330) and then rejected
// the order under minNotionalValue with retCode 110094.
func TestConvertSymbolInfo_IntegerQtyStep(t *testing.T) {
	src := instrumentsEntry{
		Symbol:       "IRYSUSDT",
		ContractType: "LinearPerpetual",
		Status:       "Trading",
		BaseCoin:     "IRYS",
		QuoteCoin:    "USDT",
		SettleCoin:   "USDT",
		PriceFilter:  instrumentsPriceFilter{MinPrice: "0.000001", MaxPrice: "1999.999998", TickSize: "0.000001"},
		LotSizeFilter: instrumentsLotFilter{
			MaxOrderQty:         "3200000",
			MinOrderQty:         "10",
			QtyStep:             "10",
			PostOnlyMaxOrderQty: "3200000",
			MinNotionalValue:    "5",
			MaxMktOrderQty:      "640000",
		},
	}
	info := convertSymbolInfo(src)

	if info.QuantityPrecision != -1 {
		t.Errorf("QuantityPrecision = %d, want -1 (qtyStep=10 → round to tens)", info.QuantityPrecision)
	}
	if info.PricePrecision != 6 {
		t.Errorf("PricePrecision = %d, want 6", info.PricePrecision)
	}
	if !info.QtyStep.Equal(dec("10")) {
		t.Errorf("QtyStep = %s, want 10", info.QtyStep)
	}
}

func TestConvertSymbolInfo_FractionalQtyStepUnchanged(t *testing.T) {
	src := instrumentsEntry{
		Symbol:        "BTCUSDT",
		PriceFilter:   instrumentsPriceFilter{TickSize: "0.1"},
		LotSizeFilter: instrumentsLotFilter{QtyStep: "0.001", MinOrderQty: "0.001"},
	}
	info := convertSymbolInfo(src)
	if info.QuantityPrecision != 3 || info.PricePrecision != 1 {
		t.Errorf("got qty=%d price=%d, want 3/1", info.QuantityPrecision, info.PricePrecision)
	}
}
