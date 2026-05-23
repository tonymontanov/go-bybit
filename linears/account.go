/*
FILE: linears/account.go

DESCRIPTION:
Account / position sub-client for the Bybit V5 linear category. Implements:

  - GetWalletBalance : GET  /v5/account/wallet-balance
  - GetPosition      : GET  /v5/position/list
  - GetOpenOrders    : GET  /v5/order/realtime           (paginated)
  - SetLeverage      : POST /v5/position/set-leverage
  - SetPositionMode  : POST /v5/position/switch-mode
  - ClosePosition    : Market + ReduceOnly order on the opposite side
                       (Bybit has no dedicated "close-position" endpoint).

BYBIT V5 SPECIFICS:
  - GetWalletBalance accepts an accountType (UNIFIED / CONTRACT). The SDK
    reads it from the request struct; the default is UNIFIED.
  - GetPosition supports paging via &cursor=... — for v1 we issue ONE page
    and return up to 200 rows (Bybit's max limit). The desk-side adapter
    typically queries a known symbol, so multi-page paging is unnecessary
    in practice.
  - GetOpenOrders DOES paginate transparently: we follow nextPageCursor
    until exhausted to deliver one consolidated slice. Used internally by
    Trading.CancelAllOrders / Trading.CancelForgottenOrders / SyncOrderMappings.
  - SetPositionMode requires either a symbol or a coin filter; the SDK
    accepts a single symbol for the linears profile.

ALL methods that hit private endpoints surface auth.ErrSignerDisabled
through the standard *bberr.Error envelope when API credentials are not
configured. Validation is performed locally first.
*/

package linears

import (
	"context"
	"net/url"
	"strconv"

	"github.com/shopspring/decimal"
	bybit "github.com/tonymontanov/go-bybit/v2"
	"github.com/tonymontanov/go-bybit/v2/internal/rest"
	"github.com/tonymontanov/go-bybit/v2/linears/types"
)

// AccountClient — account / position sub-client.
type AccountClient struct {
	c *Client
}

func newAccountClient(c *Client) *AccountClient {
	return &AccountClient{c: c}
}

// ---------------------------------------------------------------------
// Wallet balance.
// ---------------------------------------------------------------------

// WalletBalanceRequest is the input shape for GetWalletBalance.
//
// Coins is optional; passing it filters the per-coin breakdown returned
// by Bybit. AccountType defaults to UNIFIED.
type WalletBalanceRequest struct {
	AccountType types.AccountType
	Coins       []string
}

// GetWalletBalance returns wallet state for the requested account type.
// Bybit always returns ONE element in result.list per accountType; the
// SDK exposes that element as Balance directly.
func (a *AccountClient) GetWalletBalance(ctx context.Context, req WalletBalanceRequest) (types.Balance, error) {
	var out types.Balance
	if req.AccountType == "" {
		req.AccountType = types.AccountTypeUnified
	}
	var query url.Values = url.Values{}
	query.Set("accountType", string(req.AccountType))
	if len(req.Coins) > 0 {
		query.Set("coin", joinUpper(req.Coins))
	}

	var resp rest.Response
	var err error
	resp, _, err = a.c.rest().Do(ctx, rest.Options{
		Method: "GET",
		Path:   "/v5/account/wallet-balance",
		Query:  query,
		Signed: true,
		Meta: rest.RequestMeta{
			Category: string(bybit.RateLimitCategoryQuery),
		},
	})
	if err != nil {
		return out, err
	}

	var payload walletBalancePayload
	if err = resp.UnmarshalResult(&payload); err != nil {
		return out, bybit.NewError(bybit.ErrorKindUnknown, "", "account.GetWalletBalance: parse", err)
	}
	if len(payload.List) == 0 {
		return out, nil
	}
	return convertBalance(payload.List[0]), nil
}

// ---------------------------------------------------------------------
// Positions.
// ---------------------------------------------------------------------

// GetPosition returns the position rows reported by Bybit for the given
// symbol. One-way accounts return at most one row (PositionIdx=0); hedge
// accounts return up to two rows (PositionIdx=1 long, PositionIdx=2 short).
//
// Bybit reports a row even when the position is empty (Side=="", Size=0);
// callers can use PositionInfo.IsEmpty() to skip those.
func (a *AccountClient) GetPosition(ctx context.Context, symbol string) ([]types.PositionInfo, error) {
	if symbol == "" {
		return nil, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "account.GetPosition: symbol is empty", nil)
	}
	var query url.Values = url.Values{}
	query.Set("category", string(types.CategoryLinear))
	query.Set("symbol", symbol)

	var resp rest.Response
	var err error
	resp, _, err = a.c.rest().Do(ctx, rest.Options{
		Method: "GET",
		Path:   "/v5/position/list",
		Query:  query,
		Signed: true,
		Meta: rest.RequestMeta{
			Symbols:  []string{symbol},
			Category: string(bybit.RateLimitCategoryQuery),
		},
	})
	if err != nil {
		return nil, err
	}

	var payload positionListPayload
	if err = resp.UnmarshalResult(&payload); err != nil {
		return nil, bybit.NewError(bybit.ErrorKindUnknown, "", "account.GetPosition: parse", err)
	}
	var out []types.PositionInfo = make([]types.PositionInfo, 0, len(payload.List))
	var i int
	for i = 0; i < len(payload.List); i++ {
		out = append(out, convertPosition(payload.List[i]))
	}
	return out, nil
}

// ---------------------------------------------------------------------
// Open orders.
// ---------------------------------------------------------------------

// GetOpenOrders returns ALL open orders for the symbol. Pagination is
// handled internally — the caller receives one consolidated slice.
//
// Implementation note: Bybit V5 paginates with a "nextPageCursor" string.
// An empty cursor in the response (or a fully-served page < limit)
// terminates the loop. limit is fixed at 50 — small enough to avoid
// large copies but large enough to keep the page count low for typical
// market-making workloads.
func (a *AccountClient) GetOpenOrders(ctx context.Context, symbol string) ([]types.OrderInfo, error) {
	if symbol == "" {
		return nil, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "account.GetOpenOrders: symbol is empty", nil)
	}
	const pageLimit = 50
	var out []types.OrderInfo
	var cursor string

	for {
		var query url.Values = url.Values{}
		query.Set("category", string(types.CategoryLinear))
		query.Set("symbol", symbol)
		query.Set("limit", strconv.Itoa(pageLimit))
		if cursor != "" {
			query.Set("cursor", cursor)
		}

		var resp rest.Response
		var err error
		resp, _, err = a.c.rest().Do(ctx, rest.Options{
			Method: "GET",
			Path:   "/v5/order/realtime",
			Query:  query,
			Signed: true,
			Meta: rest.RequestMeta{
				Symbols:  []string{symbol},
				Category: string(bybit.RateLimitCategoryQuery),
			},
		})
		if err != nil {
			return out, err
		}

		var payload orderRealtimePayload
		if err = resp.UnmarshalResult(&payload); err != nil {
			return out, bybit.NewError(bybit.ErrorKindUnknown, "", "account.GetOpenOrders: parse", err)
		}
		var i int
		for i = 0; i < len(payload.List); i++ {
			out = append(out, convertOrderInfo(payload.List[i]))
		}
		if payload.NextPageCursor == "" || len(payload.List) < pageLimit {
			break
		}
		cursor = payload.NextPageCursor
	}
	return out, nil
}

// ---------------------------------------------------------------------
// Position configuration.
// ---------------------------------------------------------------------

// SetLeverage updates per-symbol leverage. For one-way accounts both
// buy and sell leverage are usually identical; the SDK accepts ONE
// leverage and writes it to both sides.
//
// PARAMETERS:
//   - symbol: target symbol.
//   - leverage: leverage as a decimal (e.g. 5 means "5x"). Must be > 0
//     and within the symbol's min/max from SymbolInfo.
func (a *AccountClient) SetLeverage(ctx context.Context, symbol string, leverage decimal.Decimal) error {
	if symbol == "" {
		return bybit.NewError(bybit.ErrorKindInvalidRequest, "", "account.SetLeverage: symbol is empty", nil)
	}
	if leverage.IsZero() || leverage.IsNegative() {
		return bybit.NewError(bybit.ErrorKindInvalidRequest, "", "account.SetLeverage: leverage must be positive", nil)
	}
	var s string = leverage.String()
	var body = map[string]any{
		"category":     string(types.CategoryLinear),
		"symbol":       symbol,
		"buyLeverage":  s,
		"sellLeverage": s,
	}
	var resp rest.Response
	var err error
	resp, _, err = a.c.rest().Do(ctx, rest.Options{
		Method: "POST",
		Path:   "/v5/position/set-leverage",
		Body:   body,
		Signed: true,
		Meta: rest.RequestMeta{
			Symbols:  []string{symbol},
			Category: string(bybit.RateLimitCategoryQuery),
		},
	})
	_ = resp
	return err
}

// SetPositionMode switches between one-way and hedge mode for the symbol.
// Bybit only accepts the change when there are no open positions / orders
// on that symbol — otherwise retCode 110017 / 110024 is surfaced.
func (a *AccountClient) SetPositionMode(ctx context.Context, symbol string, mode types.PositionMode) error {
	if symbol == "" {
		return bybit.NewError(bybit.ErrorKindInvalidRequest, "", "account.SetPositionMode: symbol is empty", nil)
	}
	if mode != types.PositionModeOneWay && mode != types.PositionModeHedge {
		return bybit.NewError(bybit.ErrorKindInvalidRequest, "", "account.SetPositionMode: mode must be PositionModeOneWay (0) or PositionModeHedge (3)", nil)
	}
	var body = map[string]any{
		"category": string(types.CategoryLinear),
		"symbol":   symbol,
		"mode":     int(mode),
	}
	var resp rest.Response
	var err error
	resp, _, err = a.c.rest().Do(ctx, rest.Options{
		Method: "POST",
		Path:   "/v5/position/switch-mode",
		Body:   body,
		Signed: true,
		Meta: rest.RequestMeta{
			Symbols:  []string{symbol},
			Category: string(bybit.RateLimitCategoryQuery),
		},
	})
	_ = resp
	return err
}

// ---------------------------------------------------------------------
// ClosePosition.
// ---------------------------------------------------------------------

// ClosePosition closes an open position on `symbol` by sending a Market
// order with reduceOnly=true on the OPPOSITE side, sized to the current
// position quantity. If the account has no open position on the symbol
// (Quantity==0), nothing is sent and (zero OrderInfo, nil) is returned.
//
// For hedge accounts the caller MUST specify which leg to close via
// positionIdx — otherwise both rows (long+short) are inspected and the
// first non-empty one is closed; this matches the Bybit "Close Position"
// button behaviour but is rarely what a programmatic caller wants.
//
// Returned OrderInfo carries the Market order's identifiers; the actual
// fill arrives via the WS execution channel (M3) or by polling
// GetOpenOrders.
func (a *AccountClient) ClosePosition(ctx context.Context, symbol string, positionIdx types.PositionIdx) (types.OrderInfo, error) {
	var info types.OrderInfo
	if symbol == "" {
		return info, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "account.ClosePosition: symbol is empty", nil)
	}
	var positions []types.PositionInfo
	var err error
	positions, err = a.GetPosition(ctx, symbol)
	if err != nil {
		return info, err
	}

	var target *types.PositionInfo
	var i int
	for i = 0; i < len(positions); i++ {
		if positions[i].Quantity.IsZero() {
			continue
		}
		if positionIdx != types.PositionIdxOneWay && positions[i].PositionIdx != positionIdx {
			continue
		}
		var copy types.PositionInfo = positions[i]
		target = &copy
		break
	}
	if target == nil {
		return info, nil
	}

	var oppositeSide types.SideType
	switch target.Side {
	case types.SideTypeBuy:
		oppositeSide = types.SideTypeSell
	case types.SideTypeSell:
		oppositeSide = types.SideTypeBuy
	default:
		return info, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "account.ClosePosition: position has unknown side", nil)
	}

	return a.c.Trading().CreateOrder(ctx, types.CreateOrderRequest{
		Symbol:      symbol,
		Side:        oppositeSide,
		OrderType:   types.OrderTypeMarket,
		Quantity:    target.Quantity,
		PositionIdx: target.PositionIdx,
		ReduceOnly:  true,
	})
}

// ---------------------------------------------------------------------
// Wire payloads + converters.
// ---------------------------------------------------------------------

type walletBalancePayload struct {
	List []walletEntry `json:"list"`
}

type walletEntry struct {
	AccountType            string            `json:"accountType"`
	TotalEquity            string            `json:"totalEquity"`
	TotalWalletBalance     string            `json:"totalWalletBalance"`
	TotalAvailableBalance  string            `json:"totalAvailableBalance"`
	TotalMarginBalance     string            `json:"totalMarginBalance"`
	TotalInitialMargin     string            `json:"totalInitialMargin"`
	TotalMaintenanceMargin string            `json:"totalMaintenanceMargin"`
	TotalPerpUPL           string            `json:"totalPerpUPL"`
	AccountIMRate          string            `json:"accountIMRate"`
	AccountMMRate          string            `json:"accountMMRate"`
	AccountLTV             string            `json:"accountLTV"`
	Coin                   []walletCoinEntry `json:"coin"`
}

type walletCoinEntry struct {
	Coin                string `json:"coin"`
	Equity              string `json:"equity"`
	WalletBalance       string `json:"walletBalance"`
	UsdValue            string `json:"usdValue"`
	UnrealisedPnl       string `json:"unrealisedPnl"`
	CumRealisedPnl      string `json:"cumRealisedPnl"`
	BorrowAmount        string `json:"borrowAmount"`
	AvailableToWithdraw string `json:"availableToWithdraw"`
	AvailableToBorrow   string `json:"availableToBorrow"`
	Locked              string `json:"locked"`
	TotalOrderIM        string `json:"totalOrderIM"`
	TotalPositionIM     string `json:"totalPositionIM"`
	TotalPositionMM     string `json:"totalPositionMM"`
	AccruedInterest     string `json:"accruedInterest"`
	SpotHedgingQty      string `json:"spotHedgingQty"`
	MarginCollateral    bool   `json:"marginCollateral"`
	CollateralSwitch    bool   `json:"collateralSwitch"`
}

func convertBalance(w walletEntry) types.Balance {
	var coins []types.CoinBalance = make([]types.CoinBalance, 0, len(w.Coin))
	var i int
	for i = 0; i < len(w.Coin); i++ {
		var src walletCoinEntry = w.Coin[i]
		coins = append(coins, types.CoinBalance{
			Coin:                src.Coin,
			Equity:              dec(src.Equity),
			WalletBalance:       dec(src.WalletBalance),
			UsdValue:            dec(src.UsdValue),
			UnrealizedPnL:       dec(src.UnrealisedPnl),
			CumRealizedPnL:      dec(src.CumRealisedPnl),
			BorrowAmount:        dec(src.BorrowAmount),
			AvailableToWithdraw: dec(src.AvailableToWithdraw),
			AvailableToBorrow:   dec(src.AvailableToBorrow),
			Locked:              dec(src.Locked),
			TotalOrderIM:        dec(src.TotalOrderIM),
			TotalPositionIM:     dec(src.TotalPositionIM),
			TotalPositionMM:     dec(src.TotalPositionMM),
			AccruedInterest:     dec(src.AccruedInterest),
			SpotHedgingQty:      dec(src.SpotHedgingQty),
			MarginCollateral:    src.MarginCollateral,
			CollateralSwitch:    src.CollateralSwitch,
		})
	}
	return types.Balance{
		AccountType:            types.AccountType(w.AccountType),
		TotalEquity:            dec(w.TotalEquity),
		TotalWalletBalance:     dec(w.TotalWalletBalance),
		TotalAvailableBalance:  dec(w.TotalAvailableBalance),
		TotalMarginBalance:     dec(w.TotalMarginBalance),
		TotalInitialMargin:     dec(w.TotalInitialMargin),
		TotalMaintenanceMargin: dec(w.TotalMaintenanceMargin),
		TotalPerpUPL:           dec(w.TotalPerpUPL),
		AccountIMRate:          dec(w.AccountIMRate),
		AccountMMRate:          dec(w.AccountMMRate),
		AccountLTV:             dec(w.AccountLTV),
		Coins:                  coins,
	}
}

type positionListPayload struct {
	Category       string          `json:"category"`
	List           []positionEntry `json:"list"`
	NextPageCursor string          `json:"nextPageCursor"`
}

type positionEntry struct {
	PositionIdx    int    `json:"positionIdx"`
	Symbol         string `json:"symbol"`
	Side           string `json:"side"`
	Size           string `json:"size"`
	AvgPrice       string `json:"avgPrice"`
	MarkPrice      string `json:"markPrice"`
	LiqPrice       string `json:"liqPrice"`
	Leverage       string `json:"leverage"`
	UnrealisedPnl  string `json:"unrealisedPnl"`
	CumRealisedPnl string `json:"cumRealisedPnl"`
	PositionValue  string `json:"positionValue"`
	UpdatedTime    string `json:"updatedTime"`
}

func convertPosition(src positionEntry) types.PositionInfo {
	return types.PositionInfo{
		Symbol:        src.Symbol,
		Side:          types.SideType(src.Side),
		PositionIdx:   types.PositionIdx(src.PositionIdx),
		Quantity:      dec(src.Size),
		AvgEntryPrice: dec(src.AvgPrice),
		MarkPrice:     dec(src.MarkPrice),
		LiqPrice:      dec(src.LiqPrice),
		Leverage:      dec(src.Leverage),
		UnrealizedPnL: dec(src.UnrealisedPnl),
		RealizedPnL:   dec(src.CumRealisedPnl),
		PositionValue: dec(src.PositionValue),
		UpdatedAtMs:   ms(src.UpdatedTime),
	}
}

type orderRealtimePayload struct {
	Category       string       `json:"category"`
	List           []orderEntry `json:"list"`
	NextPageCursor string       `json:"nextPageCursor"`
}

type orderEntry struct {
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
	AvgPrice     string `json:"avgPrice"`
	CumExecFee   string `json:"cumExecFee"`
	OrderStatus  string `json:"orderStatus"`
	PositionIdx  int    `json:"positionIdx"`
	ReduceOnly   bool   `json:"reduceOnly"`
	RejectReason string `json:"rejectReason"`
	CreatedTime  string `json:"createdTime"`
	UpdatedTime  string `json:"updatedTime"`
}

func convertOrderInfo(src orderEntry) types.OrderInfo {
	return types.OrderInfo{
		OrderID:       src.OrderID,
		ClientOrderID: src.OrderLinkID,
		Symbol:        src.Symbol,
		Side:          types.SideType(src.Side),
		OrderType:     types.OrderType(src.OrderType),
		TimeInForce:   types.TimeInForceType(src.TimeInForce),
		Price:         dec(src.Price),
		Quantity:      dec(src.Qty),
		LeavesQty:     dec(src.LeavesQty),
		CumExecQty:    dec(src.CumExecQty),
		AvgPrice:      dec(src.AvgPrice),
		CumExecFee:    dec(src.CumExecFee),
		Status:        types.OrderStatus(src.OrderStatus),
		PositionIdx:   types.PositionIdx(src.PositionIdx),
		ReduceOnly:    src.ReduceOnly,
		RejectReason:  normalizeRejectReason(src.RejectReason),
		CreatedAtMs:   ms(src.CreatedTime),
		UpdatedAtMs:   ms(src.UpdatedTime),
	}
}

// joinUpper concatenates coin tickers with a comma. Bybit accepts coins
// in upper-case; we normalise here so callers can pass either case.
func joinUpper(coins []string) string {
	var out []byte
	var i int
	for i = 0; i < len(coins); i++ {
		if i > 0 {
			out = append(out, ',')
		}
		var s string = coins[i]
		var j int
		for j = 0; j < len(s); j++ {
			var c byte = s[j]
			if c >= 'a' && c <= 'z' {
				c -= 'a' - 'A'
			}
			out = append(out, c)
		}
	}
	return string(out)
}

// ensureSigned mirrors the trading.ensureSigned helper. Wired into M3
// pre-checks; kept silent here so unused-symbol warnings do not break
// the build.
func (a *AccountClient) ensureSigned() error {
	if !a.c.signerEnabled() {
		return bybit.NewError(bybit.ErrorKindAuth, "", "account: APIKey/SecretKey not configured", nil)
	}
	return nil
}

// _ — suppress unused warnings until M3 consumes the helper everywhere.
var _ = (*AccountClient).ensureSigned
