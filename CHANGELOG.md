# Changelog

All notable changes to `github.com/tonymontanov/go-bybit` will be
documented in this file. The project follows [Semantic Versioning].

## [Unreleased]

### Changed

- `OrderInfo.RejectReason` is now normalized to an empty string when
  Bybit ships its `"EC_NoError"` sentinel for non-rejected orders.
  Other reason codes are surfaced verbatim. The change is symmetric
  across REST (`Account.GetOpenOrders` / `GetOrderHistory`) and the
  private `order` WebSocket stream. Callers can now reliably branch on
  `o.RejectReason != ""` without filtering "EC_NoError".
- `linears.TradingClient.ModifyOrder`: doc clarified — Bybit's
  `/v5/order/amend` response contains only `orderId / orderLinkId`,
  the SDK echoes the AMENDED fields (`NewPrice` / `NewQuantity`) into
  `OrderInfo`; untouched fields stay at zero. To get the full
  post-amend state poll `GetOpenOrders` or watch the `order` WS stream.

### Examples

- `examples/simple-trade`: amend output now prints only `newQty`
  (the amended field) instead of confusing `price=0`.
- `examples/market-data`: prints `requested=K returned=asks/bids=N`
  and caps the on-screen rows to TOP-10 — Bybit V5 only accepts depth
  in {1, 50, 200, 500}, so the SDK clamps to the nearest allowed
  level; the example now makes that explicit.
- `examples/inventory-tracker`: progress logs around `ClosePosition`
  with a 10-second per-call REST budget so a stuck request fails
  loudly instead of hanging the demo.

## [v1.0.0-alpha.0] — 2026-05-20

Initial public release of the Bybit V5 Go SDK. Covers the **linear**
category (USDT/USDC perpetuals + USDC futures); spot, inverse and option
profiles are out of scope for v1.

### Added

- **Root client / transport** (M0)
  - `bybit.Client` with lazy domain sub-clients (`Linears()`, `Spot()`).
  - HMAC-SHA256 signing for REST (`X-BAPI-*` headers) and WebSocket
    (`{"op":"auth","args":[apiKey, expires, signature]}`).
  - `internal/rest` HTTP client with V5 envelope parsing, rate-limit
    header forwarding (`X-Bapi-Limit-*`), and structured `RateLimitEvent`
    observer.
  - `internal/ws` connection: dial → auth → subscribe → app-level ping,
    transparent reconnect with exponential backoff + jitter, automatic
    resubscribe on every connect.
  - Typed `*bybit.Error` with orthogonal axes (Kind, HTTPStatus,
    BybitCode, Cause) and `Is*` predicates (`IsRateLimit`, `IsAuth`,
    `IsInvalidRequest`, `IsExchange`, `IsNetwork`).
  - `MapBybitCode` / `MapHTTPStatus` mapping with table-driven tests.

- **`linears/` REST core** (M1)
  - Trading: `CreateOrder`, `ModifyOrder`, `CancelOrder` and batch
    variants (chunked at `MaxBatchSize=20`), `CancelAllOrders` (per
    symbol), `CancelForgottenOrders` (age-based).
  - Account: `GetWalletBalance` (UNIFIED + CONTRACT), `GetPosition`,
    paginated `GetOpenOrders`, `SetLeverage`, `SetPositionMode`,
    `ClosePosition` (emulated as a market reduce-only order).
  - MarketData: `GetSymbolInfo`, `GetOrderBook` (depth clamped to
    1/50/200/500), `GetHistoricalCandles`.
  - Local validation surfaces `ErrorKindInvalidRequest` without
    contacting Bybit (orderLinkId pattern, qty > 0, etc.).

- **`orderbook/` engine** (M2)
  - L2 engine with `ApplySnapshot` / `ApplyDelta`, sequence- and
    service-restart-based gap detection (Bybit does not ship CRC32),
    `TopLevels` / `BestBidAsk` / `LastUpdateID` accessors. No locks on
    the read path.

- **`linears/` WebSocket streams** (M3)
  - Public: `WatchOrderBook` (engine-backed, snapshot-vs-delta routed
    via the envelope `type`), `WatchTicker` (delta merging on top of
    the last snapshot), `WatchTrades`, `WatchKline`.
  - Private: `WatchOrders`, `WatchPositions`, `WatchExecutions`,
    `WatchWallet`, all filtered to `category=linear`.
  - Mock-WS tests cover snapshot+delta, sequence gap, ticker delta
    merging, batched trades, kline decode, the auth handshake on the
    private endpoint, and the auth-required short-circuit when keys are
    missing.

- **Examples** (M4)
  - `examples/market-data`: public REST — instrument spec, orderbook
    snapshot, historical klines.
  - `examples/orderbook-watcher`: live order book over WS with a local
    engine (snapshot + deltas).
  - `examples/public-streams`: aggregated public WS (orderbook + ticker
    + trades + 1m klines).
  - `examples/account-info`: read-only signed REST — wallet, position,
    open orders.
  - `examples/inventory-monitor`: private WS — `order`, `position`,
    `execution`, `wallet`.
  - `examples/simple-trade`: signed PostOnly limit create → amend →
    cancel.
  - `examples/inventory-tracker`: end-to-end MARKET BUY → hold →
    `ClosePosition` with live private-stream updates.
  - Shared `examples/internal/exhelp` helper (.env loading, guards,
    classify).

### Notes

- Demo / Testnet support is wired in `Config.Testnet` / `Config.Demo` but
  exercised only superficially in v1.
- `spot/` profile is reserved for a future milestone (`M5+`).

[Semantic Versioning]: https://semver.org
[Unreleased]: https://github.com/tonymontanov/go-bybit/compare/v1.0.0-alpha.0...HEAD
[v1.0.0-alpha.0]: https://github.com/tonymontanov/go-bybit/releases/tag/v1.0.0-alpha.0
