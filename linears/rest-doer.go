/*
FILE: linears/rest-doer.go

DESCRIPTION:
Declares the minimal restDoer interface that the Trading / Account /
MarketData sub-clients depend on. Using an interface (rather than
*rest.Client directly) provides two benefits:

 1. Testability — sub-client tests can inject a fake REST without a real
    http.Client.
 2. Isolation — domain code does not depend on transport implementation
    details; the REST client could theoretically be replaced by an async
    pipeline in the future.

restDoer matches the public API of *internal/rest.Client.Do.
*/

package linears

import (
	"context"

	"github.com/tonymontanov/go-bybit/internal/rest"
)

// restDoer — minimal REST transport contract.
type restDoer interface {
	Do(ctx context.Context, opts rest.Options) (rest.Response, map[string]string, error)
}
