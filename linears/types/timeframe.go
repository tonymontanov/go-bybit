/*
FILE: linears/types/timeframe.go

DESCRIPTION:
Type alias re-export of the protocol-common
`github.com/tonymontanov/go-bybit/v2/types.Timeframe`. Bybit V5 kline
intervals are the same across every category, so the linears profile
reuses the common type.
*/

package types

import commontypes "github.com/tonymontanov/go-bybit/v2/types"

// Timeframe — Bybit V5 kline interval. See commontypes.Timeframe.
type Timeframe = commontypes.Timeframe

const (
	// Timeframe1m — 1 minute.
	Timeframe1m = commontypes.Timeframe1m
	// Timeframe3m — 3 minutes.
	Timeframe3m = commontypes.Timeframe3m
	// Timeframe5m — 5 minutes.
	Timeframe5m = commontypes.Timeframe5m
	// Timeframe15m — 15 minutes.
	Timeframe15m = commontypes.Timeframe15m
	// Timeframe30m — 30 minutes.
	Timeframe30m = commontypes.Timeframe30m
	// Timeframe1h — 1 hour.
	Timeframe1h = commontypes.Timeframe1h
	// Timeframe2h — 2 hours.
	Timeframe2h = commontypes.Timeframe2h
	// Timeframe4h — 4 hours.
	Timeframe4h = commontypes.Timeframe4h
	// Timeframe6h — 6 hours.
	Timeframe6h = commontypes.Timeframe6h
	// Timeframe12h — 12 hours.
	Timeframe12h = commontypes.Timeframe12h
	// Timeframe1d — 1 day.
	Timeframe1d = commontypes.Timeframe1d
	// Timeframe1w — 1 week.
	Timeframe1w = commontypes.Timeframe1w
	// Timeframe1M — 1 month.
	Timeframe1M = commontypes.Timeframe1M
)
