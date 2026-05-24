/*
FILE: premarket/types/instrument.go
*/

package types

import (
	"github.com/shopspring/decimal"

	commontypes "github.com/tonymontanov/go-bybit/v2/types"
)

// InstrumentsRequest — filters for GET /v5/market/instruments-info.
type InstrumentsRequest struct {
	Category commontypes.Category
	Symbol   string
	Status   InstrumentStatus
	BaseCoin string
	Limit    int
	Cursor   string
}

// AuctionPhaseWindow — one phase window from preListingInfo.phases.
type AuctionPhaseWindow struct {
	Phase       AuctionPhase
	StartTimeMs int64
	EndTimeMs   int64
}

// AuctionFeeInfo — fee schedule for pre-market auction / continuous trading.
type AuctionFeeInfo struct {
	AuctionFeeRate decimal.Decimal
	TakerFeeRate   decimal.Decimal
	MakerFeeRate   decimal.Decimal
}

// PreListingInfo — pre-market listing metadata on a linear/inverse symbol.
type PreListingInfo struct {
	CurAuctionPhase  AuctionPhase
	Phases           []AuctionPhaseWindow
	AuctionFeeInfo   AuctionFeeInfo
	SkipCallAuction  bool
}

// Instrument — linear/inverse instrument with optional pre-listing block.
type Instrument struct {
	Symbol            string
	SymbolID          int64
	ContractType      string
	Status            string
	BaseCoin          string
	QuoteCoin         string
	SettleCoin        string
	LaunchTimeMs      int64
	TickSize          decimal.Decimal
	MinPrice          decimal.Decimal
	MaxPrice          decimal.Decimal
	QtyStep           decimal.Decimal
	MinOrderQty       decimal.Decimal
	MaxOrderQty       decimal.Decimal
	MaxMarketOrderQty decimal.Decimal
	MinNotionalValue  decimal.Decimal
	MinLeverage       decimal.Decimal
	MaxLeverage       decimal.Decimal
	LeverageStep      decimal.Decimal
	IsPreListing      bool
	PreListingInfo    *PreListingInfo
}

// InstrumentList — paginated instruments-info page.
type InstrumentList struct {
	Instruments    []Instrument
	NextPageCursor string
}
