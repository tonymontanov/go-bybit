/*
Package bybit is a high-performance Go SDK for the Bybit V5 exchange API,
targeting HFT / algorithmic trading.

The package is organised as a "fat" domain client (Variant B in the
architecture spec) with a layered type system:

  - bybit.Client            — root SDK object: REST transport, signer,
    config, logger; lazily exposes domain
    sub-clients.
  - linears.Client          — Linear category (USDT/USDC perpetual + USDC
    futures). Exposes Trading/Account/MarketData/
    Stream sub-clients.
  - spot.Client             — Spot category (added in v2.0). Same shape
    as linears.
  - asset.Client            — Asset / funding-account REST (added in
    v2.5-dev). Coin info, internal transfers, deposits, withdrawals.

Type layout (since v2.1):

  - github.com/tonymontanov/go-bybit/v2/types        layer 1 — protocol-common
    types reused by every profile (Side / OrderType / TIF /
    OrderBookLevel / Snapshot / Candle / Timeframe / TradeUpdate /
    KlineUpdate / CancelOrderRequest / Balance / CoinBalance /
    AccountType).
  - github.com/tonymontanov/go-bybit/v2/linears/types layer 2 (profile)
    — alias re-exports of layer 1 + linears-only types
    (PositionIdx, PositionMode, SymbolInfo, OrderInfo,
    Create/Modify Request, ExecutionInfo, TickerUpdate, PositionInfo,
    BatchOrderResult).
  - github.com/tonymontanov/go-bybit/v2/spot/types    layer 2 (profile)
    — alias re-exports of layer 1 + spot-only types (MarginTrading,
    MarketUnit, SymbolInfo, OrderInfo, Create/Modify Request,
    ExecutionInfo, TickerUpdate, BatchOrderResult).

The two profile packages are siblings: neither imports the other; both
import only the neutral layer-1 package. Mixing derivatives methods
into the spot client (or vice versa) is impossible by construction.

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

The SDK module path is github.com/tonymontanov/go-bybit/v2. Versioning
follows semver starting at v1.0.0.

Quick start:

	import (
	    bybit "github.com/tonymontanov/go-bybit/v2"
	    "github.com/tonymontanov/go-bybit/v2/linears"
	    "github.com/tonymontanov/go-bybit/v2/linears/types"
	)

	func main() {
	    var cfg bybit.Config = bybit.DefaultConfig()
	    cfg.APIKey = "..."
	    cfg.SecretKey = "..."

	    var c, err = bybit.NewClient(cfg)
	    if err != nil { panic(err) }
	    defer c.Close()

	    var lc = c.Linears().(*linears.Client)

	    // REST: 50-level orderbook snapshot.
	    var ob, _ = lc.MarketData().GetOrderBook(ctx, "BTCUSDT", 50)
	    _ = ob

	    // WS: live engine-backed top-of-book updates.
	    _ = lc.Stream().WatchOrderBook(ctx, "BTCUSDT", 50, 5,
	        func(s types.OrderBookSnapshot) { _ = s },
	        func(err error) { _ = err },
	    )
	}

End-to-end runnable demos live under examples/ (marketdata, trade,
stream-orderbook).
*/
package bybit
