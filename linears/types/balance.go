/*
FILE: linears/types/balance.go

DESCRIPTION:
Wallet state for the Bybit V5 linear profile. The struct shapes
(`Balance`, `CoinBalance`) are protocol-common — sourced from
`github.com/tonymontanov/go-bybit/v2/types` via type aliases. The
`AccountType` selector is also aliased on the common enum, but only
the values relevant to the linears profile (UNIFIED / CONTRACT) are
re-exported here:

  - UNIFIED  — Unified Trading Account; the recommended account type
               and the one targeted by the linears profile.
  - CONTRACT — Classic Account derivatives sub-wallet (legacy API
               keys may still default to it).

The desk-side adapter converts these into its own Balance
representation.
*/

package types

import commontypes "github.com/tonymontanov/go-bybit/v2/types"

// AccountType — Bybit V5 wallet partition selector. See commontypes.AccountType.
type AccountType = commontypes.AccountType

const (
	// AccountTypeUnified — Unified Trading Account.
	AccountTypeUnified = commontypes.AccountTypeUnified
	// AccountTypeContract — Classic Account derivatives wallet.
	AccountTypeContract = commontypes.AccountTypeContract
)

// Balance — account-level wallet state. See commontypes.Balance.
type Balance = commontypes.Balance

// CoinBalance — wallet state for a single asset within Balance. See commontypes.CoinBalance.
type CoinBalance = commontypes.CoinBalance
