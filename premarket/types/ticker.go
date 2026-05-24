/*
FILE: premarket/types/ticker.go
*/

package types

import "github.com/shopspring/decimal"

// Ticker — linear/inverse ticker with pre-market fields.
type Ticker struct {
	Symbol               string
	LastPrice            decimal.Decimal
	IndexPrice           decimal.Decimal
	MarkPrice            decimal.Decimal
	Bid1Price            decimal.Decimal
	Ask1Price            decimal.Decimal
	Bid1Size             decimal.Decimal
	Ask1Size             decimal.Decimal
	OpenInterest         decimal.Decimal
	FundingRate          decimal.Decimal
	PreOpenPrice         decimal.Decimal
	PreQty               decimal.Decimal
	CurPreListingPhase   string
	FundingIntervalHour  string
}

// TickerList — tickers response list.
type TickerList struct {
	Tickers []Ticker
}
