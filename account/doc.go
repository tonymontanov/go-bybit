/*
FILE: account/doc.go

DESCRIPTION:
Bybit V5 extended account profile — UTA settings, fees, transaction log,
collateral/borrow metadata, and account configuration writes.

Import anonymously to register the factory:

	import _ "github.com/tonymontanov/go-bybit/v2/account"

	var ac = c.Account().(*account.Client)

Endpoints (C2 scope):
  - GET  /v5/account/info
  - GET  /v5/account/fee-rate
  - GET  /v5/account/transaction-log
  - GET  /v5/account/collateral-info
  - GET  /v5/account/borrow-history
  - GET  /v5/asset/coin-greeks
  - POST /v5/account/set-margin-mode
  - POST /v5/account/set-collateral-switch
  - POST /v5/account/set-hedging-mode

NOTE: wallet balance, positions, and open orders remain on the profile-local
AccountClient in linears/ and spot/ (trading-adjacent endpoints).
*/

package account
