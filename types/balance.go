/*
FILE: types/balance.go

DESCRIPTION:
Wallet state — protocol-common across every Bybit V5 category, sourced
from GET /v5/account/wallet-balance and the wallet WebSocket topic.

The struct flattens Bybit's two-level structure (account totals +
per-coin breakdown) into a Balance + []CoinBalance pair. UTA-only
metrics (TotalMarginBalance, AccountIMRate, AccountLTV, ...) are
populated for accountType=UNIFIED and zero-valued for the legacy
profile-specific accountTypes (CONTRACT for linears, SPOT for spot).

The accountType selector itself lives in `types/enums.go` as
`AccountType`; profiles re-export the subset they actually use.
*/

package types

import "github.com/shopspring/decimal"

// Balance — account-level wallet state for a single accountType.
type Balance struct {
	// AccountType — UNIFIED / CONTRACT (linears) or UNIFIED / SPOT (spot).
	AccountType AccountType
	// TotalEquity — total account equity in USD-equivalent.
	TotalEquity decimal.Decimal
	// TotalWalletBalance — total wallet balance in USD-equivalent.
	TotalWalletBalance decimal.Decimal
	// TotalAvailableBalance — funds available to open new positions.
	TotalAvailableBalance decimal.Decimal
	// TotalMarginBalance — equity committed as margin (open positions
	// + open orders). UTA-only; zero for legacy accountTypes.
	TotalMarginBalance decimal.Decimal
	// TotalInitialMargin — sum of initial margin across positions.
	TotalInitialMargin decimal.Decimal
	// TotalMaintenanceMargin — sum of maintenance margin across positions.
	TotalMaintenanceMargin decimal.Decimal
	// TotalPerpUPL — unrealised PnL across all linear/inverse positions.
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
