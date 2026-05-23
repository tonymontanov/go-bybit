/*
FILE: spot/types/timeframe.go

DESCRIPTION:
Bybit V5 kline `interval` enum for the spot category. The wire format
is identical to linear ("1" / "3" / ... / "D" / "W" / "M") so the type
duplicates the linears one rather than importing it (see spot/types/doc.go
for the rationale).
*/

package types

// Timeframe is a closed enum of kline intervals supported by Bybit V5.
type Timeframe string

const (
	Timeframe1m  Timeframe = "1"
	Timeframe3m  Timeframe = "3"
	Timeframe5m  Timeframe = "5"
	Timeframe15m Timeframe = "15"
	Timeframe30m Timeframe = "30"
	Timeframe1h  Timeframe = "60"
	Timeframe2h  Timeframe = "120"
	Timeframe4h  Timeframe = "240"
	Timeframe6h  Timeframe = "360"
	Timeframe12h Timeframe = "720"
	Timeframe1d  Timeframe = "D"
	Timeframe1w  Timeframe = "W"
	Timeframe1M  Timeframe = "M"
)

// Wire returns the Bybit V5 string representation of the timeframe.
func (t Timeframe) Wire() string {
	return string(t)
}
