/*
FILE: types/candle.go

DESCRIPTION:
Historical kline (candlestick) — protocol-common across every Bybit V5
category. Mapped from the array returned by GET /v5/market/kline:

	[ startMs, open, high, low, close, volume, turnover ]

All numeric fields are strings on the wire and are normalised into
decimal.Decimal here. Bybit V5 sorts klines descending by start time
(newest first) — the SDK preserves that order to match Bybit
documentation; callers that prefer chronological order should reverse.

PROFILE NOTES (informational, no schema impact):
  - For the linear profile Volume is denominated in BASE asset and
    Turnover in QUOTE asset; there is no contract multiplier.
  - For the spot profile Volume / Turnover are likewise in
    base / quote — also no multiplier.

Bybit does NOT include a "closed" flag in REST responses; only the
most recent (newest) candle in a kline.{tf}.{symbol} stream message has
type "snapshot" and is treated as still forming. For REST historical
fetches all candles are considered closed.
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
