/*
FILE: spot/types/timeframe.go

DESCRIPTION:
Type alias re-export of the protocol-common
`github.com/tonymontanov/go-bybit/v2/types.Timeframe`. Bybit V5 kline
intervals are the same across every category, so the spot profile
reuses the common type.
*/

package types

import commontypes "github.com/tonymontanov/go-bybit/v2/types"

// Timeframe — Bybit V5 kline interval. See commontypes.Timeframe.
type Timeframe = commontypes.Timeframe

const (
	Timeframe1m  = commontypes.Timeframe1m
	Timeframe3m  = commontypes.Timeframe3m
	Timeframe5m  = commontypes.Timeframe5m
	Timeframe15m = commontypes.Timeframe15m
	Timeframe30m = commontypes.Timeframe30m
	Timeframe1h  = commontypes.Timeframe1h
	Timeframe2h  = commontypes.Timeframe2h
	Timeframe4h  = commontypes.Timeframe4h
	Timeframe6h  = commontypes.Timeframe6h
	Timeframe12h = commontypes.Timeframe12h
	Timeframe1d  = commontypes.Timeframe1d
	Timeframe1w  = commontypes.Timeframe1w
	Timeframe1M  = commontypes.Timeframe1M
)
