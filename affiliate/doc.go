/*
FILE: affiliate/doc.go

DESCRIPTION:
Bybit V5 affiliate / referral profile — affiliate portal and friend invites.

Import anonymously to register the factory:

	import _ "github.com/tonymontanov/go-bybit/v2/affiliate"

	var ac = c.Affiliate().(*affiliate.Client)

Endpoints (C5 scope):
  - GET /v5/affiliate/aff-user-list
  - GET /v5/user/aff-customer-info
  - GET /v5/user/invitation/referrals

NOTE: affiliate endpoints require an API key with Affiliate permission only.
Friend referrals accept any signed key.
*/

package affiliate
