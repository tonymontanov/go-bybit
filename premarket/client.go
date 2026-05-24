/*
FILE: premarket/client.go
*/

package premarket

import (
	bybit "github.com/tonymontanov/go-bybit/v2"
)

// Client — Bybit V5 pre-market perpetual profile client (public REST).
type Client struct {
	parent *bybit.Client
}

// NewClient creates a pre-market client. The parent argument is required.
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

func init() {
	bybit.RegisterPreMarketFactory(func(parent *bybit.Client) any {
		return NewClient(parent)
	})
}
