/*
FILE: affiliate/client.go

DESCRIPTION:
Root sub-client for Bybit V5 affiliate / referral REST. Requires signed
credentials; affiliate endpoints need an API key with Affiliate permission.
*/

package affiliate

import (
	bybit "github.com/tonymontanov/go-bybit/v2"
)

// Client — Bybit V5 affiliate profile client.
type Client struct {
	parent *bybit.Client
}

// NewClient creates an affiliate client. The parent argument is required.
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
	bybit.RegisterAffiliateFactory(func(parent *bybit.Client) any {
		return NewClient(parent)
	})
}
