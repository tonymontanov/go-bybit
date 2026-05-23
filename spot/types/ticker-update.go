/*
FILE: spot/types/ticker-update.go

DESCRIPTION:
TickerUpdate is the payload of the Bybit V5 WebSocket "tickers.{symbol}"
topic for the spot category. Bybit sends a full snapshot first and then
deltas that contain only the changed fields — the SDK merges deltas
into the last known snapshot before invoking the user handler, so
callers always receive a fully populated TickerUpdate.

DIFFERENCES vs LINEARS:
  - No `IndexPrice`, no `MarkPrice` (spot has no mark price model).
  - No `FundingRate`, no `NextFundingTimeMs`, no `OpenInterest`
    (these only exist for derivatives).
  - Adds `UsdIndexPrice` (Bybit's USD reference for non-USDT quote pairs).
*/

package types

import "github.com/shopspring/decimal"

// TickerUpdate — merged ticker snapshot for the spot category.
type TickerUpdate struct {
	Symbol        string
	LastPrice     decimal.Decimal
	BestBid       decimal.Decimal
	BestBidSize   decimal.Decimal
	BestAsk       decimal.Decimal
	BestAskSize   decimal.Decimal
	PrevPrice24h  decimal.Decimal
	HighPrice24h  decimal.Decimal
	LowPrice24h   decimal.Decimal
	Volume24h     decimal.Decimal
	Turnover24h   decimal.Decimal
	UsdIndexPrice decimal.Decimal
	TsMs          int64
}
