/*
FILE: spot/types/symbol-info.go

DESCRIPTION:
Instrument specification for the Bybit V5 spot category, sourced from
GET /v5/market/instruments-info?category=spot. Used by callers for
tick / lot quantization and to pick the right tradeable status.

DIFFERENCES vs LINEARS:
  - No leverage filter (spot has no leverage on the symbol level —
    margin trading is configured at the account level via UTA).
  - Adds `MarginTrading` (string enum, see enums.go) describing whether
    the symbol can be margin-traded and on which account model.
  - Adds `Innovation` flag (Bybit innovation zone — higher-risk symbols
    requiring an explicit risk acknowledgement).
  - No `ContractType` / `SettleCoin` (spot has no settlement asset).

FIELDS:
  - Symbol            — e.g. "BTCUSDT".
  - BaseCoin / QuoteCoin — base and quote asset.
  - Status            — "Trading" / "Closed" / etc.
  - TickSize          — minimum price increment (priceFilter.tickSize).
  - MinPrice          — minimum allowed price (priceFilter.minPrice).
  - MaxPrice          — maximum allowed price (priceFilter.maxPrice).
  - BasePrecision     — decimal places for base quantity (lotSizeFilter.basePrecision).
  - QuotePrecision    — decimal places for quote amount (lotSizeFilter.quotePrecision).
  - MinOrderQty       — minimum base quantity per order.
  - MaxOrderQty       — maximum base quantity per order.
  - MinOrderAmt       — minimum quote notional per order (lotSizeFilter.minOrderAmt).
  - MaxOrderAmt       — maximum quote notional per order (lotSizeFilter.maxOrderAmt).
  - MarginTrading     — see MarginTrading enum.
  - Innovation        — true when the symbol is in Bybit's Innovation zone.
  - PricePrecision    — derived from TickSize (decimal places).
  - QuantityPrecision — derived from BasePrecision (decimal places).
*/

package types

import "github.com/shopspring/decimal"

// SymbolInfo — spot instrument specification.
type SymbolInfo struct {
	Symbol            string
	BaseCoin          string
	QuoteCoin         string
	Status            string
	TickSize          decimal.Decimal
	MinPrice          decimal.Decimal
	MaxPrice          decimal.Decimal
	BasePrecision     decimal.Decimal
	QuotePrecision    decimal.Decimal
	MinOrderQty       decimal.Decimal
	MaxOrderQty       decimal.Decimal
	MinOrderAmt       decimal.Decimal
	MaxOrderAmt       decimal.Decimal
	MarginTrading     MarginTrading
	Innovation        bool
	PricePrecision    int32
	QuantityPrecision int32
}
