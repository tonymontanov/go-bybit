# go-bybit

High-performance Go SDK for the **Bybit V5** exchange API, targeting
HFT / algorithmic trading.

Module path: `github.com/tonymontanov/go-bybit`

## Status

| Module | Status | Notes |
| --- | --- | --- |
| **M0** scaffolding (root client, config, errors, logger, metrics, rate-limit event) | done | unit tests for codec / signer / error mapping / REST transport |
| M0 internal/auth (HMAC-SHA256 hex for REST + WS) | done | canonical-vector + property tests |
| M0 internal/rest (V5 envelope, X-BAPI-* headers, observers) | done | httptest-based tests |
| M0 internal/ws (auth, app-ping, reconnect+jitter, resubscribe, dispatch) | done | mock-server tests |
| **M1** `linears/` REST core (Trading / Account / MarketData) | done | CreateOrder / Modify / Cancel / Batch\* / CancelAll / CancelForgotten / GetOpenOrders / GetPosition / GetWalletBalance / SetLeverage / SetPositionMode / ClosePosition / GetSymbolInfo / GetOrderBook / GetHistoricalCandles + httptest contract tests |
| **M2** `orderbook/` engine (snapshot + delta + u/seq + resync) | done | sequence + service-restart gap detection (no CRC32 — Bybit does not ship one) |
| **M3** `linears/stream.go` (WS subscriptions) | done | public: WatchOrderBook (engine-backed) / WatchTicker (delta merge) / WatchTrades / WatchKline; private: WatchOrders / WatchPositions / WatchExecutions / WatchWallet; mock-WS tests for all paths |
| **M4** errors mapping + examples | done | extended `MapBybitCode` (10009/10017/10029/110003/110004/110009/110012/110020/110025/110052/170140) with table-driven tests; `examples/` for marketdata, signed trade, WS orderbook |
| **M5** `spot/` profile | planned | mirrors `linears/` |
| **MVP+** testnet / demo support | deferred | flags exist already in Config; URL switching is wired but not yet used in v1 |

`v1` covers the **linear** category only. Inverse and option are out of
scope for v1.

## Quick start

```go
import (
    bybit "github.com/tonymontanov/go-bybit"
    "github.com/tonymontanov/go-bybit/linears"
    "github.com/tonymontanov/go-bybit/linears/types"
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

End-to-end runnable demos live under `examples/` (marketdata, trade,
stream-orderbook).

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
    auth/    — HMAC-SHA256 signing for Bybit V5 REST + WS
    bberr/   — Error type, ErrorKind, MapBybitCode / MapHTTPStatus
    bblog/   — Logger interface + Field / NoopLogger
    bbmet/   — Counter / CounterFactory + NoopMetrics
    codec/   — jsoniter wrappers + ParseDecimal / ParseInt64 / RawJSON
    rest/    — low-level HTTP client, V5 envelope { retCode, retMsg, result }
    ws/      — Conn: connect / auth / app-ping / reconnect+jitter / resubscribe
  linears/                # M1+
  spot/                   # M5+
  orderbook/              # M2+
  examples/               # M4+
    marketdata/           # public REST: instrument-info + orderbook
    trade/                # signed REST: PostOnly create + cancel
    stream-orderbook/     # WS: live BTCUSDT order book, top-of-book
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
