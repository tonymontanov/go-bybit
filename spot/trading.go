/*
FILE: spot/trading.go

DESCRIPTION:
Trading sub-client for the Bybit V5 spot category. Implements:

  - CreateOrder         — POST /v5/order/create        (category=spot)
  - ModifyOrder         — POST /v5/order/amend         (category=spot)
  - CancelOrder         — POST /v5/order/cancel        (category=spot)
  - CreateBatchOrders   — POST /v5/order/create-batch  (category=spot, ≤ 10 per call, UTA only)
  - ModifyBatchOrders   — POST /v5/order/amend-batch   (category=spot, UTA only)
  - CancelBatchOrders   — POST /v5/order/cancel-batch  (category=spot, UTA only)
  - CancelAllOrders     — POST /v5/order/cancel-all    (category=spot, one symbol)
  - CancelForgottenOrders — age-based cancel via GetOpenOrders + cancel-batch.

BYBIT V5 SPECIFICS (spot):
  - Batch endpoints REQUIRE the Unified Trading Account; classic spot
    accounts get retCode 10005.
  - Spot batch limit is 10 orders per call (linear is 20).
  - Market-order Quantity interpretation depends on (Side, MarketUnit) —
    see spot/types.MarketUnit. The SDK forwards MarketUnit verbatim.
  - timeInForce=PostOnly is honoured by Bybit V5 spot only when
    orderType=Limit; the SDK enforces that pairing.
  - orderLinkId pattern: 1..36 chars [A-Za-z0-9_.-] (same as linears).
  - retCode 10001 ("order not modified") on amend / amend-batch is
    treated as an idempotent SUCCESS — symmetric to linears.

INTERNAL STATE:
  - clOrdToOrd / ordToClOrd / createdAtMs: ID mapping caches, mirroring
    the linears profile. Removed synchronously on cancel.
*/

package spot

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"sync"
	"time"

	bybit "github.com/tonymontanov/go-bybit/v2"
	"github.com/tonymontanov/go-bybit/v2/internal/codec"
	"github.com/tonymontanov/go-bybit/v2/internal/rest"
	bybitspottypes "github.com/tonymontanov/go-bybit/v2/spot/types"
)

// MaxBatchSize — Bybit V5 limit on batch trade endpoints for the spot
// category. Linear and option cap higher; spot is the strictest.
const MaxBatchSize = 10

// retCodeOrderNotModified is Bybit's "no-op amend" sentinel; treated
// here as an idempotent SUCCESS (symmetric to linears).
const retCodeOrderNotModified = "10001"

// orderLinkIDPattern — allowed characters and length for orderLinkId.
// Bybit V5 docs §Trade > Place Order: max 36 chars, [A-Za-z0-9_.-].
var orderLinkIDPattern = regexp.MustCompile(`^[A-Za-z0-9_.\-]{1,36}$`)

// TradingClient — authenticated spot trading sub-client.
type TradingClient struct {
	c *Client

	mu          sync.RWMutex
	clOrdToOrd  map[string]string
	ordToClOrd  map[string]string
	createdAtMs map[string]int64
}

func newTradingClient(c *Client) *TradingClient {
	return &TradingClient{
		c:           c,
		clOrdToOrd:  make(map[string]string, 1024),
		ordToClOrd:  make(map[string]string, 1024),
		createdAtMs: make(map[string]int64, 1024),
	}
}

// ---------------------------------------------------------------------
// Single-order endpoints.
// ---------------------------------------------------------------------

// CreateOrder places a single spot order.
func (t *TradingClient) CreateOrder(ctx context.Context, req bybitspottypes.CreateOrderRequest) (bybitspottypes.OrderInfo, error) {
	var info bybitspottypes.OrderInfo

	var body map[string]any
	var err error
	body, err = t.buildCreateOrderBody(req)
	if err != nil {
		return info, err
	}

	var resp rest.Response
	var rateLimits map[string]string
	resp, rateLimits, err = t.c.rest().Do(ctx, rest.Options{
		Method: "POST",
		Path:   "/v5/order/create",
		Body:   body,
		Signed: true,
		Meta: rest.RequestMeta{
			OrderCount: 1,
			Symbols:    []string{req.Symbol},
			Category:   string(bybit.RateLimitCategoryPlace),
		},
	})
	if err != nil {
		return info, err
	}

	var ack orderAckPayload
	if err = resp.UnmarshalResult(&ack); err != nil {
		return info, bybit.NewError(bybit.ErrorKindUnknown, "", "trading.CreateOrder: parse", err)
	}

	var now int64 = time.Now().UnixMilli()
	info = bybitspottypes.OrderInfo{
		OrderID:       ack.OrderID,
		ClientOrderID: ack.OrderLinkID,
		Symbol:        req.Symbol,
		Side:          req.Side,
		OrderType:     resolveOrderType(req),
		TimeInForce:   resolveTimeInForce(req),
		Price:         req.Price,
		Quantity:      req.Quantity,
		LeavesQty:     req.Quantity,
		Status:        bybitspottypes.OrderStatusNew,
		MarketUnit:    req.MarketUnit,
		IsLeverage:    req.IsLeverage,
		CreatedAtMs:   now,
		UpdatedAtMs:   now,
		RateLimits:    rateLimits,
	}
	t.rememberMapping(ack.OrderLinkID, ack.OrderID, now)
	return info, nil
}

// ModifyOrder amends a single spot order. retCode 10001 ("order not
// modified") is treated as an idempotent SUCCESS — symmetric to linears.
func (t *TradingClient) ModifyOrder(ctx context.Context, req bybitspottypes.ModifyOrderRequest) (bybitspottypes.OrderInfo, error) {
	var info bybitspottypes.OrderInfo

	var body map[string]any
	var err error
	body, err = buildModifyOrderBody(req)
	if err != nil {
		return info, err
	}

	var resp rest.Response
	var rateLimits map[string]string
	resp, rateLimits, err = t.c.rest().Do(ctx, rest.Options{
		Method: "POST",
		Path:   "/v5/order/amend",
		Body:   body,
		Signed: true,
		Meta: rest.RequestMeta{
			OrderCount: 1,
			Symbols:    []string{req.Symbol},
			Category:   string(bybit.RateLimitCategoryAmend),
		},
	})
	// Idempotent path: retCode=10001 → not an error, just nothing to do.
	if err != nil {
		var bbErr *bybit.Error
		if errors.As(err, &bbErr) && bbErr.BybitCode == retCodeOrderNotModified {
			info = bybitspottypes.OrderInfo{
				Symbol:        req.Symbol,
				OrderID:       req.OrderID,
				ClientOrderID: req.ClientOrderID,
				Price:         req.NewPrice,
				Quantity:      req.NewQuantity,
				LeavesQty:     req.NewQuantity,
				UpdatedAtMs:   time.Now().UnixMilli(),
			}
			return info, nil
		}
		return info, err
	}

	var ack orderAckPayload
	if err = resp.UnmarshalResult(&ack); err != nil {
		return info, bybit.NewError(bybit.ErrorKindUnknown, "", "trading.ModifyOrder: parse", err)
	}
	info = bybitspottypes.OrderInfo{
		OrderID:       ack.OrderID,
		ClientOrderID: ack.OrderLinkID,
		Symbol:        req.Symbol,
		Price:         req.NewPrice,
		Quantity:      req.NewQuantity,
		LeavesQty:     req.NewQuantity,
		UpdatedAtMs:   time.Now().UnixMilli(),
		RateLimits:    rateLimits,
	}
	return info, nil
}

// CancelOrder cancels a single spot order. Exactly one of OrderID /
// ClientOrderID must be set.
func (t *TradingClient) CancelOrder(ctx context.Context, req bybitspottypes.CancelOrderRequest) error {
	var body map[string]any
	var err error
	body, err = buildCancelOrderBody(req)
	if err != nil {
		return err
	}

	var resp rest.Response
	resp, _, err = t.c.rest().Do(ctx, rest.Options{
		Method: "POST",
		Path:   "/v5/order/cancel",
		Body:   body,
		Signed: true,
		Meta: rest.RequestMeta{
			OrderCount: 1,
			Symbols:    []string{req.Symbol},
			Category:   string(bybit.RateLimitCategoryCancel),
		},
	})
	if err != nil {
		return err
	}

	var ack orderAckPayload
	if err = resp.UnmarshalResult(&ack); err != nil {
		return bybit.NewError(bybit.ErrorKindUnknown, "", "trading.CancelOrder: parse", err)
	}
	t.forgetMappingByClOrdOrOrd(ack.OrderLinkID, ack.OrderID)
	t.forgetMappingByClOrdOrOrd(req.ClientOrderID, req.OrderID)
	return nil
}

// ---------------------------------------------------------------------
// Batch endpoints (UTA only).
// ---------------------------------------------------------------------

// CreateBatchOrders places a batch of spot orders. Slices longer than
// MaxBatchSize are split into chunks. UTA-only — classic spot accounts
// get retCode 10005.
func (t *TradingClient) CreateBatchOrders(ctx context.Context, reqs []bybitspottypes.CreateOrderRequest) ([]bybitspottypes.BatchOrderResult, error) {
	if len(reqs) == 0 {
		return nil, nil
	}
	var out []bybitspottypes.BatchOrderResult = make([]bybitspottypes.BatchOrderResult, 0, len(reqs))
	var aggErrs []error

	var chunkStart int
	for chunkStart = 0; chunkStart < len(reqs); chunkStart += MaxBatchSize {
		var chunkEnd int = chunkStart + MaxBatchSize
		if chunkEnd > len(reqs) {
			chunkEnd = len(reqs)
		}
		var chunkResult []bybitspottypes.BatchOrderResult
		var err error
		chunkResult, err = t.createBatchChunk(ctx, reqs[chunkStart:chunkEnd])
		out = append(out, chunkResult...)
		if err != nil {
			aggErrs = append(aggErrs, err)
		}
	}
	if len(aggErrs) > 0 {
		return out, errors.Join(aggErrs...)
	}
	return out, nil
}

func (t *TradingClient) createBatchChunk(ctx context.Context, chunk []bybitspottypes.CreateOrderRequest) ([]bybitspottypes.BatchOrderResult, error) {
	var bodies []map[string]any = make([]map[string]any, 0, len(chunk))
	var bodyErrs []error
	var keptIdx []int
	var i int
	for i = 0; i < len(chunk); i++ {
		var b map[string]any
		var err error
		b, err = t.buildCreateOrderBody(chunk[i])
		if err != nil {
			bodyErrs = append(bodyErrs, fmt.Errorf("batch[%d]: %w", i, err))
			continue
		}
		delete(b, "category")
		bodies = append(bodies, b)
		keptIdx = append(keptIdx, i)
	}
	var out []bybitspottypes.BatchOrderResult = placeholderBatchResults(chunk)
	if len(bodies) == 0 {
		return out, errors.Join(bodyErrs...)
	}

	var envelope = map[string]any{
		"category": string(bybitspottypes.CategorySpot),
		"request":  bodies,
	}
	var resp rest.Response
	var rateLimits map[string]string
	var err error
	resp, rateLimits, err = t.c.rest().Do(ctx, rest.Options{
		Method: "POST",
		Path:   "/v5/order/create-batch",
		Body:   envelope,
		Signed: true,
		Meta: rest.RequestMeta{
			OrderCount: len(bodies),
			Symbols:    uniqSortedSymbolsCreate(chunk),
			Category:   string(bybit.RateLimitCategoryPlace),
		},
	})
	if err != nil {
		annotateBatchResultsRateLimits(out, rateLimits)
		return out, err
	}

	var resultPayload batchListPayload
	if err = resp.UnmarshalResult(&resultPayload); err != nil {
		return out, bybit.NewError(bybit.ErrorKindUnknown, "", "trading.CreateBatchOrders: parse result", err)
	}
	var extPayload batchExtListPayload
	if err = unmarshalRetExt(resp, &extPayload); err != nil {
		return out, bybit.NewError(bybit.ErrorKindUnknown, "", "trading.CreateBatchOrders: parse retExtInfo", err)
	}

	var aggErrs []error = bodyErrs
	var now int64 = time.Now().UnixMilli()
	var rowIdx int
	for rowIdx = 0; rowIdx < len(keptIdx); rowIdx++ {
		var inputIdx int = keptIdx[rowIdx]
		var src bybitspottypes.CreateOrderRequest = chunk[inputIdx]

		var orderID string
		var orderLinkID string
		if rowIdx < len(resultPayload.List) {
			orderID = resultPayload.List[rowIdx].OrderID
			orderLinkID = resultPayload.List[rowIdx].OrderLinkID
		}
		var code int
		var msg string
		if rowIdx < len(extPayload.List) {
			code = extPayload.List[rowIdx].Code
			msg = extPayload.List[rowIdx].Msg
		}

		var info bybitspottypes.OrderInfo = bybitspottypes.OrderInfo{
			OrderID:       orderID,
			ClientOrderID: orderLinkID,
			Symbol:        src.Symbol,
			Side:          src.Side,
			OrderType:     resolveOrderType(src),
			TimeInForce:   resolveTimeInForce(src),
			Price:         src.Price,
			Quantity:      src.Quantity,
			LeavesQty:     src.Quantity,
			Status:        bybitspottypes.OrderStatusNew,
			MarketUnit:    src.MarketUnit,
			IsLeverage:    src.IsLeverage,
			CreatedAtMs:   now,
			UpdatedAtMs:   now,
			RateLimits:    rateLimits,
		}
		if code != 0 {
			info.Status = ""
			info.RejectReason = msg
			aggErrs = append(aggErrs, &bybit.Error{
				Kind:      bybit.MapBybitCode(strconv.Itoa(code), msg),
				BybitCode: strconv.Itoa(code),
				Message:   msg,
			})
		} else if orderLinkID != "" && orderID != "" {
			t.rememberMapping(orderLinkID, orderID, now)
		}
		out[inputIdx] = bybitspottypes.BatchOrderResult{
			Order:   info,
			Code:    code,
			Message: msg,
		}
	}
	if len(aggErrs) > 0 {
		return out, errors.Join(aggErrs...)
	}
	return out, nil
}

// ModifyBatchOrders amends a batch of spot orders. retCode 10001 rows
// are filtered out of the aggregated error (idempotent success).
func (t *TradingClient) ModifyBatchOrders(ctx context.Context, reqs []bybitspottypes.ModifyOrderRequest) ([]bybitspottypes.BatchOrderResult, error) {
	if len(reqs) == 0 {
		return nil, nil
	}
	var out []bybitspottypes.BatchOrderResult = make([]bybitspottypes.BatchOrderResult, 0, len(reqs))
	var aggErrs []error

	var chunkStart int
	for chunkStart = 0; chunkStart < len(reqs); chunkStart += MaxBatchSize {
		var chunkEnd int = chunkStart + MaxBatchSize
		if chunkEnd > len(reqs) {
			chunkEnd = len(reqs)
		}
		var chunkResult []bybitspottypes.BatchOrderResult
		var err error
		chunkResult, err = t.modifyBatchChunk(ctx, reqs[chunkStart:chunkEnd])
		out = append(out, chunkResult...)
		if err != nil {
			aggErrs = append(aggErrs, err)
		}
	}
	if len(aggErrs) > 0 {
		return out, filterOrderNotModifiedFromBatchErr(errors.Join(aggErrs...))
	}
	return out, nil
}

func (t *TradingClient) modifyBatchChunk(ctx context.Context, chunk []bybitspottypes.ModifyOrderRequest) ([]bybitspottypes.BatchOrderResult, error) {
	var bodies []map[string]any = make([]map[string]any, 0, len(chunk))
	var bodyErrs []error
	var keptIdx []int
	var i int
	for i = 0; i < len(chunk); i++ {
		var b map[string]any
		var err error
		b, err = buildModifyOrderBody(chunk[i])
		if err != nil {
			bodyErrs = append(bodyErrs, fmt.Errorf("batch[%d]: %w", i, err))
			continue
		}
		delete(b, "category")
		bodies = append(bodies, b)
		keptIdx = append(keptIdx, i)
	}
	var out []bybitspottypes.BatchOrderResult = placeholderBatchModifyResults(chunk)
	if len(bodies) == 0 {
		return out, errors.Join(bodyErrs...)
	}

	var envelope = map[string]any{
		"category": string(bybitspottypes.CategorySpot),
		"request":  bodies,
	}
	var resp rest.Response
	var rateLimits map[string]string
	var err error
	resp, rateLimits, err = t.c.rest().Do(ctx, rest.Options{
		Method: "POST",
		Path:   "/v5/order/amend-batch",
		Body:   envelope,
		Signed: true,
		Meta: rest.RequestMeta{
			OrderCount: len(bodies),
			Symbols:    uniqSortedSymbolsModify(chunk),
			Category:   string(bybit.RateLimitCategoryAmend),
		},
	})
	if err != nil {
		return out, err
	}

	var resultPayload batchListPayload
	if err = resp.UnmarshalResult(&resultPayload); err != nil {
		return out, bybit.NewError(bybit.ErrorKindUnknown, "", "trading.ModifyBatchOrders: parse result", err)
	}
	var extPayload batchExtListPayload
	if err = unmarshalRetExt(resp, &extPayload); err != nil {
		return out, bybit.NewError(bybit.ErrorKindUnknown, "", "trading.ModifyBatchOrders: parse retExtInfo", err)
	}

	var aggErrs []error = bodyErrs
	var now int64 = time.Now().UnixMilli()
	var rowIdx int
	for rowIdx = 0; rowIdx < len(keptIdx); rowIdx++ {
		var inputIdx int = keptIdx[rowIdx]
		var src bybitspottypes.ModifyOrderRequest = chunk[inputIdx]

		var orderID string
		var orderLinkID string
		if rowIdx < len(resultPayload.List) {
			orderID = resultPayload.List[rowIdx].OrderID
			orderLinkID = resultPayload.List[rowIdx].OrderLinkID
		}
		var code int
		var msg string
		if rowIdx < len(extPayload.List) {
			code = extPayload.List[rowIdx].Code
			msg = extPayload.List[rowIdx].Msg
		}
		var info bybitspottypes.OrderInfo = bybitspottypes.OrderInfo{
			OrderID:       orderID,
			ClientOrderID: orderLinkID,
			Symbol:        src.Symbol,
			Price:         src.NewPrice,
			Quantity:      src.NewQuantity,
			LeavesQty:     src.NewQuantity,
			UpdatedAtMs:   now,
			RateLimits:    rateLimits,
		}
		// Per-row idempotent: code=10001 ⇒ treat as success (no agg err).
		var isOrderNotModified = code == 10001
		if code != 0 && !isOrderNotModified {
			info.RejectReason = msg
			aggErrs = append(aggErrs, &bybit.Error{
				Kind:      bybit.MapBybitCode(strconv.Itoa(code), msg),
				BybitCode: strconv.Itoa(code),
				Message:   msg,
			})
		}
		out[inputIdx] = bybitspottypes.BatchOrderResult{
			Order:   info,
			Code:    code,
			Message: msg,
		}
	}
	if len(aggErrs) > 0 {
		return out, errors.Join(aggErrs...)
	}
	return out, nil
}

// CancelBatchOrders cancels a batch of spot orders.
func (t *TradingClient) CancelBatchOrders(ctx context.Context, reqs []bybitspottypes.CancelOrderRequest) ([]bybitspottypes.BatchOrderResult, error) {
	if len(reqs) == 0 {
		return nil, nil
	}
	var out []bybitspottypes.BatchOrderResult = make([]bybitspottypes.BatchOrderResult, 0, len(reqs))
	var aggErrs []error

	var chunkStart int
	for chunkStart = 0; chunkStart < len(reqs); chunkStart += MaxBatchSize {
		var chunkEnd int = chunkStart + MaxBatchSize
		if chunkEnd > len(reqs) {
			chunkEnd = len(reqs)
		}
		var chunkResult []bybitspottypes.BatchOrderResult
		var err error
		chunkResult, err = t.cancelBatchChunk(ctx, reqs[chunkStart:chunkEnd])
		out = append(out, chunkResult...)
		if err != nil {
			aggErrs = append(aggErrs, err)
		}
	}
	if len(aggErrs) > 0 {
		return out, errors.Join(aggErrs...)
	}
	return out, nil
}

func (t *TradingClient) cancelBatchChunk(ctx context.Context, chunk []bybitspottypes.CancelOrderRequest) ([]bybitspottypes.BatchOrderResult, error) {
	var bodies []map[string]any = make([]map[string]any, 0, len(chunk))
	var bodyErrs []error
	var keptIdx []int
	var i int
	for i = 0; i < len(chunk); i++ {
		var b map[string]any
		var err error
		b, err = buildCancelOrderBody(chunk[i])
		if err != nil {
			bodyErrs = append(bodyErrs, fmt.Errorf("batch[%d]: %w", i, err))
			continue
		}
		delete(b, "category")
		bodies = append(bodies, b)
		keptIdx = append(keptIdx, i)
	}
	var out []bybitspottypes.BatchOrderResult = placeholderBatchCancelResults(chunk)
	if len(bodies) == 0 {
		return out, errors.Join(bodyErrs...)
	}

	var envelope = map[string]any{
		"category": string(bybitspottypes.CategorySpot),
		"request":  bodies,
	}
	var resp rest.Response
	var rateLimits map[string]string
	var err error
	resp, rateLimits, err = t.c.rest().Do(ctx, rest.Options{
		Method: "POST",
		Path:   "/v5/order/cancel-batch",
		Body:   envelope,
		Signed: true,
		Meta: rest.RequestMeta{
			OrderCount: len(bodies),
			Symbols:    uniqSortedSymbolsCancel(chunk),
			Category:   string(bybit.RateLimitCategoryCancel),
		},
	})
	if err != nil {
		return out, err
	}

	var resultPayload batchListPayload
	if err = resp.UnmarshalResult(&resultPayload); err != nil {
		return out, bybit.NewError(bybit.ErrorKindUnknown, "", "trading.CancelBatchOrders: parse result", err)
	}
	var extPayload batchExtListPayload
	if err = unmarshalRetExt(resp, &extPayload); err != nil {
		return out, bybit.NewError(bybit.ErrorKindUnknown, "", "trading.CancelBatchOrders: parse retExtInfo", err)
	}

	var aggErrs []error = bodyErrs
	var rowIdx int
	for rowIdx = 0; rowIdx < len(keptIdx); rowIdx++ {
		var inputIdx int = keptIdx[rowIdx]
		var src bybitspottypes.CancelOrderRequest = chunk[inputIdx]

		var orderID string
		var orderLinkID string
		if rowIdx < len(resultPayload.List) {
			orderID = resultPayload.List[rowIdx].OrderID
			orderLinkID = resultPayload.List[rowIdx].OrderLinkID
		}
		var code int
		var msg string
		if rowIdx < len(extPayload.List) {
			code = extPayload.List[rowIdx].Code
			msg = extPayload.List[rowIdx].Msg
		}
		var info bybitspottypes.OrderInfo = bybitspottypes.OrderInfo{
			OrderID:       orderID,
			ClientOrderID: orderLinkID,
			Symbol:        src.Symbol,
			Status:        bybitspottypes.OrderStatusCancelled,
			RateLimits:    rateLimits,
		}
		if code != 0 {
			info.Status = ""
			info.RejectReason = msg
			aggErrs = append(aggErrs, &bybit.Error{
				Kind:      bybit.MapBybitCode(strconv.Itoa(code), msg),
				BybitCode: strconv.Itoa(code),
				Message:   msg,
			})
		} else {
			t.forgetMappingByClOrdOrOrd(orderLinkID, orderID)
			t.forgetMappingByClOrdOrOrd(src.ClientOrderID, src.OrderID)
		}
		out[inputIdx] = bybitspottypes.BatchOrderResult{
			Order:   info,
			Code:    code,
			Message: msg,
		}
	}
	if len(aggErrs) > 0 {
		return out, errors.Join(aggErrs...)
	}
	return out, nil
}

// ---------------------------------------------------------------------
// Bulk endpoints.
// ---------------------------------------------------------------------

// CancelAllOrders cancels every open spot order for `symbol`.
func (t *TradingClient) CancelAllOrders(ctx context.Context, symbol string) ([]bybitspottypes.BatchOrderResult, error) {
	if symbol == "" {
		return nil, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "trading.CancelAllOrders: symbol is empty", nil)
	}
	var body = map[string]any{
		"category": string(bybitspottypes.CategorySpot),
		"symbol":   symbol,
	}
	var resp rest.Response
	var rateLimits map[string]string
	var err error
	resp, rateLimits, err = t.c.rest().Do(ctx, rest.Options{
		Method: "POST",
		Path:   "/v5/order/cancel-all",
		Body:   body,
		Signed: true,
		Meta: rest.RequestMeta{
			Symbols:  []string{symbol},
			Category: string(bybit.RateLimitCategoryCancel),
		},
	})
	if err != nil {
		return nil, err
	}

	var payload batchListPayload
	if err = resp.UnmarshalResult(&payload); err != nil {
		return nil, bybit.NewError(bybit.ErrorKindUnknown, "", "trading.CancelAllOrders: parse", err)
	}

	var out []bybitspottypes.BatchOrderResult = make([]bybitspottypes.BatchOrderResult, 0, len(payload.List))
	var i int
	for i = 0; i < len(payload.List); i++ {
		var row = payload.List[i]
		var info bybitspottypes.OrderInfo = bybitspottypes.OrderInfo{
			OrderID:       row.OrderID,
			ClientOrderID: row.OrderLinkID,
			Symbol:        symbol,
			Status:        bybitspottypes.OrderStatusCancelled,
			RateLimits:    rateLimits,
		}
		t.forgetMappingByClOrdOrOrd(row.OrderLinkID, row.OrderID)
		out = append(out, bybitspottypes.BatchOrderResult{Order: info})
	}
	return out, nil
}

// CancelForgottenOrders cancels every open spot order whose CreatedAtMs
// is older than maxAge. Mirrors the linears profile's reconciliation
// pattern: GetOpenOrders → filter → CancelBatchOrders.
func (t *TradingClient) CancelForgottenOrders(ctx context.Context, symbol string, maxAge time.Duration) ([]bybitspottypes.BatchOrderResult, error) {
	if symbol == "" {
		return nil, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "trading.CancelForgottenOrders: symbol is empty", nil)
	}
	if maxAge <= 0 {
		return nil, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "trading.CancelForgottenOrders: maxAge must be positive", nil)
	}

	var open []bybitspottypes.OrderInfo
	var err error
	open, err = t.c.Account().GetOpenOrders(ctx, symbol)
	if err != nil {
		return nil, err
	}

	var threshold int64 = time.Now().UnixMilli() - maxAge.Milliseconds()
	var reqs []bybitspottypes.CancelOrderRequest
	var i int
	for i = 0; i < len(open); i++ {
		if open[i].CreatedAtMs > 0 && open[i].CreatedAtMs <= threshold {
			reqs = append(reqs, bybitspottypes.CancelOrderRequest{
				Symbol:  symbol,
				OrderID: open[i].OrderID,
			})
		}
	}
	if len(reqs) == 0 {
		return nil, nil
	}
	return t.CancelBatchOrders(ctx, reqs)
}

// ---------------------------------------------------------------------
// Body builders + helpers.
// ---------------------------------------------------------------------

// buildCreateOrderBody validates a CreateOrderRequest and returns the
// JSON body for /v5/order/create.
func (t *TradingClient) buildCreateOrderBody(req bybitspottypes.CreateOrderRequest) (map[string]any, error) {
	if req.Symbol == "" {
		return nil, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "trading.CreateOrder: Symbol is empty", nil)
	}
	if req.Side != bybitspottypes.SideTypeBuy && req.Side != bybitspottypes.SideTypeSell {
		return nil, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "trading.CreateOrder: Side must be Buy/Sell", nil)
	}
	if req.Quantity.IsZero() || req.Quantity.IsNegative() {
		return nil, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "trading.CreateOrder: Quantity must be positive", nil)
	}
	if req.ClientOrderID != "" && !orderLinkIDPattern.MatchString(req.ClientOrderID) {
		return nil, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "trading.CreateOrder: invalid ClientOrderID (1..36 chars [A-Za-z0-9_.-])", nil)
	}

	var orderType bybitspottypes.OrderType = resolveOrderType(req)
	var tif bybitspottypes.TimeInForceType = resolveTimeInForce(req)

	if orderType == bybitspottypes.OrderTypeLimit && (req.Price.IsZero() || req.Price.IsNegative()) {
		return nil, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "trading.CreateOrder: Price must be positive for limit orders", nil)
	}

	var body map[string]any = make(map[string]any, 12)
	body["category"] = string(bybitspottypes.CategorySpot)
	body["symbol"] = req.Symbol
	body["side"] = string(req.Side)
	body["orderType"] = string(orderType)
	body["qty"] = req.Quantity.String()
	if orderType == bybitspottypes.OrderTypeLimit {
		body["price"] = req.Price.String()
	}
	if tif != "" {
		body["timeInForce"] = string(tif)
	}
	if req.ClientOrderID != "" {
		body["orderLinkId"] = req.ClientOrderID
	}
	if req.MarketUnit != "" {
		body["marketUnit"] = string(req.MarketUnit)
	}
	if req.IsLeverage {
		// Bybit V5 expects the integer 1 for margin spot in UTA.
		body["isLeverage"] = 1
	}
	return body, nil
}

// buildModifyOrderBody — shared body constructor for /v5/order/amend
// (category=spot).
func buildModifyOrderBody(req bybitspottypes.ModifyOrderRequest) (map[string]any, error) {
	if req.Symbol == "" {
		return nil, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "trading.ModifyOrder: Symbol is empty", nil)
	}
	if (req.OrderID == "" && req.ClientOrderID == "") || (req.OrderID != "" && req.ClientOrderID != "") {
		return nil, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "trading.ModifyOrder: exactly one of OrderID/ClientOrderID must be set", nil)
	}
	if req.NewQuantity.IsZero() && req.NewPrice.IsZero() {
		return nil, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "trading.ModifyOrder: NewQuantity or NewPrice must be set", nil)
	}
	var body map[string]any = make(map[string]any, 6)
	body["category"] = string(bybitspottypes.CategorySpot)
	body["symbol"] = req.Symbol
	if req.OrderID != "" {
		body["orderId"] = req.OrderID
	}
	if req.ClientOrderID != "" {
		body["orderLinkId"] = req.ClientOrderID
	}
	if !req.NewQuantity.IsZero() {
		body["qty"] = req.NewQuantity.String()
	}
	if !req.NewPrice.IsZero() {
		body["price"] = req.NewPrice.String()
	}
	return body, nil
}

// buildCancelOrderBody — shared body constructor for /v5/order/cancel
// (category=spot).
func buildCancelOrderBody(req bybitspottypes.CancelOrderRequest) (map[string]any, error) {
	if req.Symbol == "" {
		return nil, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "trading.CancelOrder: Symbol is empty", nil)
	}
	if (req.OrderID == "" && req.ClientOrderID == "") || (req.OrderID != "" && req.ClientOrderID != "") {
		return nil, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "trading.CancelOrder: exactly one of OrderID/ClientOrderID must be set", nil)
	}
	var body map[string]any = map[string]any{
		"category": string(bybitspottypes.CategorySpot),
		"symbol":   req.Symbol,
	}
	if req.OrderID != "" {
		body["orderId"] = req.OrderID
	}
	if req.ClientOrderID != "" {
		body["orderLinkId"] = req.ClientOrderID
	}
	return body, nil
}

// resolveOrderType decides the effective OrderType for a request.
//
// PostOnly TIF requires Limit; if the caller leaves OrderType empty
// we default to Limit so the wire stays consistent.
func resolveOrderType(req bybitspottypes.CreateOrderRequest) bybitspottypes.OrderType {
	if req.OrderType != "" {
		return req.OrderType
	}
	return bybitspottypes.OrderTypeLimit
}

// resolveTimeInForce returns the wire TIF — empty leaves it for Bybit's
// server-side default (GTC for Limit, IOC for Market).
func resolveTimeInForce(req bybitspottypes.CreateOrderRequest) bybitspottypes.TimeInForceType {
	return req.TimeInForce
}

// filterOrderNotModifiedFromBatchErr drops retCode=10001 sub-errors
// from an aggregated batch error so callers don't see "order not
// modified" messages in the log when no real error happened. Returns
// nil if every sub-error was a 10001.
func filterOrderNotModifiedFromBatchErr(err error) error {
	if err == nil {
		return nil
	}
	type unwrapper interface{ Unwrap() []error }
	var u unwrapper
	if !errors.As(err, &u) {
		// Single error path — drop only if it's exactly a 10001.
		var bbErr *bybit.Error
		if errors.As(err, &bbErr) && bbErr.BybitCode == retCodeOrderNotModified {
			return nil
		}
		return err
	}
	var kept []error
	var children []error = u.Unwrap()
	var i int
	for i = 0; i < len(children); i++ {
		var ch error = children[i]
		var bbErr *bybit.Error
		if errors.As(ch, &bbErr) && bbErr.BybitCode == retCodeOrderNotModified {
			continue
		}
		kept = append(kept, ch)
	}
	if len(kept) == 0 {
		return nil
	}
	return errors.Join(kept...)
}

// ---------------------------------------------------------------------
// Internal payload structs.
// ---------------------------------------------------------------------

type orderAckPayload struct {
	OrderID     string `json:"orderId"`
	OrderLinkID string `json:"orderLinkId"`
}

type batchListPayload struct {
	List []batchListEntry `json:"list"`
}

type batchListEntry struct {
	Category    string `json:"category"`
	Symbol      string `json:"symbol"`
	OrderID     string `json:"orderId"`
	OrderLinkID string `json:"orderLinkId"`
	CreateAt    string `json:"createAt"`
}

type batchExtListPayload struct {
	List []batchExtListEntry `json:"list"`
}

type batchExtListEntry struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func unmarshalRetExt(resp rest.Response, dest any) error {
	if len(resp.RetExtInfo) == 0 {
		return nil
	}
	if string(resp.RetExtInfo) == "null" {
		return nil
	}
	return codec.Unmarshal(resp.RetExtInfo, dest)
}

// ---------------------------------------------------------------------
// Placeholder helpers (request-side fields preserved when batch fails).
// ---------------------------------------------------------------------

func placeholderBatchResults(chunk []bybitspottypes.CreateOrderRequest) []bybitspottypes.BatchOrderResult {
	var out []bybitspottypes.BatchOrderResult = make([]bybitspottypes.BatchOrderResult, len(chunk))
	var i int
	for i = 0; i < len(chunk); i++ {
		out[i] = bybitspottypes.BatchOrderResult{
			Order: bybitspottypes.OrderInfo{
				Symbol:        chunk[i].Symbol,
				Side:          chunk[i].Side,
				OrderType:     resolveOrderType(chunk[i]),
				TimeInForce:   resolveTimeInForce(chunk[i]),
				Price:         chunk[i].Price,
				Quantity:      chunk[i].Quantity,
				ClientOrderID: chunk[i].ClientOrderID,
				MarketUnit:    chunk[i].MarketUnit,
				IsLeverage:    chunk[i].IsLeverage,
			},
		}
	}
	return out
}

func placeholderBatchModifyResults(chunk []bybitspottypes.ModifyOrderRequest) []bybitspottypes.BatchOrderResult {
	var out []bybitspottypes.BatchOrderResult = make([]bybitspottypes.BatchOrderResult, len(chunk))
	var i int
	for i = 0; i < len(chunk); i++ {
		out[i] = bybitspottypes.BatchOrderResult{
			Order: bybitspottypes.OrderInfo{
				Symbol:        chunk[i].Symbol,
				OrderID:       chunk[i].OrderID,
				ClientOrderID: chunk[i].ClientOrderID,
				Price:         chunk[i].NewPrice,
				Quantity:      chunk[i].NewQuantity,
			},
		}
	}
	return out
}

func placeholderBatchCancelResults(chunk []bybitspottypes.CancelOrderRequest) []bybitspottypes.BatchOrderResult {
	var out []bybitspottypes.BatchOrderResult = make([]bybitspottypes.BatchOrderResult, len(chunk))
	var i int
	for i = 0; i < len(chunk); i++ {
		out[i] = bybitspottypes.BatchOrderResult{
			Order: bybitspottypes.OrderInfo{
				Symbol:        chunk[i].Symbol,
				OrderID:       chunk[i].OrderID,
				ClientOrderID: chunk[i].ClientOrderID,
			},
		}
	}
	return out
}

func annotateBatchResultsRateLimits(out []bybitspottypes.BatchOrderResult, rateLimits map[string]string) {
	if rateLimits == nil {
		return
	}
	var i int
	for i = 0; i < len(out); i++ {
		out[i].Order.RateLimits = rateLimits
	}
}

func uniqSortedSymbolsCreate(chunk []bybitspottypes.CreateOrderRequest) []string {
	if len(chunk) == 0 {
		return nil
	}
	var set map[string]struct{} = make(map[string]struct{}, len(chunk))
	var i int
	for i = 0; i < len(chunk); i++ {
		if chunk[i].Symbol == "" {
			continue
		}
		set[chunk[i].Symbol] = struct{}{}
	}
	if len(set) == 0 {
		return nil
	}
	var out []string = make([]string, 0, len(set))
	var s string
	for s = range set {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

func uniqSortedSymbolsModify(chunk []bybitspottypes.ModifyOrderRequest) []string {
	if len(chunk) == 0 {
		return nil
	}
	var set map[string]struct{} = make(map[string]struct{}, len(chunk))
	var i int
	for i = 0; i < len(chunk); i++ {
		if chunk[i].Symbol == "" {
			continue
		}
		set[chunk[i].Symbol] = struct{}{}
	}
	if len(set) == 0 {
		return nil
	}
	var out []string = make([]string, 0, len(set))
	var s string
	for s = range set {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

func uniqSortedSymbolsCancel(chunk []bybitspottypes.CancelOrderRequest) []string {
	if len(chunk) == 0 {
		return nil
	}
	var set map[string]struct{} = make(map[string]struct{}, len(chunk))
	var i int
	for i = 0; i < len(chunk); i++ {
		if chunk[i].Symbol == "" {
			continue
		}
		set[chunk[i].Symbol] = struct{}{}
	}
	if len(set) == 0 {
		return nil
	}
	var out []string = make([]string, 0, len(set))
	var s string
	for s = range set {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// ---------------------------------------------------------------------
// ID mapping helpers.
// ---------------------------------------------------------------------

func (t *TradingClient) rememberMapping(clOrdID, ordID string, createdAtMs int64) {
	if clOrdID == "" || ordID == "" {
		return
	}
	t.mu.Lock()
	t.clOrdToOrd[clOrdID] = ordID
	t.ordToClOrd[ordID] = clOrdID
	t.createdAtMs[clOrdID] = createdAtMs
	t.mu.Unlock()
}

func (t *TradingClient) forgetMappingByClOrdOrOrd(clOrdID, ordID string) {
	if clOrdID == "" && ordID == "" {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if clOrdID == "" {
		clOrdID = t.ordToClOrd[ordID]
	}
	if ordID == "" {
		ordID = t.clOrdToOrd[clOrdID]
	}
	delete(t.clOrdToOrd, clOrdID)
	delete(t.ordToClOrd, ordID)
	delete(t.createdAtMs, clOrdID)
}

// OrderIDByClientID returns the exchange OrderID for a known
// orderLinkId. Reports false if the SDK has not seen the mapping.
func (t *TradingClient) OrderIDByClientID(clOrdID string) (string, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	var v string
	var ok bool
	v, ok = t.clOrdToOrd[clOrdID]
	return v, ok
}

// ClientIDByOrderID returns the orderLinkId for a known exchange OrderID.
func (t *TradingClient) ClientIDByOrderID(ordID string) (string, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	var v string
	var ok bool
	v, ok = t.ordToClOrd[ordID]
	return v, ok
}

// SyncOrderMappings reconciles the in-process ID map with Bybit's open
// orders. Stale entries are dropped; new entries are added.
func (t *TradingClient) SyncOrderMappings(ctx context.Context, symbol string) error {
	if symbol == "" {
		return bybit.NewError(bybit.ErrorKindInvalidRequest, "", "trading.SyncOrderMappings: symbol is empty", nil)
	}
	var open []bybitspottypes.OrderInfo
	var err error
	open, err = t.c.Account().GetOpenOrders(ctx, symbol)
	if err != nil {
		return err
	}
	var seen map[string]struct{} = make(map[string]struct{}, len(open))
	var i int
	for i = 0; i < len(open); i++ {
		if open[i].ClientOrderID != "" && open[i].OrderID != "" {
			t.rememberMapping(open[i].ClientOrderID, open[i].OrderID, open[i].CreatedAtMs)
			seen[open[i].ClientOrderID] = struct{}{}
		}
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	var clOrdID, ordID string
	for clOrdID, ordID = range t.clOrdToOrd {
		if _, ok := seen[clOrdID]; !ok {
			delete(t.clOrdToOrd, clOrdID)
			delete(t.ordToClOrd, ordID)
			delete(t.createdAtMs, clOrdID)
		}
	}
	return nil
}
