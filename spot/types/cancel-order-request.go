/*
FILE: spot/types/cancel-order-request.go

DESCRIPTION:
Type alias re-export of the protocol-common
`github.com/tonymontanov/go-bybit/v2/types.CancelOrderRequest`. The
wire format is identical across every Bybit V5 category — Symbol is
mandatory and exactly one of OrderID / ClientOrderID must be set.
*/

package types

import commontypes "github.com/tonymontanov/go-bybit/v2/types"

// CancelOrderRequest — order cancellation request. See commontypes.CancelOrderRequest.
type CancelOrderRequest = commontypes.CancelOrderRequest
