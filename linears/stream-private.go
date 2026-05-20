/*
FILE: linears/stream-private.go

DESCRIPTION:
Private WebSocket subscriptions — own orders, positions, executions and
wallet state. All methods route through the StreamClient on the parent
linears.Client.

PRIVATE TOPICS (Bybit V5):
  - "order"      — own order updates across categories. The dispatcher
                   filters by category=="linear" before invoking the
                   handler so the linears stream stays focused.
  - "position"   — own position updates (linear only).
  - "execution"  — own fills (linear only).
  - "wallet"     — wallet state (account-wide; no per-category filter).

REQUIRES API CREDENTIALS:
The private endpoint mandates a successful auth handshake. If
Config.APIKey / Config.SecretKey are not set, the underlying ws.Conn
fails the handshake and reconnect storms; Watch* methods short-circuit
locally with ErrorKindAuth before opening the socket.
*/

package linears

import (
	"context"

	bybit "github.com/tonymontanov/go-bybit"
	"github.com/tonymontanov/go-bybit/internal/codec"
	"github.com/tonymontanov/go-bybit/internal/ws"
	"github.com/tonymontanov/go-bybit/linears/types"
)

// =====================================================================
// PRIVATE: own orders.
// =====================================================================

// privateOrderEntry — wire shape of one element of the "order" topic.
// Identical to orderEntry in account.go but kept separate to avoid
// coupling the private stream parser to the REST parser (Bybit may
// independently extend either).
type privateOrderEntry struct {
	Category      string `json:"category"`
	OrderID       string `json:"orderId"`
	OrderLinkID   string `json:"orderLinkId"`
	Symbol        string `json:"symbol"`
	Side          string `json:"side"`
	OrderType     string `json:"orderType"`
	TimeInForce   string `json:"timeInForce"`
	Price         string `json:"price"`
	Qty           string `json:"qty"`
	LeavesQty     string `json:"leavesQty"`
	CumExecQty    string `json:"cumExecQty"`
	AvgPrice      string `json:"avgPrice"`
	CumExecFee    string `json:"cumExecFee"`
	OrderStatus   string `json:"orderStatus"`
	PositionIdx   int    `json:"positionIdx"`
	ReduceOnly    bool   `json:"reduceOnly"`
	RejectReason  string `json:"rejectReason"`
	CreatedTime   string `json:"createdTime"`
	UpdatedTime   string `json:"updatedTime"`
}

// WatchOrders subscribes to the private "order" topic. The handler is
// invoked once per order update; rows for non-linear categories are
// filtered out so callers do not see spot/option crosstalk.
func (s *StreamClient) WatchOrders(
	ctx context.Context,
	handler func(types.OrderInfo),
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
				if rows[i].Category != "" && rows[i].Category != string(types.CategoryLinear) {
					continue
				}
				handler(types.OrderInfo{
					OrderID:       rows[i].OrderID,
					ClientOrderID: rows[i].OrderLinkID,
					Symbol:        rows[i].Symbol,
					Side:          types.SideType(rows[i].Side),
					OrderType:     types.OrderType(rows[i].OrderType),
					TimeInForce:   types.TimeInForceType(rows[i].TimeInForce),
					Price:         dec(rows[i].Price),
					Quantity:      dec(rows[i].Qty),
					LeavesQty:     dec(rows[i].LeavesQty),
					CumExecQty:    dec(rows[i].CumExecQty),
					AvgPrice:      dec(rows[i].AvgPrice),
					CumExecFee:    dec(rows[i].CumExecFee),
					Status:        types.OrderStatus(rows[i].OrderStatus),
					PositionIdx:   types.PositionIdx(rows[i].PositionIdx),
					ReduceOnly:    rows[i].ReduceOnly,
					RejectReason:  rows[i].RejectReason,
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
// PRIVATE: own positions.
// =====================================================================

type privatePositionEntry struct {
	Category       string `json:"category"`
	Symbol         string `json:"symbol"`
	Side           string `json:"side"`
	PositionIdx    int    `json:"positionIdx"`
	Size           string `json:"size"`
	AvgPrice       string `json:"entryPrice"`
	MarkPrice      string `json:"markPrice"`
	LiqPrice       string `json:"liqPrice"`
	Leverage       string `json:"leverage"`
	UnrealisedPnl  string `json:"unrealisedPnl"`
	CumRealisedPnl string `json:"cumRealisedPnl"`
	PositionValue  string `json:"positionValue"`
	UpdatedTime    string `json:"updatedTime"`
}

// WatchPositions subscribes to the private "position" topic. Linear-only
// rows reach the handler.
//
// Note: Bybit V5 sends a row even after a full close (Side="" Size=0).
// PositionInfo.IsEmpty() helps callers skip those.
func (s *StreamClient) WatchPositions(
	ctx context.Context,
	handler func(types.PositionInfo),
	errHandler func(error),
) error {
	if !s.c.signerEnabled() {
		return bybit.NewError(bybit.ErrorKindAuth, "", "stream.WatchPositions: APIKey/SecretKey not configured", nil)
	}
	var sub *ws.Subscription = &ws.Subscription{
		Topic: "position",
		Handler: func(_, _ string, payload []byte) {
			var rows []privatePositionEntry
			if err := codec.Unmarshal(payload, &rows); err != nil {
				s.c.logger().Warn("stream.WatchPositions: parse", bybit.Err(err))
				return
			}
			var i int
			for i = 0; i < len(rows); i++ {
				if rows[i].Category != "" && rows[i].Category != string(types.CategoryLinear) {
					continue
				}
				handler(types.PositionInfo{
					Symbol:        rows[i].Symbol,
					Side:          types.SideType(rows[i].Side),
					PositionIdx:   types.PositionIdx(rows[i].PositionIdx),
					Quantity:      dec(rows[i].Size),
					AvgEntryPrice: dec(rows[i].AvgPrice),
					MarkPrice:     dec(rows[i].MarkPrice),
					LiqPrice:      dec(rows[i].LiqPrice),
					Leverage:      dec(rows[i].Leverage),
					UnrealizedPnL: dec(rows[i].UnrealisedPnl),
					RealizedPnL:   dec(rows[i].CumRealisedPnl),
					PositionValue: dec(rows[i].PositionValue),
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
	PositionIdx int    `json:"positionIdx"`
	ExecTime    string `json:"execTime"`
}

// WatchExecutions subscribes to the private "execution" topic.
func (s *StreamClient) WatchExecutions(
	ctx context.Context,
	handler func(types.ExecutionInfo),
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
				if rows[i].Category != "" && rows[i].Category != string(types.CategoryLinear) {
					continue
				}
				handler(types.ExecutionInfo{
					Symbol:        rows[i].Symbol,
					OrderID:       rows[i].OrderID,
					ClientOrderID: rows[i].OrderLinkID,
					ExecID:        rows[i].ExecID,
					Side:          types.SideType(rows[i].Side),
					ExecPrice:     dec(rows[i].ExecPrice),
					ExecQty:       dec(rows[i].ExecQty),
					ExecValue:     dec(rows[i].ExecValue),
					ExecFee:       dec(rows[i].ExecFee),
					FeeCurrency:   rows[i].FeeCurrency,
					IsMaker:       rows[i].IsMaker,
					PositionIdx:   types.PositionIdx(rows[i].PositionIdx),
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

// WatchWallet subscribes to the private "wallet" topic. The wire payload
// matches the REST wallet-balance response one-for-one — the converter
// in account.go is reused.
func (s *StreamClient) WatchWallet(
	ctx context.Context,
	handler func(types.Balance),
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
