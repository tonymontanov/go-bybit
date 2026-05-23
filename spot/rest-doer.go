/*
FILE: spot/rest-doer.go

DESCRIPTION:
Declares the minimal restDoer interface that the spot Trading / Account /
MarketData sub-clients depend on. The contract matches the linears
profile so tests across both packages can share fakes.

Using an interface (rather than *rest.Client directly) provides:
 1. Testability — sub-client tests inject a fake REST without standing
    up an http.Client.
 2. Isolation — domain code does not depend on transport
    implementation details.
*/

package spot

import (
	"context"

	"github.com/tonymontanov/go-bybit/v2/internal/rest"
)

// restDoer — minimal REST transport contract.
type restDoer interface {
	Do(ctx context.Context, opts rest.Options) (rest.Response, map[string]string, error)
}
