/*
FILE: linears/types/balance.go

DESCRIPTION:
Wallet state for the Bybit V5 linear category, sourced from
GET /v5/account/wallet-balance. Bybit's account model:
  - UNIFIED  — Unified Trading Account: a single equity pool covers spot,
               linear, options. This is the recommended account type and
               the one targeted by the linears profile.
  - CONTRACT — Classic Account: derivatives-only sub-wallet. Supported as
               a fallback (older API keys may still default to it).

The struct flattens Bybit's two-level structure (account totals + per-coin
breakdown) into a Balance + []CoinBalance pair. The desk-side adapter
converts these into its own Balance representation.

NOTE on accountType: callers pick the wire string at request time
("UNIFIED" / "CONTRACT"); the SDK exposes constants for both.
*/

package types

import "github.com/shopspring/decimal"

// AccountType is the Bybit V5 wallet partition selector.
type AccountType string

const (
	// AccountTypeUnified — Unified Trading Account.
	AccountTypeUnified AccountType = "UNIFIED"
	// AccountTypeContract — Classic Account derivatives wallet.
	AccountTypeContract AccountType = "CONTRACT"
)

// Balance — account-level wallet state for a single accountType.
type Balance struct {
	// AccountType — UNIFIED / CONTRACT.
	AccountType AccountType
	// TotalEquity — total account equity in USD-equivalent.
	TotalEquity decimal.Decimal
	// TotalWalletBalance — total wallet balance in USD-equivalent.
	TotalWalletBalance decimal.Decimal
	// TotalAvailableBalance — funds available to open new positions.
	TotalAvailableBalance decimal.Decimal
	// TotalMarginBalance — equity committed as margin (open positions
	// + open orders).
	TotalMarginBalance decimal.Decimal
	// TotalInitialMargin — sum of initial margin across positions.
	TotalInitialMargin decimal.Decimal
	// TotalMaintenanceMargin — sum of maintenance margin across positions.
	TotalMaintenanceMargin decimal.Decimal
	// TotalPerpUPL — unrealized PnL across all linear/inverse positions.
	TotalPerpUPL decimal.Decimal
	// AccountIMRate / AccountMMRate — account-level IM / MM rates;
	// 0..1 in normal conditions, > 1 indicates liquidation pressure.
	AccountIMRate decimal.Decimal
	AccountMMRate decimal.Decimal
	// AccountLTV — loan-to-value ratio (only meaningful when borrowing).
	AccountLTV decimal.Decimal
	// Coins — per-currency breakdown.
	Coins []CoinBalance
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
