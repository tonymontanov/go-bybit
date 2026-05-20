/*
Package bybit is a high-performance Go SDK for the Bybit V5 exchange API,
targeting HFT / algorithmic trading.

The package is organised as a "fat" domain client (Variant B in the
architecture spec):

  - bybit.Client            — root SDK object: REST transport, signer,
    config, logger; lazily exposes domain
    sub-clients.
  - linears.Client          — Linear category (USDT/USDC perpetual + USDC
    futures). Exposes Trading/Account/MarketData/
    Stream sub-clients.
  - spot.Client             — Spot category (added in a later milestone,
    same shape as linears).

Errors are typed as *bybit.Error (Kind = Network|RateLimit|Auth|
InvalidRequest|Exchange|Unknown). Callers branch on bybit.IsRateLimit /
bybit.IsAuth / etc. The Bybit V5 retCode is preserved in Error.BybitCode
for debugging and contract tests.

Rate-limiting is exposed via Config.RateLimitEventObserver: every REST
response yields one RateLimitEvent with the path, the X-Bapi-Limit-*
headers and structured RequestMeta (OrderCount/Symbols/Category) so an
external rate-limiter can model Bybit's per-(UID+Symbol) and
sub-account-level budgets accurately.

WebSocket streams (orderbook/tickers/positions/orders) reconnect with
exponential backoff + jitter, re-authenticate, and re-subscribe
transparently. The application-level keep-alive ({"op":"ping"} every
20s) is built in; users do not interact with it.

The SDK module path is github.com/tonymontanov/go-bybit. Versioning
follows semver starting at v1.0.0.

Quick start (after the linears package is implemented in M1):

	import (
	    bybit "github.com/tonymontanov/go-bybit"
	    _ "github.com/tonymontanov/go-bybit/linears"
	)

	func main() {
	    var cfg bybit.Config = bybit.DefaultConfig()
	    cfg.APIKey = "..."
	    cfg.SecretKey = "..."

	    var c, err = bybit.NewClient(cfg)
	    if err != nil { panic(err) }
	    defer c.Close()

	    // var lc = c.Linears().(*linears.Client) — see linears/doc.go
	    _ = c
	}
*/
package bybit
