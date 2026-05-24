/*
FILE: broker/client.go

DESCRIPTION:
Root sub-client for the Bybit V5 Exchange Broker REST API (/v5/broker/*).
Requires signed credentials on the broker master account API key.

CONTRACT:
  - Client is safe for concurrent use.
  - All REST calls go through parent.REST() — shared connection pool.
*/

package broker

import (
	bybit "github.com/tonymontanov/go-bybit/v2"
)

// Client — Bybit V5 exchange broker profile client.
type Client struct {
	parent *bybit.Client
}

// NewClient creates a broker client. The parent argument is required.
func NewClient(parent *bybit.Client) *Client {
	if parent == nil {
		return nil
	}
	return &Client{parent: parent}
}

// Parent returns the root bybit.Client.
func (c *Client) Parent() *bybit.Client { return c.parent }

func (c *Client) logger() bybit.Logger { return c.parent.Logger() }
func (c *Client) rest() restDoer       { return c.parent.REST() }
func (c *Client) config() bybit.Config { return c.parent.Config() }

func init() {
	bybit.RegisterBrokerFactory(func(parent *bybit.Client) any {
		return NewClient(parent)
	})
}
