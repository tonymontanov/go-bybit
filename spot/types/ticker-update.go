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

WARNING — best bid/ask are NOT delivered on spot tickers:
The Bybit V5 docs page for tickers.{symbol} shows a single field table
that lists `bid1Price` / `ask1Price` / `bid1Size` / `ask1Size`. That
table mixes fields across linear / inverse / option / spot — for the
spot category these four fields are NEVER populated by the exchange.
Verified by live subscription to wss://stream.bybit.com/v5/public/spot:
spot ticker pushes contain only {lastPrice, prevPrice24h, highPrice24h,
lowPrice24h, volume24h, turnover24h, price24hPcnt, usdIndexPrice}.

`BestBid`, `BestBidSize`, `BestAsk`, `BestAskSize` are kept on this
type purely for symmetry with linears (so callers that switch profiles
do not need to reflect on different struct shapes), but on spot they
will remain zero forever. To receive top-of-book on spot, subscribe to
`orderbook.1.{symbol}` via StreamClient.WatchOrderBook(depth=1) — that
channel pushes consistent best bid/ask at ~10ms cadence.
*/

package types

import "github.com/shopspring/decimal"

// TickerUpdate — merged ticker snapshot for the spot category.
//
// On spot, Bybit V5 does NOT populate BestBid / BestBidSize / BestAsk /
// BestAskSize regardless of what the public docs table claims (see
// file-level WARNING). Use StreamClient.WatchOrderBook(depth=1) to
// obtain top-of-book on spot.
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
