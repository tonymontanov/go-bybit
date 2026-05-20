/*
FILE: linears/doc.go

DESCRIPTION:
Package linears implements the Bybit V5 LINEAR profile (USDT-M / USDC-M
perpetual futures and USDC-dated futures). It is a "fat" domain client
shaped after the desk's connector contract: the user receives four
sub-clients — Trading, Account, MarketData, Stream — each with their own
methods.

PUBLIC ENTRY POINTS:
  - linears.NewClient(parent *bybit.Client) *Client    — standard path.
  - parent.Linears().(*linears.Client)                 — same lazily.

SUB-CLIENTS:
  - (*Client).Trading()    : CreateOrder, ModifyOrder, CancelOrder + batch
    variants, CancelAllOrders, CancelForgottenOrders.
  - (*Client).Account()    : GetWalletBalance, GetPosition, GetOpenOrders,
    SetLeverage, SetPositionMode, ClosePosition.
  - (*Client).MarketData() : GetSymbolInfo, GetOrderBook,
    GetHistoricalCandles.
  - (*Client).Stream()     : Watch* (WebSocket subscriptions).
    Stub in M1; populated in M3.

TYPES:
All domain structs (CreateOrderRequest, OrderInfo, PositionInfo, ...)
live in the linears/types sub-package and are used by the SDK
sub-clients and the desk-side adapter alike.

Bybit V5 uses category="linear" for this profile. The category value is
hard-pinned by every method in the package; callers never pass it.
*/
package linears
