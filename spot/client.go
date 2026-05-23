/*
FILE: spot/client.go

DESCRIPTION:
Root sub-client for the Bybit V5 spot profile. Holds a reference to
the parent bybit.Client (REST, Signer, Logger, Config) and exposes four
domain sub-clients — Trading, Account, MarketData, Stream.

CONTRACT:
  - Client is safe for concurrent use; sub-clients are read-only after
    construction.
  - All REST calls go through parent.REST() — shared connection pool
    with the linears profile.
  - Public WS opens against the spot endpoint (cfg.WS.PublicSpotURL);
    private WS uses cfg.WS.PrivateURL (shared per UID with linears).

ID CONSTRAINTS:
  - orderLinkId in Bybit V5 spot must match [A-Za-z0-9_.-]{1,36}; the
    SDK validates this at request-build time and returns
    ErrorKindInvalidRequest without sending. Identical rule to linears.
*/

package spot

import (
	"sync"

	bybit "github.com/tonymontanov/go-bybit/v2"
	"github.com/tonymontanov/go-bybit/v2/internal/ws"
)

// Client — Bybit V5 spot profile client.
type Client struct {
	parent *bybit.Client

	trading    *TradingClient
	account    *AccountClient
	marketData *MarketDataClient
	stream     *StreamClient

	publicWsOnce  sync.Once
	publicWs      *ws.Conn
	privateWsOnce sync.Once
	privateWs     *ws.Conn
}

// NewClient creates a spot client. The parent argument is required.
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

// Account returns the account / wallet sub-client.
func (c *Client) Account() *AccountClient { return c.account }

// MarketData returns the public market-data sub-client.
func (c *Client) MarketData() *MarketDataClient { return c.marketData }

// Stream returns the WebSocket subscription sub-client.
func (c *Client) Stream() *StreamClient { return c.stream }

// Internal shortcuts shared by sub-clients.
func (c *Client) logger() bybit.Logger { return c.parent.Logger() }
func (c *Client) rest() restDoer       { return c.parent.REST() }
func (c *Client) config() bybit.Config { return c.parent.Config() }
func (c *Client) signerEnabled() bool  { return c.parent.Signer().Enabled() }

// publicConn lazily creates and returns the public spot WS connection.
// Shared by every Watch* method on StreamClient that talks to the
// public endpoint.
func (c *Client) publicConn() *ws.Conn {
	c.publicWsOnce.Do(func() {
		var cfg bybit.Config = c.parent.Config()
		c.publicWs = ws.NewConn(
			toWsConfig(cfg, cfg.WS.PublicSpotURL, false),
			c.parent.Signer(),
			cfg.Logger,
			cfg.Metrics,
		)
	})
	return c.publicWs
}

// privateConn lazily creates and returns the private WS connection.
func (c *Client) privateConn() *ws.Conn {
	c.privateWsOnce.Do(func() {
		var cfg bybit.Config = c.parent.Config()
		c.privateWs = ws.NewConn(
			toWsConfig(cfg, cfg.WS.PrivateURL, true),
			c.parent.Signer(),
			cfg.Logger,
			cfg.Metrics,
		)
	})
	return c.privateWs
}

// toWsConfig copies the public WsConfig into the internal ws.Config.
// Mirrors linears.toWsConfig — duplicated rather than imported because
// we keep profile packages decoupled (see spot/types/doc.go).
func toWsConfig(cfg bybit.Config, url string, private bool) ws.Config {
	return ws.Config{
		URL:                     url,
		IsPrivate:               private,
		HandshakeTimeout:        cfg.WS.HandshakeTimeout,
		ReadTimeout:             cfg.WS.ReadTimeout,
		WriteTimeout:            cfg.WS.WriteTimeout,
		PingInterval:            cfg.WS.PingInterval,
		AuthExpiresWindow:       cfg.WS.AuthExpiresWindow,
		AuthTimeout:             cfg.WS.AuthTimeout,
		ReconnectInitialBackoff: cfg.WS.ReconnectInitialBackoff,
		ReconnectMaxBackoff:     cfg.WS.ReconnectMaxBackoff,
		ReconnectJitter:         cfg.WS.ReconnectJitter,
		ReadBufferSize:          cfg.WS.ReadBufferSize,
		WriteBufferSize:         cfg.WS.WriteBufferSize,
	}
}

// init registers the factory in the root package so that bybit.Client.
// Spot() lazily returns *spot.Client.
func init() {
	bybit.RegisterSpotFactory(func(parent *bybit.Client) any {
		return NewClient(parent)
	})
}
