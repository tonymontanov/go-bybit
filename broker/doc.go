/*
FILE: broker/doc.go

DESCRIPTION:
Bybit V5 Exchange Broker profile — master-account REST for rebates,
sub-account deposits, and voucher awards.

Import anonymously to register the factory:

	import _ "github.com/tonymontanov/go-bybit/v2/broker"

	var bc = c.Broker().(*broker.Client)

Endpoints (C4 scope):
  - GET  /v5/broker/account-info
  - GET  /v5/broker/earnings-info
  - GET  /v5/broker/asset/query-sub-member-deposit-record
  - POST /v5/broker/award/info
  - POST /v5/broker/award/distribute-award
  - POST /v5/broker/award/distribution-record
*/

package broker
