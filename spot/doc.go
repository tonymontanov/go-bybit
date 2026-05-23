/*
Package spot is the Bybit V5 spot domain client. It exposes four
sub-clients (Trading / Account / MarketData / Stream) as a thin layer
over the V5 REST and WebSocket transports provided by the root SDK.

USAGE:

	import (
		bybit "github.com/tonymontanov/go-bybit"
		"github.com/tonymontanov/go-bybit/spot"
	)

	var cfg = bybit.DefaultConfig()
	cfg.APIKey, cfg.SecretKey = "...", "..."
	c, err := bybit.NewClient(cfg)
	if err != nil { panic(err) }
	defer c.Close()

	var s = c.Spot().(*spot.Client)
	info, err := s.MarketData().GetSymbolInfo(ctx, "BTCUSDT")

ACCOUNT MODEL:
  - Spot is fully supported under both classic spot accounts and the
    Unified Trading Account (UTA). UTA is required for batch trading
    and for the spot WebSocket private streams (executions / orders /
    wallet).
  - Margin spot trading lives in UTA only; surface it by setting
    `IsLeverage = true` on CreateOrderRequest. Plain cash spot is the
    default (IsLeverage = false / unset).

CATEGORIES:
  - Hard-pinned to "spot" — every REST call sends category=spot, every
    WS subscription opens against the public spot endpoint.

LIMITS:
  - The orderbook depth filter on /v5/market/orderbook accepts {1, 50,
    200} for spot. The SDK clamps caller-supplied values to this set.
  - V5 endpoints support category=spot; the schema is documented at
    https://bybit-exchange.github.io/docs/v5/intro.

INTEROP:
  - Accepts the same root *bybit.Client as the linears profile; the
    REST connection pool and rate-limit observer are shared.
  - Public WS spot has its own connection (separate URL); private WS
    is unified per UID and may be shared if the linears profile is
    also active. The SDK opens at most one private WS per root client.
*/
package spot
