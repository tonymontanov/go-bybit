/*
FILE: linears/client.go

DESCRIPTION:
Root sub-client for the Bybit V5 linear profile. Holds a reference to
the parent bybit.Client (REST, Signer, Logger, Config) and exposes four
domain sub-clients — Trading, Account, MarketData, Stream.

CONTRACT:
  - Client is safe for concurrent use; sub-clients are read-only after
    construction.
  - All REST calls go through parent.REST() — shared connection pool.
  - The Stream sub-client is a stub in M1 (M3 will replace it with a full
    WebSocket implementation).

ID CONSTRAINTS:
  - orderLinkId in Bybit V5 must match [A-Za-z0-9_.-]{1,36}; the SDK
    validates this at request-build time (see trading.go) and returns
    ErrorKindInvalidRequest without sending.
*/

package linears

import (
	bybit "github.com/tonymontanov/go-bybit"
)

// Client — Bybit V5 linears profile client.
type Client struct {
	parent *bybit.Client

	trading    *TradingClient
	account    *AccountClient
	marketData *MarketDataClient
	stream     *StreamClient
}

// NewClient creates a linears client. The parent argument is required.
func NewClient(parent *bybit.Client) *Client {
	if parent == nil {
		return nil
	}
	var c *Client = &Client{parent: parent}
	c.trading = newTradingClient(c)
	c.account = newAccountClient(c)
	c.marketData = newMarketDataClient(c)
	c.stream = newStreamClient(c)
	return c
}

// Parent returns the root bybit.Client.
func (c *Client) Parent() *bybit.Client { return c.parent }

// Trading returns the trading sub-client.
func (c *Client) Trading() *TradingClient { return c.trading }

// Account returns the account / position sub-client.
func (c *Client) Account() *AccountClient { return c.account }

// MarketData returns the public market-data sub-client.
func (c *Client) MarketData() *MarketDataClient { return c.marketData }

// Stream returns the WebSocket subscription sub-client. M1 ships a stub
// that returns ErrorKindInvalidRequest from Watch*; M3 wires the full
// connection.
func (c *Client) Stream() *StreamClient { return c.stream }

// Internal shortcuts shared by sub-clients.
func (c *Client) logger() bybit.Logger { return c.parent.Logger() }
func (c *Client) rest() restDoer       { return c.parent.REST() }
func (c *Client) config() bybit.Config { return c.parent.Config() }
func (c *Client) signerEnabled() bool  { return c.parent.Signer().Enabled() }

// init registers the factory in the root package so that bybit.Client.
// Linears() lazily returns *linears.Client. This allows users to avoid
// an explicit linears import when working only through the root Client
// (a blank-import of "github.com/tonymontanov/go-bybit/linears" is still
// required in that case).
func init() {
	bybit.RegisterLinearsFactory(func(parent *bybit.Client) any {
		return NewClient(parent)
	})
}
