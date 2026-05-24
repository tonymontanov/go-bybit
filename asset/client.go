/*
FILE: asset/client.go

DESCRIPTION:
Root sub-client for the Bybit V5 Asset REST API (/v5/asset/*). Holds a
reference to the parent bybit.Client and exposes capital-management
methods split across coin.go, transfer.go, deposit.go, withdraw.go.

CONTRACT:
  - Client is safe for concurrent use.
  - All REST calls go through parent.REST() — shared connection pool
    with linears/ and spot/.
  - Every method requires signed credentials (API key + secret).
*/

package asset

import (
	bybit "github.com/tonymontanov/go-bybit/v2"
)

// Client — Bybit V5 asset / funding-account profile client.
type Client struct {
	parent *bybit.Client
}

// NewClient creates an asset client. The parent argument is required.
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
func (c *Client) signerEnabled() bool  { return c.parent.Signer().Enabled() }

func init() {
	bybit.RegisterAssetFactory(func(parent *bybit.Client) any {
		return NewClient(parent)
	})
}
