/*
FILE: asset/doc.go

DESCRIPTION:
Bybit V5 Asset profile — funding-account REST endpoints.

Import anonymously to register the factory:

	import _ "github.com/tonymontanov/go-bybit/v2/asset"

	var ac = c.Asset().(*asset.Client)

Endpoints (C1 scope):
  - GET  /v5/asset/coin/query-info
  - GET  /v5/asset/transfer/query-account-coins-balance
  - GET  /v5/asset/transfer/query-account-coin-balance
  - GET  /v5/asset/transfer/query-transfer-coin-list
  - POST /v5/asset/transfer/inter-transfer
  - GET  /v5/asset/transfer/query-inter-transfer-list
  - GET  /v5/asset/deposit/query-address
  - GET  /v5/asset/deposit/query-record
  - GET  /v5/asset/deposit/query-allowed-list
  - POST /v5/asset/deposit/deposit-to-account
  - POST /v5/asset/withdraw/create
  - POST /v5/asset/withdraw/cancel
  - GET  /v5/asset/withdraw/query-record
  - GET  /v5/asset/withdraw/withdrawable-amount
*/

package asset
