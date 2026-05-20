/*
FILE: linears/types/ticker-update.go

DESCRIPTION:
TickerUpdate is the payload of the WebSocket "tickers.{symbol}" topic for
the linear category. Bybit sends a full snapshot first and then deltas
that contain only the changed fields — the SDK merges the deltas into the
last known snapshot before invoking the user handler, so callers always
receive a fully populated TickerUpdate.

FIELDS:
  - Symbol            : Bybit symbol.
  - LastPrice         : last traded price.
  - IndexPrice        : index reference price.
  - MarkPrice         : mark price (used for liquidations).
  - BestBid / BestBidSize / BestAsk / BestAskSize : top-of-book.
  - PrevPrice24h      : price 24h ago.
  - HighPrice24h / LowPrice24h : 24h high/low.
  - Volume24h / Turnover24h    : 24h base/quote volume.
  - FundingRate                : current funding rate.
  - NextFundingTimeMs          : timestamp of the next funding event.
  - OpenInterest / OpenInterestValue : aggregate open interest.
  - TsMs                       : Bybit publish timestamp (ms).
*/

package types

import "github.com/shopspring/decimal"

// TickerUpdate — merged ticker snapshot for the linear category.
type TickerUpdate struct {
	Symbol             string
	LastPrice          decimal.Decimal
	IndexPrice         decimal.Decimal
	MarkPrice          decimal.Decimal
	BestBid            decimal.Decimal
	BestBidSize        decimal.Decimal
	BestAsk            decimal.Decimal
	BestAskSize        decimal.Decimal
	PrevPrice24h       decimal.Decimal
	HighPrice24h       decimal.Decimal
	LowPrice24h        decimal.Decimal
	Volume24h          decimal.Decimal
	Turnover24h        decimal.Decimal
	FundingRate        decimal.Decimal
	NextFundingTimeMs  int64
	OpenInterest       decimal.Decimal
	OpenInterestValue  decimal.Decimal
	TsMs               int64
}
