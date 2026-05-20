/*
FILE: linears/types/kline-update.go

DESCRIPTION:
KlineUpdate is one element of the "kline.{interval}.{symbol}" WebSocket
topic. Bybit pushes a fresh kline on every interval boundary and updates
the in-progress kline at most once per second; the "Confirmed" flag
distinguishes the two cases.

FIELDS:
  - Symbol      : Bybit symbol.
  - Interval    : Bybit interval ("1", "5", "60", "D", ...).
  - StartMs     : kline start timestamp (ms).
  - EndMs       : kline close timestamp (ms).
  - Open / High / Low / Close : OHLC.
  - Volume      : volume in base asset.
  - Turnover    : volume in quote asset.
  - Confirmed   : true on the kline-close push; false otherwise.
*/

package types

import "github.com/shopspring/decimal"

// KlineUpdate — one event from the kline.{interval}.{symbol} topic.
type KlineUpdate struct {
	Symbol    string
	Interval  Timeframe
	StartMs   int64
	EndMs     int64
	Open      decimal.Decimal
	High      decimal.Decimal
	Low       decimal.Decimal
	Close     decimal.Decimal
	Volume    decimal.Decimal
	Turnover  decimal.Decimal
	Confirmed bool
}
