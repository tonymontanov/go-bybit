/*
FILE: spot/types/order-book-snapshot.go

DESCRIPTION:
Order book snapshot for the Bybit V5 spot category. Wire shape and
synchronisation model are identical to linear: each push carries a "u"
sequence and snapshots include a "seq" symbol-wide counter; gap
detection is purely sequence-based (no CRC32).

The struct is duplicated from linears/types rather than imported,
because spot consumers should not pick up an implicit dependency on
linears/types. Keeping the two profile packages decoupled simplifies
versioning when one profile evolves faster than the other.
*/

package types

// OrderBookSnapshot — order book snapshot for a single symbol.
type OrderBookSnapshot struct {
	Symbol   string
	Bids     []OrderBookLevel
	Asks     []OrderBookLevel
	UpdateID int64
	SeqID    int64
	TsMs     int64
}
