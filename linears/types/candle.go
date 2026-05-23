/*
FILE: linears/types/candle.go

DESCRIPTION:
Type alias re-export of the protocol-common Candle / Candles from
`github.com/tonymontanov/go-bybit/v2/types`. Bybit V5 kline shape is
identical across every category, so the linears profile reuses the
common types.
*/

package types

import commontypes "github.com/tonymontanov/go-bybit/v2/types"

// Candle — one historical kline. See commontypes.Candle.
type Candle = commontypes.Candle

// Candles — slice of candles. See commontypes.Candles.
type Candles = commontypes.Candles
