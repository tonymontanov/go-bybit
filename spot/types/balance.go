/*
FILE: spot/types/balance.go

DESCRIPTION:
Wallet state for the Bybit V5 spot category, sourced from
GET /v5/account/wallet-balance.

Bybit's account model for spot:
  - UNIFIED — Unified Trading Account: a single equity pool covers spot,
              linear, options. UTA is the recommended account type and
              REQUIRED for spot batch trading and spot WebSocket private
              streams.
  - SPOT    — Classic spot wallet (legacy API keys). Limited capability:
              no batch endpoints, no UTA-style cross-margin.

The struct flattens Bybit's two-level structure (account totals +
per-coin breakdown) into a Balance + []CoinBalance pair, matching the
linears profile. UTA-only metrics (TotalMarginBalance, AccountIMRate,
AccountLTV, ...) are populated for UNIFIED and zero-valued for SPOT.

NOTE on accountType: callers pick the wire string at request time
("UNIFIED" / "SPOT"); the SDK exposes constants for both.
*/

package types

import "github.com/shopspring/decimal"

// AccountType is the Bybit V5 wallet partition selector for spot.
type AccountType string

const (
	// AccountTypeUnified — Unified Trading Account.
	AccountTypeUnified AccountType = "UNIFIED"
	// AccountTypeSpot — classic spot wallet.
	AccountTypeSpot AccountType = "SPOT"
)

// Balance — account-level wallet state for a single accountType.
type Balance struct {
	AccountType            AccountType
	TotalEquity            decimal.Decimal
	TotalWalletBalance     decimal.Decimal
	TotalAvailableBalance  decimal.Decimal
	TotalMarginBalance     decimal.Decimal
	TotalInitialMargin     decimal.Decimal
	TotalMaintenanceMargin decimal.Decimal
	TotalPerpUPL           decimal.Decimal
	AccountIMRate          decimal.Decimal
	AccountMMRate          decimal.Decimal
	AccountLTV             decimal.Decimal
	Coins                  []CoinBalance
}

// CoinBalance — wallet state for a single asset within Balance.
type CoinBalance struct {
	Coin                string
	Equity              decimal.Decimal
	WalletBalance       decimal.Decimal
	UsdValue            decimal.Decimal
	UnrealizedPnL       decimal.Decimal
	CumRealizedPnL      decimal.Decimal
	BorrowAmount        decimal.Decimal
	AvailableToWithdraw decimal.Decimal
	AvailableToBorrow   decimal.Decimal
	Locked              decimal.Decimal
	TotalOrderIM        decimal.Decimal
	TotalPositionIM     decimal.Decimal
	TotalPositionMM     decimal.Decimal
	AccruedInterest     decimal.Decimal
	SpotHedgingQty      decimal.Decimal
	MarginCollateral    bool
	CollateralSwitch    bool
}
