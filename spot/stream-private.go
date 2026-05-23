/*
FILE: spot/stream-private.go

DESCRIPTION:
Private WebSocket subscriptions for Bybit V5 spot — own orders, fills,
wallet state. All methods route through the StreamClient on the parent
spot.Client.

PRIVATE TOPICS (Bybit V5):
  - "order"     — own order updates across categories. The dispatcher
                  filters by category=="spot" before invoking the
                  handler so the spot stream stays focused.
  - "execution" — own fills.
  - "wallet"    — wallet state (account-wide, no per-category filter).

REQUIRES API CREDENTIALS:
The private endpoint mandates a successful auth handshake. Watch*
methods short-circuit with ErrorKindAuth when API credentials are not
configured. Bybit's UTA-only constraint applies on the wire — classic
spot accounts will see auth fail or receive empty pushes.
*/

package spot

import (
	"context"

	bybit "github.com/tonymontanov/go-bybit/v2"
	"github.com/tonymontanov/go-bybit/v2/internal/codec"
	"github.com/tonymontanov/go-bybit/v2/internal/ws"
	bybitspottypes "github.com/tonymontanov/go-bybit/v2/spot/types"
)

// =====================================================================
// PRIVATE: own orders.
// =====================================================================

// privateOrderEntry — wire shape of one element of the "order" topic
// for category=spot.
type privateOrderEntry struct {
	Category     string `json:"category"`
	OrderID      string `json:"orderId"`
	OrderLinkID  string `json:"orderLinkId"`
	Symbol       string `json:"symbol"`
	Side         string `json:"side"`
	OrderType    string `json:"orderType"`
	TimeInForce  string `json:"timeInForce"`
	Price        string `json:"price"`
	Qty          string `json:"qty"`
	LeavesQty    string `json:"leavesQty"`
	CumExecQty   string `json:"cumExecQty"`
	CumExecValue string `json:"cumExecValue"`
	AvgPrice     string `json:"avgPrice"`
	CumExecFee   string `json:"cumExecFee"`
	OrderStatus  string `json:"orderStatus"`
	MarketUnit   string `json:"marketUnit"`
	IsLeverage   string `json:"isLeverage"`
	RejectReason string `json:"rejectReason"`
	CreatedTime  string `json:"createdTime"`
	UpdatedTime  string `json:"updatedTime"`
}

// WatchOrders subscribes to the private "order" topic. Rows for
// non-spot categories (e.g. linear/option crosstalk) are filtered out.
func (s *StreamClient) WatchOrders(
	ctx context.Context,
	handler func(bybitspottypes.OrderInfo),
	errHandler func(error),
) error {
	if !s.c.signerEnabled() {
		return bybit.NewError(bybit.ErrorKindAuth, "", "stream.WatchOrders: APIKey/SecretKey not configured", nil)
	}
	var sub *ws.Subscription = &ws.Subscription{
		Topic: "order",
		Handler: func(_, _ string, payload []byte) {
			var rows []privateOrderEntry
			if err := codec.Unmarshal(payload, &rows); err != nil {
				s.c.logger().Warn("stream.WatchOrders: parse", bybit.Err(err))
				return
			}
			var i int
			for i = 0; i < len(rows); i++ {
				if rows[i].Category != "" && rows[i].Category != string(bybitspottypes.CategorySpot) {
					continue
				}
				handler(bybitspottypes.OrderInfo{
					OrderID:       rows[i].OrderID,
					ClientOrderID: rows[i].OrderLinkID,
					Symbol:        rows[i].Symbol,
					Side:          bybitspottypes.SideType(rows[i].Side),
					OrderType:     bybitspottypes.OrderType(rows[i].OrderType),
					TimeInForce:   bybitspottypes.TimeInForceType(rows[i].TimeInForce),
					Price:         dec(rows[i].Price),
					Quantity:      dec(rows[i].Qty),
					LeavesQty:     dec(rows[i].LeavesQty),
					CumExecQty:    dec(rows[i].CumExecQty),
					CumExecValue:  dec(rows[i].CumExecValue),
					AvgPrice:      dec(rows[i].AvgPrice),
					CumExecFee:    dec(rows[i].CumExecFee),
					Status:        bybitspottypes.OrderStatus(rows[i].OrderStatus),
					MarketUnit:    bybitspottypes.MarketUnit(rows[i].MarketUnit),
					IsLeverage:    rows[i].IsLeverage == "1",
					RejectReason:  normalizeRejectReason(rows[i].RejectReason),
					CreatedAtMs:   ms(rows[i].CreatedTime),
					UpdatedAtMs:   ms(rows[i].UpdatedTime),
				})
			}
		},
	}
	s.c.privateConn().Start(ctx)
	if err := s.c.privateConn().Subscribe(sub); err != nil {
		if errHandler != nil {
			errHandler(err)
		}
		return err
	}
	return nil
}

// =====================================================================
// PRIVATE: executions (own fills).
// =====================================================================

type privateExecutionEntry struct {
	Category    string `json:"category"`
	Symbol      string `json:"symbol"`
	OrderID     string `json:"orderId"`
	OrderLinkID string `json:"orderLinkId"`
	ExecID      string `json:"execId"`
	Side        string `json:"side"`
	ExecPrice   string `json:"execPrice"`
	ExecQty     string `json:"execQty"`
	ExecValue   string `json:"execValue"`
	ExecFee     string `json:"execFee"`
	FeeCurrency string `json:"feeCurrency"`
	IsMaker     bool   `json:"isMaker"`
	IsLeverage  string `json:"isLeverage"`
	ExecTime    string `json:"execTime"`
}

// WatchExecutions subscribes to the private "execution" topic.
func (s *StreamClient) WatchExecutions(
	ctx context.Context,
	handler func(bybitspottypes.ExecutionInfo),
	errHandler func(error),
) error {
	if !s.c.signerEnabled() {
		return bybit.NewError(bybit.ErrorKindAuth, "", "stream.WatchExecutions: APIKey/SecretKey not configured", nil)
	}
	var sub *ws.Subscription = &ws.Subscription{
		Topic: "execution",
		Handler: func(_, _ string, payload []byte) {
			var rows []privateExecutionEntry
			if err := codec.Unmarshal(payload, &rows); err != nil {
				s.c.logger().Warn("stream.WatchExecutions: parse", bybit.Err(err))
				return
			}
			var i int
			for i = 0; i < len(rows); i++ {
				if rows[i].Category != "" && rows[i].Category != string(bybitspottypes.CategorySpot) {
					continue
				}
				handler(bybitspottypes.ExecutionInfo{
					Symbol:        rows[i].Symbol,
					OrderID:       rows[i].OrderID,
					ClientOrderID: rows[i].OrderLinkID,
					ExecID:        rows[i].ExecID,
					Side:          bybitspottypes.SideType(rows[i].Side),
					ExecPrice:     dec(rows[i].ExecPrice),
					ExecQty:       dec(rows[i].ExecQty),
					ExecValue:     dec(rows[i].ExecValue),
					ExecFee:       dec(rows[i].ExecFee),
					FeeCurrency:   rows[i].FeeCurrency,
					IsMaker:       rows[i].IsMaker,
					IsLeverage:    rows[i].IsLeverage == "1",
					ExecTimeMs:    ms(rows[i].ExecTime),
				})
			}
		},
	}
	s.c.privateConn().Start(ctx)
	if err := s.c.privateConn().Subscribe(sub); err != nil {
		if errHandler != nil {
			errHandler(err)
		}
		return err
	}
	return nil
}

// =====================================================================
// PRIVATE: wallet.
// =====================================================================

// WatchWallet subscribes to the private "wallet" topic. The wire
// payload matches the REST wallet-balance response one-for-one — the
// converter in account.go is reused.
func (s *StreamClient) WatchWallet(
	ctx context.Context,
	handler func(bybitspottypes.Balance),
	errHandler func(error),
) error {
	if !s.c.signerEnabled() {
		return bybit.NewError(bybit.ErrorKindAuth, "", "stream.WatchWallet: APIKey/SecretKey not configured", nil)
	}
	var sub *ws.Subscription = &ws.Subscription{
		Topic: "wallet",
		Handler: func(_, _ string, payload []byte) {
			var rows []walletEntry
			if err := codec.Unmarshal(payload, &rows); err != nil {
				s.c.logger().Warn("stream.WatchWallet: parse", bybit.Err(err))
				return
			}
			var i int
			for i = 0; i < len(rows); i++ {
				handler(convertBalance(rows[i]))
			}
		},
	}
	s.c.privateConn().Start(ctx)
	if err := s.c.privateConn().Subscribe(sub); err != nil {
		if errHandler != nil {
			errHandler(err)
		}
		return err
	}
	return nil
}
