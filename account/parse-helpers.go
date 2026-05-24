/*
FILE: account/parse-helpers.go
*/

package account

import (
	"github.com/shopspring/decimal"

	"github.com/tonymontanov/go-bybit/v2/internal/v5common"
)

func dec(s string) decimal.Decimal { return v5common.Dec(s) }
func ms(s string) int64             { return v5common.Ms(s) }
