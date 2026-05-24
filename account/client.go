/*
FILE: account/client.go

DESCRIPTION:
Root sub-client for Bybit V5 extended account REST (/v5/account/* and
/v5/asset/coin-greeks). Complements profile-local AccountClient methods
in linears/ and spot/ (wallet balance, positions, open orders).

CONTRACT:
  - Client is safe for concurrent use.
  - All REST calls go through parent.REST() — shared connection pool.
  - Every method requires signed credentials (API key + secret).
*/

package account

import (
	bybit "github.com/tonymontanov/go-bybit/v2"
)

// Client — Bybit V5 extended account profile client.
type Client struct {
	parent *bybit.Client
}

// NewClient creates an extended account client. The parent argument is required.
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
	bybit.RegisterAccountFactory(func(parent *bybit.Client) any {
		return NewClient(parent)
	})
}
