/*
FILE: spot/types/kline-update.go

DESCRIPTION:
Type alias re-export of the protocol-common
`github.com/tonymontanov/go-bybit/v2/types.KlineUpdate`. The wire
format of the kline.{interval}.{symbol} topic is identical across
every Bybit V5 category, so the spot profile reuses the common type.
*/

package types

import commontypes "github.com/tonymontanov/go-bybit/v2/types"

// KlineUpdate — one event from kline.{interval}.{symbol}. See commontypes.KlineUpdate.
type KlineUpdate = commontypes.KlineUpdate
