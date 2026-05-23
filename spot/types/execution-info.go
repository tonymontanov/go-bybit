/*
FILE: spot/types/execution-info.go

DESCRIPTION:
ExecutionInfo for the Bybit V5 spot private "execution" WebSocket
topic. Each event represents a single match; partial fills generate
multiple events for the same OrderID.

DIFFERENCES vs LINEARS:
  - No `PositionIdx` (spot has no positions).
  - Adds `IsLeverage` to identify margin-spot fills (UTA accounts).
*/

package types

import "github.com/shopspring/decimal"

// ExecutionInfo — one fill event for the spot category.
type ExecutionInfo struct {
	Symbol        string
	OrderID       string
	ClientOrderID string
	ExecID        string
	Side          SideType
	ExecPrice     decimal.Decimal
	ExecQty       decimal.Decimal
	ExecValue     decimal.Decimal
	ExecFee       decimal.Decimal
	FeeCurrency   string
	IsMaker       bool
	IsLeverage    bool
	ExecTimeMs    int64
}
