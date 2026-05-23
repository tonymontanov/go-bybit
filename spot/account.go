/*
FILE: spot/account.go

DESCRIPTION:
Authenticated account / wallet sub-client for the Bybit V5 spot
category. Implements:

  - GetWalletBalance — GET /v5/account/wallet-balance
  - GetOpenOrders    — GET /v5/order/realtime?category=spot

DIFFERENCES vs LINEARS:
  - No GetPosition / SetLeverage / SetPositionMode / ClosePosition —
    spot has no positions or leverage knobs.
  - GetWalletBalance accepts UNIFIED or SPOT (not CONTRACT). UNIFIED is
    the default and required for UTA accounts.
  - GetOpenOrders queries category=spot and decodes spot-specific
    fields (no positionIdx, plus marketUnit / isLeverage echo).

NOTE on pagination:
  GetOpenOrders follows nextPageCursor transparently. Used internally
  by Trading.CancelForgottenOrders / SyncOrderMappings.
*/

package spot

import (
	"context"
	"net/url"
	"strconv"

	bybit "github.com/tonymontanov/go-bybit"
	"github.com/tonymontanov/go-bybit/internal/rest"
	bybitspottypes "github.com/tonymontanov/go-bybit/spot/types"
)

// AccountClient — authenticated spot account sub-client.
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
// Coins is optional; passing it filters the per-coin breakdown.
// AccountType defaults to UNIFIED.
type WalletBalanceRequest struct {
	AccountType bybitspottypes.AccountType
	Coins       []string
}

// GetWalletBalance returns wallet state for the requested account type.
// Bybit always returns ONE element in result.list per accountType.
//
// For backwards compatibility this method also accepts the simple
// `(ctx, accountType)` form via WalletBalanceSimple — see below.
func (a *AccountClient) GetWalletBalance(ctx context.Context, req WalletBalanceRequest) (bybitspottypes.Balance, error) {
	var out bybitspottypes.Balance
	if req.AccountType == "" {
		req.AccountType = bybitspottypes.AccountTypeUnified
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
// Open orders.
// ---------------------------------------------------------------------

// GetOpenOrders returns ALL open orders for the spot symbol. Pagination
// is handled internally — the caller receives one consolidated slice.
//
// Implementation note: Bybit V5 paginates with a "nextPageCursor"
// string. An empty cursor in the response (or a fully-served page <
// limit) terminates the loop. limit is fixed at 50 — small enough to
// avoid large copies, large enough to keep the page count low.
func (a *AccountClient) GetOpenOrders(ctx context.Context, symbol string) ([]bybitspottypes.OrderInfo, error) {
	if symbol == "" {
		return nil, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "account.GetOpenOrders: symbol is empty", nil)
	}
	const pageLimit = 50
	var out []bybitspottypes.OrderInfo
	var cursor string

	for {
		var query url.Values = url.Values{}
		query.Set("category", string(bybitspottypes.CategorySpot))
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

func convertBalance(w walletEntry) bybitspottypes.Balance {
	var coins []bybitspottypes.CoinBalance = make([]bybitspottypes.CoinBalance, 0, len(w.Coin))
	var i int
	for i = 0; i < len(w.Coin); i++ {
		var src walletCoinEntry = w.Coin[i]
		coins = append(coins, bybitspottypes.CoinBalance{
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
	return bybitspottypes.Balance{
		AccountType:            bybitspottypes.AccountType(w.AccountType),
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

type orderRealtimePayload struct {
	Category       string       `json:"category"`
	List           []orderEntry `json:"list"`
	NextPageCursor string       `json:"nextPageCursor"`
}

// orderEntry mirrors Bybit's /v5/order/realtime row for category=spot.
// Fields specific to derivatives (positionIdx, reduceOnly) are absent
// or always 0/false on spot — kept out of the struct entirely.
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

func convertOrderInfo(src orderEntry) bybitspottypes.OrderInfo {
	return bybitspottypes.OrderInfo{
		OrderID:       src.OrderID,
		ClientOrderID: src.OrderLinkID,
		Symbol:        src.Symbol,
		Side:          bybitspottypes.SideType(src.Side),
		OrderType:     bybitspottypes.OrderType(src.OrderType),
		TimeInForce:   bybitspottypes.TimeInForceType(src.TimeInForce),
		Price:         dec(src.Price),
		Quantity:      dec(src.Qty),
		LeavesQty:     dec(src.LeavesQty),
		CumExecQty:    dec(src.CumExecQty),
		CumExecValue:  dec(src.CumExecValue),
		AvgPrice:      dec(src.AvgPrice),
		CumExecFee:    dec(src.CumExecFee),
		Status:        bybitspottypes.OrderStatus(src.OrderStatus),
		MarketUnit:    bybitspottypes.MarketUnit(src.MarketUnit),
		IsLeverage:    src.IsLeverage == "1",
		RejectReason:  normalizeRejectReason(src.RejectReason),
		CreatedAtMs:   ms(src.CreatedTime),
		UpdatedAtMs:   ms(src.UpdatedTime),
	}
}

// joinUpper concatenates coin tickers with a comma, normalising case
// to upper. Bybit accepts coins in upper-case; we normalise to keep
// the API forgiving on caller input.
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

// ensureSigned surfaces a clear typed error when the SDK is configured
// without API credentials.
func (a *AccountClient) ensureSigned() error {
	if !a.c.signerEnabled() {
		return bybit.NewError(bybit.ErrorKindAuth, "", "account: APIKey/SecretKey not configured", nil)
	}
	return nil
}

// _ — keep ensureSigned referenced; consumed by a follow-up patch.
var _ = (*AccountClient).ensureSigned
