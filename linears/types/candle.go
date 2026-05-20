/*
FILE: linears/types/candle.go

DESCRIPTION:
Historical kline (candlestick) for the Bybit V5 linear category. Mapped
from the array returned by GET /v5/market/kline:

	[ startMs, open, high, low, close, volume, turnover ]

All numeric fields are strings on the wire and are normalized into
decimal.Decimal here. Bybit V5 sorts klines descending by start time
(newest first) — the SDK preserves that order to match Bybit
documentation; callers that prefer chronological order should reverse.

For the linear profile:
  - Volume is denominated in BASE asset (BTC, ETH...). No contract
    multiplier conversion is needed.
  - Turnover is denominated in QUOTE asset (USDT/USDC).

Bybit does NOT include a "closed" flag in REST responses; only the most
recent (newest) candle in a kline.linear stream message has type
"snapshot" and is treated as still forming. For REST historical fetches
all candles are considered closed.
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

// Candles — slice of candles. Order matches what Bybit returns
// (descending by OpenTimeMs).
type Candles []Candle
