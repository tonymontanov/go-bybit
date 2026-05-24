/*
FILE: premarket/types/enums.go
*/

package types

// InstrumentStatus — instruments-info status filter (linear/inverse).
type InstrumentStatus string

const (
	InstrumentStatusTrading    InstrumentStatus = "Trading"
	InstrumentStatusPreLaunch  InstrumentStatus = "PreLaunch"
	InstrumentStatusClosed     InstrumentStatus = "Closed"
	InstrumentStatusDelivering InstrumentStatus = "Delivering"
)

// AuctionPhase — pre-market auction phase name on the wire.
type AuctionPhase string

const (
	AuctionPhaseCallAuction         AuctionPhase = "CallAuction"
	AuctionPhaseCallAuctionNoCancel AuctionPhase = "CallAuctionNoCancel"
	AuctionPhaseCrossMatching       AuctionPhase = "CrossMatching"
	AuctionPhaseContinuousTrading   AuctionPhase = "ContinuousTrading"
)
