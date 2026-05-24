# go-bybit

High-performance Go SDK for the **Bybit V5** exchange API, targeting
HFT / algorithmic trading.

Module path: `github.com/tonymontanov/go-bybit/v2`

Latest stable: **v2.1.1** — see [CHANGELOG.md](CHANGELOG.md).

## Status

`v2.1.0` covers the **linear** and **spot** categories end-to-end.
Inverse and option categories remain out of scope.

| Module | Status | Notes |
| --- | --- | --- |
| **M0** scaffolding (root client, config, errors, logger, metrics, rate-limit event) | done | unit tests for codec / signer / error mapping / REST transport |
| M0 internal/auth (HMAC-SHA256 hex for REST + WS) | done | canonical-vector + property tests |
| M0 internal/rest (V5 envelope, X-BAPI-* headers, observers) | done | httptest-based tests |
| M0 internal/ws (auth, app-ping, reconnect+jitter, resubscribe, dispatch) | done | mock-server tests |
| **M1** `linears/` REST core (Trading / Account / MarketData) | done | CreateOrder / Modify / Cancel / Batch\* / CancelAll / CancelForgotten / GetOpenOrders / GetPosition / GetWalletBalance / SetLeverage / SetPositionMode / ClosePosition / GetSymbolInfo / GetOrderBook / GetHistoricalCandles / GetFundingRateHistory / GetOpenInterest + httptest contract tests |
| **M2** `orderbook/` engine (snapshot + delta + u/seq + resync) | done | sequence + service-restart gap detection (no CRC32 — Bybit does not ship one) |
| **M3** `linears/stream.go` (WS subscriptions) | done | public: WatchOrderBook (engine-backed) / WatchTicker (delta merge) / WatchTrades / WatchKline; private: WatchOrders / WatchPositions / WatchExecutions / WatchWallet; mock-WS tests for all paths |
| **M4** errors mapping + examples | done | extended `MapBybitCode` (10009/10017/10029/110003/110004/110009/110012/110020/110025/110052/170140) with table-driven tests; `examples/` for marketdata, signed trade, WS orderbook |
| **M5** testnet / demo support | done | `Config.Testnet` / `Config.Demo` flags switch REST + WS hosts (`api-testnet.bybit.com` / `api-demo.bybit.com`, `stream-testnet.bybit.com` / `stream-demo.bybit.com`) |
| **M6** `linears/types.SymbolInfo.MinPrice` / `.MaxPrice` | done in `v1.0.0-alpha.1` | parsed from `priceFilter.minPrice` / `priceFilter.maxPrice` |
| **v2.0** `spot/` profile | done | Trading / Account / MarketData / Stream mirroring `linears/`; UTA-only batch + private streams; `internal/v5common` shared helpers; orderbook engine decoupled from profile types |
| **v2.1** shared `types/` (layered model) | done | new top-level `types/` holds protocol-common domain types; `linears/types` and `spot/types` become alias re-exports + profile-specific types — eliminates parallel copy-paste, non-breaking |
| **v2.5** `asset/` profile (C1) | done | coin info, internal transfers, deposit/withdraw REST; examples/asset-coin-info |
| **v2.5** `account/` profile (C2) | done | UTA info, fee rate, transaction log, collateral/borrow, greeks, margin settings |
| **v2.5** `linears/` market extended (C3) | done | funding rate history, open interest REST |
| **v2.5** `broker/` profile (C4) | done | rebates, sub-account deposits, voucher awards |
| **v2.5** `affiliate/` profile (C5) | in progress | affiliate user list/info, friend referrals |
| **v2.5** broader API coverage | planned | pre-market |

## Quick start

```go
import (
    bybit "github.com/tonymontanov/go-bybit/v2"
    "github.com/tonymontanov/go-bybit/v2/linears"
    "github.com/tonymontanov/go-bybit/v2/linears/types"
)

cfg := bybit.DefaultConfig()
cfg.APIKey, cfg.SecretKey = "...", "..."

client, err := bybit.NewClient(cfg)
if err != nil {
    log.Fatal(err)
}
defer client.Close()

lc := client.Linears().(*linears.Client)

// REST: best bid/ask.
ob, _ := lc.MarketData().GetOrderBook(ctx, "BTCUSDT", 50)

// WS: keep top-of-book in sync (engine-backed).
_ = lc.Stream().WatchOrderBook(ctx, "BTCUSDT", 50, 5,
    func(ob types.OrderBookSnapshot) { /* ... */ },
    func(err error) { /* ErrorKindInvalidRequest on gap, etc. */ },
)
```

For the spot profile the entry point is symmetric:

```go
import (
    bybit "github.com/tonymontanov/go-bybit/v2"
    "github.com/tonymontanov/go-bybit/v2/spot"
    spottypes "github.com/tonymontanov/go-bybit/v2/spot/types"
)

sc := client.Spot().(*spot.Client)
info, _ := sc.MarketData().GetSymbolInfo(ctx, "BTCUSDT")
_ = sc.Stream().WatchOrderBook(ctx, "BTCUSDT", 50, 5,
    func(ob spottypes.OrderBookSnapshot) { /* ... */ },
    func(err error) { /* ... */ },
)
```

End-to-end runnable demos live under `examples/` for both profiles
(`market-data` / `spot-market-data`, `simple-trade` / `spot-simple-trade`,
etc. — see `examples/README.md`).

## Dependencies

```
github.com/json-iterator/go      v1.1.12
github.com/shopspring/decimal    v1.4.0
github.com/gorilla/websocket     v1.5.3
```

The same minimal set used by the sibling `go-okx` SDK.

## Layout

```
go-bybit/
  client.go / config.go / doc.go               # public root API
  errors.go / logger.go / metrics.go / rate-limit-event.go
  internal/
    auth/      — HMAC-SHA256 signing for Bybit V5 REST + WS
    bberr/     — Error type, ErrorKind, MapBybitCode / MapHTTPStatus
    bblog/     — Logger interface + Field / NoopLogger
    bbmet/     — Counter / CounterFactory + NoopMetrics
    codec/     — jsoniter wrappers + ParseDecimal / ParseInt64 / RawJSON
    rest/      — low-level HTTP client, V5 envelope { retCode, retMsg, result }
    v5common/  — Bybit V5 helpers shared by linears/ and spot/
                 (numeric parsing, reject-reason normalisation,
                 orderbook depth clamp, generic level conversion)
    ws/        — Conn: connect / auth / app-ping / reconnect+jitter / resubscribe
  types/                  # v2.1+ — protocol-common domain types
                          #         (Side / OrderType / TIF /
                          #          OrderBookLevel / Snapshot /
                          #          Candle / Timeframe / TradeUpdate /
                          #          KlineUpdate / CancelOrderRequest /
                          #          Balance / CoinBalance / AccountType)
  linears/                # v1.0+ — USDT/USDC perps + USDC futures
                          #         linears/types: alias re-exports +
                          #         linears-only (PositionIdx / PositionMode /
                          #         SymbolInfo / OrderInfo / Create/Modify /
                          #         ExecutionInfo / TickerUpdate / PositionInfo)
  spot/                   # v2.0+ — Bybit V5 spot category
                          #         spot/types: alias re-exports +
                          #         spot-only (MarginTrading / MarketUnit /
                          #         SymbolInfo / OrderInfo / Create/Modify /
                          #         ExecutionInfo / TickerUpdate)
  orderbook/              # M2+ — profile-agnostic engine
  examples/               # runnable end-to-end demos (see examples/README.md)
    market-data/          # public REST (linears)
    orderbook-watcher/    # public WS engine-backed top-of-book (linears)
    public-streams/       # WS multi-stream (linears)
    account-info/         # private REST read-only (linears)
    inventory-monitor/    # private WS long-running monitor (linears)
    simple-trade/         # signed REST CRUD (linears)
    inventory-tracker/    # real market BUY → ClosePosition (linears)
    spot-market-data/     # public REST (spot)
    spot-orderbook-watcher/ # public WS engine-backed top-of-book (spot)
    spot-public-streams/  # WS multi-stream (spot)
    spot-account-info/    # private REST read-only (spot)
    spot-inventory-monitor/ # private WS long-running monitor (spot)
    spot-simple-trade/    # signed REST CRUD (spot)
    internal/exhelp/      # shared env loader + error formatter
  scripts/run.sh          # loads .env and forwards to `go run`
```

## Architecture (brief)

Variant B (domain-based): the user receives a "fat" sub-client per
profile (`linears.Client`, `spot.Client`). Each profile exposes four
domain sub-clients:

- `Trading()`     — Create/Modify/Cancel + Batch* + CancelAllOrders + CancelForgottenOrders.
- `Account()`     — Wallet balance, positions, open orders, leverage,
                    position-mode, ClosePosition.
- `MarketData()`  — Symbol-info, order-book snapshot, historical candles.
- `Stream()`      — Watch* (WebSocket subscriptions).

Low-level transport (`internal/rest`, `internal/ws`, `internal/auth`) is
hidden from the user and shared across all sub-clients.

## Errors

All SDK methods return `*bybit.Error`. Branch on `Kind`:

```go
if bybit.IsRateLimit(err) { /* back off */ }
if bybit.IsAuth(err)       { /* terminate */ }
```

The Bybit V5 retCode is preserved in `Error.BybitCode` for debugging.
Selected mapping:

| Bybit retCode | Kind | Notes |
| --- | --- | --- |
| `10001` / `10017` / `110001` / `110008` / `110017` / `170135` / `170140` | InvalidRequest | malformed request, params/qty out of range, route not found |
| `10002` / `10003` / `10004` / `10005` / `10007` / `10009` / `10010` | Auth | signature, recv-window, IP, permission |
| `10006` / `10018` / `10029` / `130150` | RateLimit | IP / UID / system / cancel-all rate limit |
| `10016` | Network | transient server error |
| `110007` / `110004` / `110012` / `170131` | Exchange | balance insufficient (different reasons) |
| `110003` / `110009` / `110020` / `110025` / `110043` / `110052` | InvalidRequest | trading-side validation (price band, max active orders, position-mode mismatch, leverage) |
| anything else | Exchange | preserved verbatim in `Error.BybitCode` |

## Rate-limit observer

```go
cfg.RateLimitEventObserver = func(ev bybit.RateLimitEvent) {
    // ev.Endpoint, ev.Method, ev.Headers,
    // ev.OrderCount, ev.Symbols, ev.Category
}
```

The observer fires once per completed REST response (success or
exchange-level rejection) and is invoked synchronously from the
goroutine that issued the request. Implementations must be O(1) — a
non-blocking send to a buffered channel is the typical shape.

The headers map carries `X-Bapi-Limit` / `X-Bapi-Limit-Status` /
`X-Bapi-Limit-Reset-Timestamp` / `Retry-After` when Bybit returns them.

## WebSocket

- Public streams: `wss://stream.bybit.com/v5/public/{linear,spot,inverse,option}`.
- Private stream: `wss://stream.bybit.com/v5/private`.
- Application-level ping `{"op":"ping"}` every 20s (default).
- Auth payload `{"op":"auth","args":[apiKey, expires, signature]}` with
  signature = `hex(HMAC_SHA256(secret, "GET/realtime" + expires))`.

Reconnect, backoff with jitter, resubscribe and login (for private) are
transparent to the caller.

## Code style

- File-level comments and GoDoc are written in English (this is a public
  project).
- Explicit variable declarations: `var name type = value`.
- `camelCase` for local identifiers, `PascalCase` for exported ones.
- `jsoniter` via `internal/codec` for hot-path JSON; `encoding/json` is
  not used directly.
- Every method takes `context.Context` as the first parameter; passing
  `context.Background()` inside a method that already has a `ctx` is
  forbidden.

## License

See `LICENSE` (Apache-2.0).
