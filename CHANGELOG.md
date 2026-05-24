# Changelog

All notable changes to `github.com/tonymontanov/go-bybit` will be
documented in this file. The project follows [Semantic Versioning].

> **Module path note (v2.0.0+):** Per Go's
> [SIV rule](https://go.dev/ref/mod#major-version-suffixes), majors ≥ 2
> live under a `/v2` import path:
> `github.com/tonymontanov/go-bybit/v2`. v1.x stays importable at the
> unsuffixed path. The two majors can coexist in a single binary.

## [Unreleased]

### Added

- **`asset/` (new profile, C1)** — Bybit V5 funding-account REST:
  - `GetCoinInfo` — `/v5/asset/coin/query-info`
  - `GetAllCoinsBalance`, `GetSingleCoinBalance`, `GetTransferableCoins`
  - `CreateInternalTransfer`, `GetInternalTransferRecords`
  - `GetDepositAddress`, `GetDepositRecords`, `GetAllowedDepositCoins`, `SetDepositAccount`
  - `CreateWithdraw`, `CancelWithdraw`, `GetWithdrawRecords`, `GetWithdrawableAmount`
  - `asset/types/` — coin/transfer/deposit/withdraw domain types
  - `examples/asset-coin-info` — read-only smoke test
- **Root client** — `RegisterAssetFactory` / `Client.Asset()` lazy accessor
  (import `_ "github.com/tonymontanov/go-bybit/v2/asset"` or any asset import)
- **`account/` (new profile, C2)** — Bybit V5 extended account REST:
  - `GetAccountInfo` — `/v5/account/info`
  - `GetFeeRate` — `/v5/account/fee-rate`
  - `GetTransactionLog` — `/v5/account/transaction-log`
  - `GetCollateralInfo` — `/v5/account/collateral-info`
  - `GetBorrowHistory` — `/v5/account/borrow-history`
  - `GetCoinGreeks` — `/v5/asset/coin-greeks`
  - `SetMarginMode`, `SetCollateralCoin`, `SetHedgingMode`
  - `account/types/` — domain types
  - `examples/extended-account-info` — read-only smoke test
- **Root client** — `RegisterAccountFactory` / `Client.Account()` lazy accessor
  (import `_ "github.com/tonymontanov/go-bybit/v2/account"` or any account import)
- **`linears/` market extended (C3)**:
  - `GetFundingRateHistory` — `/v5/market/funding/history`
  - `GetOpenInterest` — `/v5/market/open-interest`
  - `linears/types/` — `FundingRateHistory`, `OpenInterestHistory` domain types
  - `examples/market-data` — extended with funding/OI dumps

### Changed

- **`types/enums.go`** — add `AccountTypeFund` / `AccountTypeEarn` for asset
  wallet partitions (FUND / EARN on `/v5/asset/*` wire).

## [v2.1.1] — 2026-05-24

Documentation-only patch on top of the complete v2 line (spot profile
in v2.0.0 + shared `types/` in v2.1.0). No API or wire-format changes.

### Changed

- **`spot/types/ticker-update.go`** — file-level WARNING + struct comment
  explaining that Bybit V5 spot `tickers.{symbol}` never populates
  `bid1Price` / `ask1Price` / `bid1Size` / `ask1Size` (verified live).
  Directs consumers to `StreamClient.WatchOrderBook(depth=1)` for
  top-of-book on spot.
- **`spot/stream.go`** — `WatchTicker` doc-block adds the same
  SPOT TOP-OF-BOOK CAVEAT.

## [v2.1.0] — 2026-05-23

Pulls the protocol-common type definitions out of `linears/types/` and
`spot/types/` into a new neutral package
`github.com/tonymontanov/go-bybit/v2/types/`. The two profile packages
become thin layer-2 wrappers (alias re-exports + profile-specific
types) on top of the layer-1 common package.

This eliminates ~9 files of byte-identical parallel copy-paste between
the two profiles and matches the layout of `github.com/tonymontanov/go-okx/v2`
(top-level `types/` + per-profile alias re-exports).

### Compatibility

The refactor is **non-breaking** at the Go API level:

- All previously-exported names under `linears/types` and `spot/types`
  remain accessible at the same paths.
- Aliases preserve type identity — `linears/types.OrderBookLevel` and
  `spot/types.OrderBookLevel` are now both spelled
  `github.com/tonymontanov/go-bybit/v2/types.OrderBookLevel` under the
  hood. Code that constructs / passes these values continues to
  compile unchanged.
- Constants are re-exported with `const X = commontypes.X`, so call
  sites like `linears/types.SideTypeBuy` or `spot/types.OrderStatusFilled`
  do not need updates.

### Added

- **`types/` (new package)** — protocol-common domain types shared
  across all V5 profiles:
  - `enums.go`: `Category`, `SideType`, `OrderType`,
    `TimeInForceType`, `OrderStatus` (base catalogue), `AccountType`.
  - `order-book-level.go`, `order-book-snapshot.go`,
    `timeframe.go`, `candle.go` (+ `Candles`),
    `trade-update.go`, `kline-update.go`,
    `cancel-order-request.go`, `balance.go` (+ `CoinBalance`).

### Changed

- **`linears/types/`** — `enums.go`, `order-book-level.go`,
  `order-book-snapshot.go`, `timeframe.go`, `candle.go`,
  `trade-update.go`, `kline-update.go`, `cancel-order-request.go`,
  `balance.go` rewritten as alias re-exports of the common package.
  Linears-only types (`PositionIdx`, `PositionMode`, trigger-only
  `OrderStatus` constants `Untriggered` / `Triggered`,
  `SymbolInfo`, `CreateOrderRequest`, `ModifyOrderRequest`,
  `OrderInfo`, `ExecutionInfo`, `TickerUpdate`, `PositionInfo`,
  `BatchOrderResult`) remain declared locally.
- **`spot/types/`** — same alias-rewrite for the protocol-common
  files. Spot-only types (`MarginTrading`, `MarketUnit`,
  `SymbolInfo`, `CreateOrderRequest`, `ModifyOrderRequest`,
  `OrderInfo`, `ExecutionInfo`, `TickerUpdate`, `BatchOrderResult`)
  remain declared locally. The spot profile no longer pulls in
  `linears/types` (the two profile packages are now siblings, not
  cousins).
- **Documentation** — `linears/types/doc.go` and `spot/types/doc.go`
  rewritten to describe the new layered model.

### Internal

- `internal/v5common/` is unchanged. The generic
  `ConvertOrderBookLevels[T]` helper continues to work because the
  per-profile `OrderBookLevel` types alias the same common type.
- The `orderbook/` engine continues to use its own internal `Level`
  type for engine-level isolation; profile packages adapt at the
  boundary as before.

## [v2.0.0] — 2026-05-23

Adds first-class **spot** category support and reorganises a small slice
of internals so the new profile and the existing `linears/` package can
share low-level helpers without breaking each other.

### Module path

Per Go's semantic-import-versioning rule, the module path moves to
`github.com/tonymontanov/go-bybit/v2`. Update consumers with:

```bash
go get github.com/tonymontanov/go-bybit/v2@v2.0.0
```

and rewrite imports `github.com/tonymontanov/go-bybit{,/<sub>}` →
`github.com/tonymontanov/go-bybit/v2{,/<sub>}`. The `linears/` public
type/method surface is otherwise byte-identical to v1.0.0; the rename
is mechanical (`gofmt -r 'a -> b'` works).

The major bump is driven by the import-path move, the new
`internal/v5common` package (changes the import graph), the `spot/`
package becoming a public API, and the `orderbook/` engine now exposing
its own `Level` type instead of `linears/types.OrderBookLevel` (callers
using the engine directly need a one-line conversion; callers using
`linears.Stream` are unaffected).

### Added

- **`spot/` profile** (Bybit V5 spot category).
  - REST: `Trading.{CreateOrder, ModifyOrder, CancelOrder, CreateBatchOrders,
    ModifyBatchOrders, CancelBatchOrders, CancelAllOrders, CancelForgottenOrders}`,
    `Account.{GetWalletBalance, GetOpenOrders}`,
    `MarketData.{GetSymbolInfo, GetOrderBook, GetHistoricalCandles}`.
  - Spot batch endpoints are UTA-only on Bybit; the SDK enforces
    `MaxBatchSize=10` (vs. 20 on linears) and surfaces idempotent
    "order not modified" / "leverage not modified" responses as
    successful no-ops, mirroring linears behaviour.
  - WebSocket: `Stream.{WatchOrderBook, WatchTicker, WatchTrades,
    WatchKline}` (public) and `Stream.{WatchOrders, WatchExecutions,
    WatchWallet}` (private; UTA-only on the wire). Public stream uses
    a dedicated spot WS endpoint (`Config.WS.PublicSpotURL`); the
    private endpoint is shared with linears per UID.
  - `spot/types/`: `SymbolInfo` exposes `MarginTrading` /
    `Innovation` / `BasePrecision` / `MinOrderAmt` / `MaxOrderAmt`;
    `OrderInfo` exposes `MarketUnit` / `IsLeverage`; `TickerUpdate`
    exposes `UsdIndexPrice` (no MarkPrice / FundingRate / OpenInterest
    on spot).
  - Hard-pinned `category=spot` on every REST call; the spot package
    is decoupled from `linears/types` to keep versioning independent.
- **Examples**: `examples/spot-{market-data, orderbook-watcher,
  public-streams, account-info, simple-trade, inventory-monitor}`,
  mirroring the linears examples one-for-one. `exhelp.NewSpotClient`
  helper added so each example stays a few lines of demonstration
  code.

### Changed

- **`internal/v5common/`** (new internal package). Houses numeric
  parsing (`Dec` / `Ms`), V5 reject-reason normalisation, the
  generic `ConvertOrderBookLevels[T]` helper, and the orderbook
  depth clamp. Both `linears/` and `spot/` route through it; per-
  profile thin aliases (`linears.dec`, `spot.dec`, etc.) preserve
  the existing call sites.
- **`orderbook/` engine** is no longer typed against
  `linears/types.OrderBookLevel`. `orderbook.Snapshot` and
  `orderbook.Delta` now use the in-package `orderbook.Level` struct.
  `linears/` and `spot/` ship a tiny conversion layer at the
  package boundary so callers using `Stream.WatchOrderBook` see
  no change. Callers driving the engine directly need to map
  between `orderbook.Level` and their own profile-specific level
  type — typically a single `for { … }` loop.
- **Root client** gains `RegisterSpotFactory` /
  `Client.Spot()` plumbing symmetric to `Linears()`. Importing the
  `spot` package (or any example that does) is what wires the
  factory; the root client remains free of an explicit `spot`
  import to avoid an import cycle.

### Tests

- Full contract suite for `spot/` against `httptest`: instrument
  parsing (Innovation / MarginTrading), depth clamping, paginated
  `GetOpenOrders`, `CreateOrder` happy + Rejected paths, idempotent
  vs. propagated `ModifyOrder` retCodes (10001 vs. 110001),
  `CancelAllOrders`, `CreateBatchOrders` partial-failure shapes.
- Validation suite covers all `build*Body` paths, the orderLinkID
  pattern, the batch-error filter and the depth clamp.

### Notes

- `linears/` users do not need to change anything to upgrade to
  `v2.0.0`. Direct users of the orderbook engine update their level
  type in one place.
- Spot rate-limit budgets are intentionally NOT bundled with the SDK;
  the connector layer in the consuming application owns them.

## [v1.0.0] — 2026-05-23

First **stable** release of the Bybit V5 Go SDK. Promotes the
`v1.0.0-alpha.*` line to GA after a multi-day production soak with
`market-making-desk-core` (Spread Quoter, Backrun Chase Order, Default
templates) — see `bybit-linears-22.5.2026_02.log` and
`bybit-linears-23.5.2026.log` for the qualifying runs.

No public-API changes since `v1.0.0-alpha.1`; this is a stability
checkpoint. Future SDK additions (spot profile, broader API coverage)
will ship under `v2.0.0` / `v2.5.0` per the project roadmap.

## [v1.0.0-alpha.1] — 2026-05-22

### Added

- `linears/types.SymbolInfo` now exposes `MinPrice` and `MaxPrice`
  (`decimal.Decimal`) parsed from Bybit V5 `priceFilter.minPrice` /
  `priceFilter.maxPrice`. Required by callers that must clamp client
  prices into the exchange-allowed band before submitting an order.
- Contract tests (`linears/contract_test.go::TestContract_GetSymbolInfo`)
  assert the new fields are wired through end-to-end.

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
[Unreleased]: https://github.com/tonymontanov/go-bybit/compare/v2.0.0...HEAD
[v2.0.0]: https://github.com/tonymontanov/go-bybit/releases/tag/v2.0.0
[v1.0.0]: https://github.com/tonymontanov/go-bybit/releases/tag/v1.0.0
[v1.0.0-alpha.1]: https://github.com/tonymontanov/go-bybit/releases/tag/v1.0.0-alpha.1
[v1.0.0-alpha.0]: https://github.com/tonymontanov/go-bybit/releases/tag/v1.0.0-alpha.0
