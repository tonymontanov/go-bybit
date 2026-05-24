/*
FILE: account/rest-doer.go
*/

package account

import (
	"context"

	"github.com/tonymontanov/go-bybit/v2/internal/rest"
)

type restDoer interface {
	Do(ctx context.Context, opts rest.Options) (rest.Response, map[string]string, error)
}
