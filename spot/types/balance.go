/*
FILE: spot/types/balance.go

DESCRIPTION:
Wallet state for the Bybit V5 spot profile. The struct shapes
(`Balance`, `CoinBalance`) are protocol-common — sourced from
`github.com/tonymontanov/go-bybit/v2/types` via type aliases. The
`AccountType` selector is also aliased on the common enum, but only
the values relevant to the spot profile (UNIFIED / SPOT) are
re-exported here:

  - UNIFIED — Unified Trading Account; the recommended account type
              and REQUIRED for spot batch trading and spot WebSocket
              private streams.
  - SPOT    — Classic spot wallet (legacy API keys). Limited
              capability: no batch endpoints, no UTA-style
              cross-margin.

UTA-only metrics inside `Balance` (TotalMarginBalance, AccountIMRate,
AccountLTV, ...) are populated for UNIFIED and zero-valued for SPOT.
*/

package types

import commontypes "github.com/tonymontanov/go-bybit/v2/types"

// AccountType — Bybit V5 wallet partition selector. See commontypes.AccountType.
type AccountType = commontypes.AccountType

const (
	// AccountTypeUnified — Unified Trading Account.
	AccountTypeUnified = commontypes.AccountTypeUnified
	// AccountTypeSpot — classic spot wallet.
	AccountTypeSpot = commontypes.AccountTypeSpot
)

// Balance — account-level wallet state. See commontypes.Balance.
type Balance = commontypes.Balance

// CoinBalance — wallet state for a single asset within Balance. See commontypes.CoinBalance.
type CoinBalance = commontypes.CoinBalance
