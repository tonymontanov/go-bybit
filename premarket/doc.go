/*
FILE: premarket/doc.go

DESCRIPTION:
Bybit V5 pre-market perpetual profile — public REST for PreLaunch contracts.

Import anonymously to register the factory:

	import _ "github.com/tonymontanov/go-bybit/v2/premarket"

	var pc = c.PreMarket().(*premarket.Client)

Endpoints (C6 scope):
  - GET /v5/market/instruments-info  (status=PreLaunch, preListingInfo)
  - GET /v5/market/risk-limit
  - GET /v5/market/tickers            (preOpenPrice, preQty, curPreListingPhase)

NOTE: pre-market orders use linears.Trading().CreateOrder — no dedicated
trading endpoint exists.
*/

package premarket
