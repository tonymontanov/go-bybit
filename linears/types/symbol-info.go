/*
FILE: linears/types/symbol-info.go

DESCRIPTION:
Instrument specification for the Bybit V5 linear category, sourced from
GET /v5/market/instruments-info. Used by callers for tick/lot
quantization and by the desk-side adapter to populate its own SymbolInfo.

UNLIKE OKX SWAP — Bybit linear contracts have NO contract multiplier:
the order quantity is denominated directly in the base asset (BTC, ETH).
This means no CtVal / CtMult fields, and no base↔contracts conversion
is necessary anywhere in the SDK or the desk-side adapter. The desk
adapter populates ContractValue=1 / ContractMultiplier=1 for legacy code
paths.

FIELDS:
  - Symbol            — e.g. "BTCUSDT".
  - BaseCoin          — base asset ("BTC").
  - QuoteCoin         — quote asset ("USDT").
  - SettleCoin        — settlement asset (equals QuoteCoin for USDT-M).
  - ContractType      — Bybit contractType: "LinearPerpetual",
                        "LinearFutures" (USDC futures), etc.
  - Status            — current trading status, e.g. "Trading", "Closed".
  - TickSize          — minimum price increment (priceFilter.tickSize).
  - MinPrice          — minimum allowed order price (priceFilter.minPrice).
  - MaxPrice          — maximum allowed order price (priceFilter.maxPrice).
  - QtyStep           — minimum quantity increment (lotSizeFilter.qtyStep).
  - MinOrderQty       — minimum order quantity.
  - MaxOrderQty       — maximum quantity for a Limit order.
  - MaxMarketOrderQty — maximum quantity for a Market order
                        (lotSizeFilter.maxMktOrderQty).
  - MinNotionalValue  — minimum order notional in quote (5 USDT for most
                        symbols).
  - MinLeverage       — minimum allowed leverage.
  - MaxLeverage       — maximum allowed leverage.
  - LeverageStep      — leverage granularity.
  - PricePrecision    — number of decimal places in TickSize (derived).
  - QuantityPrecision — number of decimal places in QtyStep (derived).
*/

package types

import "github.com/shopspring/decimal"

// SymbolInfo — linear instrument specification.
type SymbolInfo struct {
	Symbol            string
	BaseCoin          string
	QuoteCoin         string
	SettleCoin        string
	ContractType      string
	Status            string
	TickSize          decimal.Decimal
	MinPrice          decimal.Decimal
	MaxPrice          decimal.Decimal
	QtyStep           decimal.Decimal
	MinOrderQty       decimal.Decimal
	MaxOrderQty       decimal.Decimal
	MaxMarketOrderQty decimal.Decimal
	MinNotionalValue  decimal.Decimal
	MinLeverage       decimal.Decimal
	MaxLeverage       decimal.Decimal
	LeverageStep      decimal.Decimal
	PricePrecision    int32
	QuantityPrecision int32
}
