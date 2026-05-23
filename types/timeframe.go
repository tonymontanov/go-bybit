/*
FILE: types/timeframe.go

DESCRIPTION:
Bybit V5 kline `interval` enum — protocol-common across every category.
Bybit treats minute-based intervals as plain integer strings ("1", "3",
"5", ..., "720") and uses single uppercase letters for daily and above
("D", "W", "M"). The Wire() method returns the exact value the
GET /v5/market/kline endpoint expects.
*/

package types

// Timeframe is a closed enum of kline intervals supported by Bybit V5.
type Timeframe string

const (
	// Timeframe1m — 1 minute.
	Timeframe1m Timeframe = "1"
	// Timeframe3m — 3 minutes.
	Timeframe3m Timeframe = "3"
	// Timeframe5m — 5 minutes.
	Timeframe5m Timeframe = "5"
	// Timeframe15m — 15 minutes.
	Timeframe15m Timeframe = "15"
	// Timeframe30m — 30 minutes.
	Timeframe30m Timeframe = "30"
	// Timeframe1h — 1 hour.
	Timeframe1h Timeframe = "60"
	// Timeframe2h — 2 hours.
	Timeframe2h Timeframe = "120"
	// Timeframe4h — 4 hours.
	Timeframe4h Timeframe = "240"
	// Timeframe6h — 6 hours.
	Timeframe6h Timeframe = "360"
	// Timeframe12h — 12 hours.
	Timeframe12h Timeframe = "720"
	// Timeframe1d — 1 day.
	Timeframe1d Timeframe = "D"
	// Timeframe1w — 1 week.
	Timeframe1w Timeframe = "W"
	// Timeframe1M — 1 month.
	Timeframe1M Timeframe = "M"
)

// Wire returns the Bybit V5 string representation of the timeframe.
func (t Timeframe) Wire() string {
	return string(t)
}
