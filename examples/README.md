# go-bybit examples

Runnable end-to-end scenarios for the SDK. Run via the helper script:

```bash
cp .env.example .env
# edit .env: BYBIT_API_KEY, BYBIT_API_SECRET, optionally BYBIT_TESTNET=1
./scripts/run.sh ./examples/<name>
```

## Read-only

| Example | What it exercises | Keys required |
| --- | --- | --- |
| `market-data` | `MarketData.{GetSymbolInfo, GetOrderBook, GetHistoricalCandles}` | no |
| `orderbook-watcher` | `Stream.WatchOrderBook` (engine-backed, prints top of book) | no |
| `public-streams` | `Stream.{WatchOrderBook, WatchTicker, WatchTrades, WatchKline}` in parallel | no |
| `account-info` | `Account.{GetWalletBalance, GetPosition, GetOpenOrders}` | yes (read-only) |
| `inventory-monitor` | `Stream.{WatchOrders, WatchPositions, WatchExecutions, WatchWallet}` (long-running) | yes (read-only) |

## Trading

The trading examples refuse to run against PRODUCTION unless `BYBIT_ALLOW_LIVE=1` is set explicitly.
The guard is bypassed automatically when `BYBIT_TESTNET=1` or `BYBIT_DEMO=1` — recommended for the
first round of testing.

| Example | What it exercises | Keys required |
| --- | --- | --- |
| `simple-trade` | `Trading.{CreateOrder, ModifyOrder, CancelOrder}` (PostOnly limit far from mid) | yes |
| `inventory-tracker` | `Trading.CreateOrder` (market BUY) + private streams + `Account.ClosePosition` | yes |

`inventory-tracker` actually opens and closes a small position. Default size is `0.001` BTC for
BTCUSDT (configurable via `BYBIT_QUANTITY`). The default hold time between open and close is
5 seconds (`BYBIT_HOLD_SECONDS`). Cost on production at $60k BTC is ~$0.10 in fees.

## Environment

See `.env.example` for the full list of variables and their defaults.
