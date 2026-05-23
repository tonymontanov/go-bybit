/*
FILE: spot/types/candle.go

DESCRIPTION:
Historical kline for the Bybit V5 spot category, mapped from the array
returned by GET /v5/market/kline (positional [startMs, o, h, l, c, v,
turnover]).

Wire shape and ordering (descending by OpenTimeMs) are identical to
linears. For spot:
  - Volume is denominated in BASE asset.
  - VolumeQuote is denominated in QUOTE asset.

There is no contract multiplier on spot, so no conversion is needed.
*/

package types

import "github.com/shopspring/decimal"

// Candle — one historical kline.
type Candle struct {
	OpenTimeMs  int64
	Open        decimal.Decimal
	High        decimal.Decimal
	Low         decimal.Decimal
	Close       decimal.Decimal
	Volume      decimal.Decimal
	VolumeQuote decimal.Decimal
}

// Candles — slice of candles, descending by OpenTimeMs (Bybit's order).
type Candles []Candle
