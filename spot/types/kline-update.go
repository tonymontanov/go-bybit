/*
FILE: spot/types/kline-update.go

DESCRIPTION:
KlineUpdate for the Bybit V5 spot WebSocket "kline.{interval}.{symbol}"
topic. Wire shape and semantics are identical to linears.
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
