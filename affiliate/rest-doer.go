/*
FILE: affiliate/rest-doer.go
*/

package affiliate

import (
	"context"

	"github.com/tonymontanov/go-bybit/v2/internal/rest"
)

type restDoer interface {
	Do(ctx context.Context, opts rest.Options) (rest.Response, map[string]string, error)
}
