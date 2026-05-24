/*
FILE: affiliate/parse-helpers.go
*/

package affiliate

import (
	"github.com/shopspring/decimal"

	"github.com/tonymontanov/go-bybit/v2/internal/v5common"
)

func dec(s string) decimal.Decimal { return v5common.Dec(s) }
func ms(s string) int64             { return v5common.Ms(s) }

func commissionMap(raw map[string]string) map[string]decimal.Decimal {
	if len(raw) == 0 {
		return nil
	}
	var out map[string]decimal.Decimal = make(map[string]decimal.Decimal, len(raw))
	for coin, amount := range raw {
		if amount == "" {
			continue
		}
		out[coin] = dec(amount)
	}
	return out
}
