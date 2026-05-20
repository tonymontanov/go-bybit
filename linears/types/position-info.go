/*
FILE: linears/types/position-info.go

DESCRIPTION:
Position state for the Bybit V5 linear category. Returned by Account.
GetPosition and pushed via the private "position" WebSocket topic.

Bybit reports a position even when it is empty: in that case Side is the
empty string and Quantity is 0. Hedge accounts report two rows per
symbol (PositionIdx 1 and 2); one-way accounts report a single row with
PositionIdx 0.

FIELDS:
  - Symbol         — symbol (e.g. "BTCUSDT").
  - Side           — Buy / Sell / "" (empty position).
  - PositionIdx    — 0 (one-way) / 1 (hedge buy) / 2 (hedge sell).
  - Quantity       — position size in BASE asset. Always non-negative;
                     direction is encoded in Side.
  - AvgEntryPrice  — average entry price.
  - MarkPrice      — current mark price.
  - LiqPrice       — liquidation price. 0 when not applicable (no
                     position or fully cross-collateralized).
  - Leverage       — current leverage as a decimal (e.g. "10" → 10).
  - UnrealizedPnL  — unrealized PnL in the settlement currency (USDT).
  - RealizedPnL    — cumulative realized PnL (cumRealisedPnl).
  - PositionValue  — notional value in quote (positionValue).
  - UpdatedAtMs    — last update timestamp (ms).
*/

package types

import "github.com/shopspring/decimal"

// PositionInfo — position state for the linear category.
type PositionInfo struct {
	Symbol        string
	Side          SideType
	PositionIdx   PositionIdx
	Quantity      decimal.Decimal
	AvgEntryPrice decimal.Decimal
	MarkPrice     decimal.Decimal
	LiqPrice      decimal.Decimal
	Leverage      decimal.Decimal
	UnrealizedPnL decimal.Decimal
	RealizedPnL   decimal.Decimal
	PositionValue decimal.Decimal
	UpdatedAtMs   int64
}

// IsEmpty reports whether the position carries zero size. Bybit returns
// such rows after a full close; they are distinct from "no row at all".
func (p PositionInfo) IsEmpty() bool {
	return p.Quantity.IsZero()
}
