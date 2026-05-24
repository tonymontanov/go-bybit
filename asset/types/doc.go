/*
FILE: asset/types/doc.go

DESCRIPTION:
Domain types for the Bybit V5 Asset REST API (/v5/asset/*). This package
is profile-specific (layer 2) — it does not alias trading types from
linears/ or spot/. Shared wallet enums (AccountType) are re-used from
github.com/tonymontanov/go-bybit/v2/types where applicable; asset-only
enums (TransferStatus, WithdrawStatus) live here.

The asset profile covers funding-account operations: coin metadata,
internal transfers between account types under the same UID, deposit
address/records, and on-chain / off-chain withdrawals. It is orthogonal
to linears/ and spot/ — callers import asset/ when they need capital
movement, not when placing orders.
*/

package types
