# go-bybit examples

Runnable end-to-end scenarios for the SDK. Run via the helper script:

```bash
cp .env.example .env
# edit .env: BYBIT_API_KEY, BYBIT_API_SECRET, optionally BYBIT_TESTNET=1
./scripts/run.sh ./examples/<name>
```

## Read-only — linears (USDT/USDC perps)

| Example | What it exercises | Keys required |
| --- | --- | --- |
| `market-data` | `linears.MarketData.{GetSymbolInfo, GetOrderBook, GetHistoricalCandles}` | no |
| `orderbook-watcher` | `linears.Stream.WatchOrderBook` (engine-backed, prints top of book) | no |
| `public-streams` | `linears.Stream.{WatchOrderBook, WatchTicker, WatchTrades, WatchKline}` in parallel | no |
| `account-info` | `linears.Account.{GetWalletBalance, GetPosition, GetOpenOrders}` | yes (read-only) |
| `inventory-monitor` | `linears.Stream.{WatchOrders, WatchPositions, WatchExecutions, WatchWallet}` (long-running) | yes (read-only) |

## Read-only — spot

The spot profile mirrors the linears layout. Keys for the private examples must belong to a UTA
account (Bybit V5 private spot streams and batch endpoints are UTA-only).

| Example | What it exercises | Keys required |
| --- | --- | --- |
| `spot-market-data` | `spot.MarketData.{GetSymbolInfo, GetOrderBook, GetHistoricalCandles}` | no |
| `spot-orderbook-watcher` | `spot.Stream.WatchOrderBook` (engine-backed, prints top of book) | no |
| `spot-public-streams` | `spot.Stream.{WatchOrderBook, WatchTicker, WatchTrades, WatchKline}` in parallel | no |
| `spot-account-info` | `spot.Account.{GetWalletBalance, GetOpenOrders}` | yes (read-only) |
| `spot-inventory-monitor` | `spot.Stream.{WatchOrders, WatchExecutions, WatchWallet}` (long-running) | yes (read-only, UTA) |

## Read-only — asset / extended account (v2.5)

| Example | What it exercises | Keys required |
| --- | --- | --- |
| `asset-coin-info` | `asset.Client.GetCoinInfo` | yes (read-only) |
| `extended-account-info` | `account.Client.{GetAccountInfo, GetFeeRate, GetCollateralInfo}` | yes (read-only) |

## Trading

The trading examples refuse to run against PRODUCTION unless `BYBIT_ALLOW_LIVE=1` is set explicitly.
The guard is bypassed automatically when `BYBIT_TESTNET=1` or `BYBIT_DEMO=1` — recommended for the
first round of testing.

| Example | What it exercises | Keys required |
| --- | --- | --- |
| `simple-trade` | `linears.Trading.{CreateOrder, ModifyOrder, CancelOrder}` (PostOnly limit far from mid) | yes |
| `inventory-tracker` | `linears.Trading.CreateOrder` (market BUY) + private streams + `Account.ClosePosition` | yes |
| `spot-simple-trade` | `spot.Trading.{CreateOrder, ModifyOrder, CancelOrder}` (PostOnly limit far from mid) | yes |

`inventory-tracker` actually opens and closes a small position. Default size is `0.001` BTC for
BTCUSDT (configurable via `BYBIT_QUANTITY`). The default hold time between open and close is
5 seconds (`BYBIT_HOLD_SECONDS`). Cost on production at $60k BTC is ~$0.10 in fees.

## Environment

See `.env.example` for the full list of variables and their defaults.
