/*
FILE: linears/stream.go

DESCRIPTION:
StreamClient — WebSocket subscription sub-client for the linear profile.
This file is a STUB intentionally — M1 ships only the REST core. M3 will
replace it with full Watch* methods (orderbook, ticker, kline, position,
order, execution, wallet) backed by internal/ws.Conn.

The StreamClient TYPE is published in M1 so callers can import the
linears package and pre-wire references; method calls return
ErrorKindInvalidRequest with a clear message until M3 lands.
*/

package linears

import (
	"context"
	"errors"
)

// errStreamNotImplemented is returned by every Watch* method until M3
// replaces this stub with a real WebSocket implementation.
var errStreamNotImplemented = errors.New("linears: stream is not implemented yet (M3 milestone); only REST is available in v1.0.0-alpha.0")

// StreamClient — WebSocket subscription sub-client (stub for M1).
type StreamClient struct {
	c *Client
}

func newStreamClient(c *Client) *StreamClient {
	return &StreamClient{c: c}
}

// EnsureConnected is a placeholder lifecycle hook reserved for M3.
// In the final implementation it will Lazy-initialize the public/private
// WS connections, perform auth, and prime resubscription state. The stub
// returns errStreamNotImplemented.
func (s *StreamClient) EnsureConnected(ctx context.Context) error {
	_ = ctx
	return errStreamNotImplemented
}

// Close is a placeholder reserved for M3. The stub is a no-op so callers
// can defer s.Close() unconditionally.
func (s *StreamClient) Close() error { return nil }
