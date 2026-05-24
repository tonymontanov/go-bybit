/*
FILE: account/types/enums.go

DESCRIPTION:
Extended-account enums matching Bybit V5 wire strings.
*/

package types

// MarginMode — unified account margin mode on the wire.
type MarginMode string

const (
	MarginModeIsolated  MarginMode = "ISOLATED_MARGIN"
	MarginModeRegular   MarginMode = "REGULAR_MARGIN"
	MarginModePortfolio MarginMode = "PORTFOLIO_MARGIN"
)

// HedgingMode — spot hedging toggle for portfolio margin accounts.
type HedgingMode string

const (
	HedgingModeOn  HedgingMode = "ON"
	HedgingModeOff HedgingMode = "OFF"
)

// CollateralSwitch — user collateral toggle for a coin.
type CollateralSwitch string

const (
	CollateralSwitchOn  CollateralSwitch = "ON"
	CollateralSwitchOff CollateralSwitch = "OFF"
)
