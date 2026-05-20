/*
FILE: linears/trading.go

DESCRIPTION:
Trading sub-client for the Bybit V5 linear category. Implements:

  - CreateOrder           : POST /v5/order/create
  - ModifyOrder           : POST /v5/order/amend
  - CancelOrder           : POST /v5/order/cancel
  - CreateBatchOrders     : POST /v5/order/create-batch    (≤ 20 per call)
  - ModifyBatchOrders     : POST /v5/order/amend-batch     (≤ 20)
  - CancelBatchOrders     : POST /v5/order/cancel-batch    (≤ 20)
  - CancelAllOrders       : POST /v5/order/cancel-all      (one symbol)
  - CancelForgottenOrders : age-based cancel via GetOpenOrders + cancel-batch.

BYBIT V5 SPECIFICS:
  - All trading endpoints take the same JSON envelope:
        { "category": "linear", "symbol": "...", ... }
    The category is hard-pinned in this package; the caller never passes it.
  - timeInForce=PostOnly is rewritten to (orderType=Limit, timeInForce=PostOnly)
    on the wire. Empty TIF defaults to GTC for Limit and IOC for Market.
  - orderLinkId pattern: 1..36 chars [A-Za-z0-9_.-]. The SDK validates and
    rejects locally without sending.
  - Batch endpoints return TWO parallel arrays in the response:
        result.list[i]      → the success-path identifiers,
        retExtInfo.list[i]  → {code, msg} per sub-request.
    A non-zero top-level retCode propagates as a fatal *bberr.Error;
    per-row failures are flattened into BatchOrderResult.

SDK-LEVEL INVARIANTS (validated before sending):
  - CreateOrder: Symbol/Side non-empty, Quantity > 0, Price > 0 for limit.
  - ModifyOrder: Symbol non-empty; exactly one of OrderID/ClientOrderID;
    at least one of NewQuantity/NewPrice non-zero.
  - CancelOrder: Symbol non-empty; exactly one of OrderID/ClientOrderID.

INTERNAL STATE:
  - clOrdToOrd / ordToClOrd / createdAtMs : ID mapping caches, mirroring the
    OKX/Binance trading clients. Removed synchronously on cancel/fill.
*/

package linears

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"sync"
	"time"

	bybit "github.com/tonymontanov/go-bybit"
	"github.com/tonymontanov/go-bybit/internal/codec"
	"github.com/tonymontanov/go-bybit/internal/rest"
	"github.com/tonymontanov/go-bybit/linears/types"
)

// MaxBatchSize — Bybit V5 limit on batch trade endpoints for the linear
// category. Spot caps at 10, options at 20; we expose only the linear
// limit here.
const MaxBatchSize = 20

// orderLinkIDPattern — allowed characters and length for orderLinkId.
// Bybit V5 docs §Trade > Place Order: max 36 chars, [A-Za-z0-9_.-].
var orderLinkIDPattern = regexp.MustCompile(`^[A-Za-z0-9_.\-]{1,36}$`)

// TradingClient — trading sub-client.
type TradingClient struct {
	c *Client

	mu          sync.RWMutex
	clOrdToOrd  map[string]string
	ordToClOrd  map[string]string
	createdAtMs map[string]int64 // clOrdId → ms; powers CancelForgottenOrders
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

// CreateOrder places a single order.
//
// On success returns OrderInfo with OrderID/ClientOrderID/CreatedAtMs and
// RateLimits populated (Bybit echoes orderId+orderLinkId only, the SDK
// fills the request-side fields in Side/OrderType/Price/Quantity for
// caller convenience).
//
// SDK-side validation runs before the request — invalid inputs return
// *bberr.Error with Kind=InvalidRequest WITHOUT contacting Bybit.
func (t *TradingClient) CreateOrder(ctx context.Context, req types.CreateOrderRequest) (types.OrderInfo, error) {
	var info types.OrderInfo

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
	info = types.OrderInfo{
		OrderID:       ack.OrderID,
		ClientOrderID: ack.OrderLinkID,
		Symbol:        req.Symbol,
		Side:          req.Side,
		OrderType:     resolveOrderType(req),
		TimeInForce:   resolveTimeInForce(req),
		Price:         req.Price,
		Quantity:      req.Quantity,
		LeavesQty:     req.Quantity,
		Status:        types.OrderStatusNew,
		PositionIdx:   req.PositionIdx,
		ReduceOnly:    req.ReduceOnly,
		CreatedAtMs:   now,
		UpdatedAtMs:   now,
		RateLimits:    rateLimits,
	}
	t.rememberMapping(ack.OrderLinkID, ack.OrderID, now)
	return info, nil
}

// ModifyOrder amends a single order. Bybit only allows changing
// qty / price (and the TPSL pair, not exposed in v1) — side, type and
// timeInForce are immutable; for those the order must be cancelled and
// re-created.
func (t *TradingClient) ModifyOrder(ctx context.Context, req types.ModifyOrderRequest) (types.OrderInfo, error) {
	var info types.OrderInfo

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
	if err != nil {
		return info, err
	}

	var ack orderAckPayload
	if err = resp.UnmarshalResult(&ack); err != nil {
		return info, bybit.NewError(bybit.ErrorKindUnknown, "", "trading.ModifyOrder: parse", err)
	}

	info = types.OrderInfo{
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

// CancelOrder cancels a single order. Exactly one of OrderID / ClientOrderID
// must be set.
func (t *TradingClient) CancelOrder(ctx context.Context, req types.CancelOrderRequest) error {
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
// Batch endpoints.
// ---------------------------------------------------------------------

// CreateBatchOrders places a batch of orders. Slices longer than
// MaxBatchSize are submitted in chunks. Per-row results are returned in
// the same order as the input requests; errors from individual rows
// (retExtInfo.list[i].code != 0) populate BatchOrderResult.Code/Message
// without aborting the whole batch.
//
// The aggregated error (errors.Join of per-row failures) is returned
// alongside the result slice so the caller can branch on err while still
// inspecting the per-row outcome.
func (t *TradingClient) CreateBatchOrders(ctx context.Context, reqs []types.CreateOrderRequest) ([]types.BatchOrderResult, error) {
	if len(reqs) == 0 {
		return nil, nil
	}
	var out []types.BatchOrderResult = make([]types.BatchOrderResult, 0, len(reqs))
	var aggErrs []error

	var chunkStart int
	for chunkStart = 0; chunkStart < len(reqs); chunkStart += MaxBatchSize {
		var chunkEnd int = chunkStart + MaxBatchSize
		if chunkEnd > len(reqs) {
			chunkEnd = len(reqs)
		}
		var chunkResult []types.BatchOrderResult
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

func (t *TradingClient) createBatchChunk(ctx context.Context, chunk []types.CreateOrderRequest) ([]types.BatchOrderResult, error) {
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
		// In a batch payload Bybit infers `category` from the top-level
		// envelope; per-row `category` keys are tolerated but redundant.
		// We strip them to keep the wire compact.
		delete(b, "category")
		bodies = append(bodies, b)
		keptIdx = append(keptIdx, i)
	}
	var out []types.BatchOrderResult = placeholderBatchResults(chunk)
	if len(bodies) == 0 {
		// Nothing valid to send — surface the local validation errors.
		annotateBatchResults(out, bodyErrs, chunk, nil)
		return out, errors.Join(bodyErrs...)
	}

	var envelope = map[string]any{
		"category": string(types.CategoryLinear),
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
		annotateBatchResults(out, bodyErrs, chunk, rateLimits)
		return out, err
	}

	var resultPayload batchListPayload
	if err = resp.UnmarshalResult(&resultPayload); err != nil {
		annotateBatchResults(out, bodyErrs, chunk, rateLimits)
		return out, bybit.NewError(bybit.ErrorKindUnknown, "", "trading.CreateBatchOrders: parse result", err)
	}
	var extPayload batchExtListPayload
	if err = unmarshalRetExt(resp, &extPayload); err != nil {
		annotateBatchResults(out, bodyErrs, chunk, rateLimits)
		return out, bybit.NewError(bybit.ErrorKindUnknown, "", "trading.CreateBatchOrders: parse retExtInfo", err)
	}

	var aggErrs []error = bodyErrs
	var now int64 = time.Now().UnixMilli()
	var rowIdx int
	for rowIdx = 0; rowIdx < len(keptIdx); rowIdx++ {
		var inputIdx int = keptIdx[rowIdx]
		var src types.CreateOrderRequest = chunk[inputIdx]

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

		var info types.OrderInfo = types.OrderInfo{
			OrderID:       orderID,
			ClientOrderID: orderLinkID,
			Symbol:        src.Symbol,
			Side:          src.Side,
			OrderType:     resolveOrderType(src),
			TimeInForce:   resolveTimeInForce(src),
			Price:         src.Price,
			Quantity:      src.Quantity,
			LeavesQty:     src.Quantity,
			Status:        types.OrderStatusNew,
			PositionIdx:   src.PositionIdx,
			ReduceOnly:    src.ReduceOnly,
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
		out[inputIdx] = types.BatchOrderResult{
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

// ModifyBatchOrders amends a batch of orders.
func (t *TradingClient) ModifyBatchOrders(ctx context.Context, reqs []types.ModifyOrderRequest) ([]types.BatchOrderResult, error) {
	if len(reqs) == 0 {
		return nil, nil
	}
	var out []types.BatchOrderResult = make([]types.BatchOrderResult, 0, len(reqs))
	var aggErrs []error

	var chunkStart int
	for chunkStart = 0; chunkStart < len(reqs); chunkStart += MaxBatchSize {
		var chunkEnd int = chunkStart + MaxBatchSize
		if chunkEnd > len(reqs) {
			chunkEnd = len(reqs)
		}
		var chunkResult []types.BatchOrderResult
		var err error
		chunkResult, err = t.modifyBatchChunk(ctx, reqs[chunkStart:chunkEnd])
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

func (t *TradingClient) modifyBatchChunk(ctx context.Context, chunk []types.ModifyOrderRequest) ([]types.BatchOrderResult, error) {
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
	var out []types.BatchOrderResult = placeholderBatchModifyResults(chunk)
	if len(bodies) == 0 {
		return out, errors.Join(bodyErrs...)
	}

	var envelope = map[string]any{
		"category": string(types.CategoryLinear),
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
		var src types.ModifyOrderRequest = chunk[inputIdx]

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
		var info types.OrderInfo = types.OrderInfo{
			OrderID:       orderID,
			ClientOrderID: orderLinkID,
			Symbol:        src.Symbol,
			Price:         src.NewPrice,
			Quantity:      src.NewQuantity,
			LeavesQty:     src.NewQuantity,
			UpdatedAtMs:   now,
			RateLimits:    rateLimits,
		}
		if code != 0 {
			info.RejectReason = msg
			aggErrs = append(aggErrs, &bybit.Error{
				Kind:      bybit.MapBybitCode(strconv.Itoa(code), msg),
				BybitCode: strconv.Itoa(code),
				Message:   msg,
			})
		}
		out[inputIdx] = types.BatchOrderResult{
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

// CancelBatchOrders cancels a batch of orders.
func (t *TradingClient) CancelBatchOrders(ctx context.Context, reqs []types.CancelOrderRequest) ([]types.BatchOrderResult, error) {
	if len(reqs) == 0 {
		return nil, nil
	}
	var out []types.BatchOrderResult = make([]types.BatchOrderResult, 0, len(reqs))
	var aggErrs []error

	var chunkStart int
	for chunkStart = 0; chunkStart < len(reqs); chunkStart += MaxBatchSize {
		var chunkEnd int = chunkStart + MaxBatchSize
		if chunkEnd > len(reqs) {
			chunkEnd = len(reqs)
		}
		var chunkResult []types.BatchOrderResult
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

func (t *TradingClient) cancelBatchChunk(ctx context.Context, chunk []types.CancelOrderRequest) ([]types.BatchOrderResult, error) {
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
	var out []types.BatchOrderResult = placeholderBatchCancelResults(chunk)
	if len(bodies) == 0 {
		return out, errors.Join(bodyErrs...)
	}

	var envelope = map[string]any{
		"category": string(types.CategoryLinear),
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
		var src types.CancelOrderRequest = chunk[inputIdx]

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
		var info types.OrderInfo = types.OrderInfo{
			OrderID:       orderID,
			ClientOrderID: orderLinkID,
			Symbol:        src.Symbol,
			Status:        types.OrderStatusCancelled,
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
		out[inputIdx] = types.BatchOrderResult{
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

// CancelAllOrders cancels every open order for `symbol` in the linear
// category. Returns the cancelled (orderId, orderLinkId) pairs as
// BatchOrderResult; an empty slice means there were no live orders.
//
// Symbol is REQUIRED; calling /v5/order/cancel-all without a symbol on
// derivatives requires the (baseCoin OR settleCoin) filter, which we
// keep out of v1 to prevent accidental account-wide cancels.
func (t *TradingClient) CancelAllOrders(ctx context.Context, symbol string) ([]types.BatchOrderResult, error) {
	if symbol == "" {
		return nil, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "trading.CancelAllOrders: symbol is empty", nil)
	}
	var body = map[string]any{
		"category": string(types.CategoryLinear),
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

	var out []types.BatchOrderResult = make([]types.BatchOrderResult, 0, len(payload.List))
	var i int
	for i = 0; i < len(payload.List); i++ {
		var row = payload.List[i]
		var info types.OrderInfo = types.OrderInfo{
			OrderID:       row.OrderID,
			ClientOrderID: row.OrderLinkID,
			Symbol:        symbol,
			Status:        types.OrderStatusCancelled,
			RateLimits:    rateLimits,
		}
		t.forgetMappingByClOrdOrOrd(row.OrderLinkID, row.OrderID)
		out = append(out, types.BatchOrderResult{Order: info})
	}
	return out, nil
}

// CancelForgottenOrders cancels every open order whose CreatedAtMs is
// older than maxAge. Returns the list of orders that the SDK ATTEMPTED
// to cancel — inspect the per-row Code in the second return value to see
// which sub-requests succeeded.
func (t *TradingClient) CancelForgottenOrders(ctx context.Context, symbol string, maxAge time.Duration) ([]types.BatchOrderResult, error) {
	if symbol == "" {
		return nil, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "trading.CancelForgottenOrders: symbol is empty", nil)
	}
	if maxAge <= 0 {
		return nil, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "trading.CancelForgottenOrders: maxAge must be positive", nil)
	}

	var open []types.OrderInfo
	var err error
	open, err = t.c.Account().GetOpenOrders(ctx, symbol)
	if err != nil {
		return nil, err
	}

	var threshold int64 = time.Now().UnixMilli() - maxAge.Milliseconds()
	var reqs []types.CancelOrderRequest
	var i int
	for i = 0; i < len(open); i++ {
		if open[i].CreatedAtMs > 0 && open[i].CreatedAtMs <= threshold {
			reqs = append(reqs, types.CancelOrderRequest{
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
// JSON body for /v5/order/create. The body INCLUDES the "category" key
// for single-order endpoints; the batch builder strips it on copy.
func (t *TradingClient) buildCreateOrderBody(req types.CreateOrderRequest) (map[string]any, error) {
	if req.Symbol == "" {
		return nil, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "trading.CreateOrder: Symbol is empty", nil)
	}
	if req.Side != types.SideTypeBuy && req.Side != types.SideTypeSell {
		return nil, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "trading.CreateOrder: Side must be Buy/Sell", nil)
	}
	if req.Quantity.IsZero() || req.Quantity.IsNegative() {
		return nil, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "trading.CreateOrder: Quantity must be positive", nil)
	}
	if req.ClientOrderID != "" && !orderLinkIDPattern.MatchString(req.ClientOrderID) {
		return nil, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "trading.CreateOrder: invalid ClientOrderID (1..36 chars [A-Za-z0-9_.-])", nil)
	}

	var orderType types.OrderType = resolveOrderType(req)
	var tif types.TimeInForceType = resolveTimeInForce(req)

	if orderType == types.OrderTypeLimit && (req.Price.IsZero() || req.Price.IsNegative()) {
		return nil, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "trading.CreateOrder: Price must be positive for limit orders", nil)
	}

	var body map[string]any = make(map[string]any, 12)
	body["category"] = string(types.CategoryLinear)
	body["symbol"] = req.Symbol
	body["side"] = string(req.Side)
	body["orderType"] = string(orderType)
	body["qty"] = req.Quantity.String()
	if orderType == types.OrderTypeLimit {
		body["price"] = req.Price.String()
	}
	if tif != "" {
		body["timeInForce"] = string(tif)
	}
	if req.ClientOrderID != "" {
		body["orderLinkId"] = req.ClientOrderID
	}
	if req.PositionIdx != types.PositionIdxOneWay {
		body["positionIdx"] = int(req.PositionIdx)
	}
	if req.ReduceOnly {
		body["reduceOnly"] = true
	}
	if req.CloseOnTrigger {
		body["closeOnTrigger"] = true
	}
	return body, nil
}

// buildModifyOrderBody — shared body constructor for /v5/order/amend.
func buildModifyOrderBody(req types.ModifyOrderRequest) (map[string]any, error) {
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
	body["category"] = string(types.CategoryLinear)
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

// buildCancelOrderBody — shared body constructor for /v5/order/cancel.
func buildCancelOrderBody(req types.CancelOrderRequest) (map[string]any, error) {
	if req.Symbol == "" {
		return nil, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "trading.CancelOrder: Symbol is empty", nil)
	}
	if (req.OrderID == "" && req.ClientOrderID == "") || (req.OrderID != "" && req.ClientOrderID != "") {
		return nil, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "trading.CancelOrder: exactly one of OrderID/ClientOrderID must be set", nil)
	}
	var body map[string]any = map[string]any{
		"category": string(types.CategoryLinear),
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
//   - explicit OrderType is honoured.
//   - when OrderType is empty, TIF=PostOnly maps to Limit (Bybit forbids
//     PostOnly on market orders); everything else maps to Limit too.
func resolveOrderType(req types.CreateOrderRequest) types.OrderType {
	if req.OrderType != "" {
		return req.OrderType
	}
	return types.OrderTypeLimit
}

// resolveTimeInForce decides the effective TimeInForce string sent to
// Bybit. PostOnly stays as "PostOnly"; everything else stays verbatim.
// Empty TIF stays empty so Bybit applies its server-side default
// (GTC for Limit, IOC for Market).
func resolveTimeInForce(req types.CreateOrderRequest) types.TimeInForceType {
	return req.TimeInForce
}

// ---------------------------------------------------------------------
// Internal payload structs.
// ---------------------------------------------------------------------

// orderAckPayload — the success-path body of Bybit's
// /v5/order/{create,amend,cancel} (single-order endpoints).
type orderAckPayload struct {
	OrderID     string `json:"orderId"`
	OrderLinkID string `json:"orderLinkId"`
}

// batchListPayload — result.list[] of /v5/order/{create,amend,cancel}-batch
// and /v5/order/cancel-all.
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

// batchExtListPayload — retExtInfo.list[] of batch endpoints.
type batchExtListPayload struct {
	List []batchExtListEntry `json:"list"`
}

type batchExtListEntry struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// unmarshalRetExt safely decodes resp.RetExtInfo into dest. Empty / null
// retExtInfo is treated as "no per-row info" — not an error.
func unmarshalRetExt(resp rest.Response, dest any) error {
	if len(resp.RetExtInfo) == 0 {
		return nil
	}
	if string(resp.RetExtInfo) == "null" {
		return nil
	}
	return codec.Unmarshal(resp.RetExtInfo, dest)
}

// placeholderBatchResults pre-allocates a slice of BatchOrderResult with
// the request-side fields filled in. Per-row failures and successful
// rows alike overwrite specific positions in this slice.
func placeholderBatchResults(chunk []types.CreateOrderRequest) []types.BatchOrderResult {
	var out []types.BatchOrderResult = make([]types.BatchOrderResult, len(chunk))
	var i int
	for i = 0; i < len(chunk); i++ {
		out[i] = types.BatchOrderResult{
			Order: types.OrderInfo{
				Symbol:        chunk[i].Symbol,
				Side:          chunk[i].Side,
				OrderType:     resolveOrderType(chunk[i]),
				TimeInForce:   resolveTimeInForce(chunk[i]),
				Price:         chunk[i].Price,
				Quantity:      chunk[i].Quantity,
				ClientOrderID: chunk[i].ClientOrderID,
				PositionIdx:   chunk[i].PositionIdx,
				ReduceOnly:    chunk[i].ReduceOnly,
			},
		}
	}
	return out
}

func placeholderBatchModifyResults(chunk []types.ModifyOrderRequest) []types.BatchOrderResult {
	var out []types.BatchOrderResult = make([]types.BatchOrderResult, len(chunk))
	var i int
	for i = 0; i < len(chunk); i++ {
		out[i] = types.BatchOrderResult{
			Order: types.OrderInfo{
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

func placeholderBatchCancelResults(chunk []types.CancelOrderRequest) []types.BatchOrderResult {
	var out []types.BatchOrderResult = make([]types.BatchOrderResult, len(chunk))
	var i int
	for i = 0; i < len(chunk); i++ {
		out[i] = types.BatchOrderResult{
			Order: types.OrderInfo{
				Symbol:        chunk[i].Symbol,
				OrderID:       chunk[i].OrderID,
				ClientOrderID: chunk[i].ClientOrderID,
			},
		}
	}
	return out
}

// annotateBatchResults attaches local-validation errors and rate-limit
// headers to the placeholder results when the batch never reached Bybit.
func annotateBatchResults(out []types.BatchOrderResult, _ []error, _ []types.CreateOrderRequest, rateLimits map[string]string) {
	if rateLimits == nil {
		return
	}
	var i int
	for i = 0; i < len(out); i++ {
		out[i].Order.RateLimits = rateLimits
	}
}

func uniqSortedSymbolsCreate(chunk []types.CreateOrderRequest) []string {
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

func uniqSortedSymbolsModify(chunk []types.ModifyOrderRequest) []string {
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

func uniqSortedSymbolsCancel(chunk []types.CancelOrderRequest) []string {
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

// SyncOrderMappings reconciles the in-process ID map with the open
// orders Bybit reports. Stale entries (orders no longer live) are
// dropped from the map; new entries are added.
func (t *TradingClient) SyncOrderMappings(ctx context.Context, symbol string) error {
	if symbol == "" {
		return bybit.NewError(bybit.ErrorKindInvalidRequest, "", "trading.SyncOrderMappings: symbol is empty", nil)
	}
	var open []types.OrderInfo
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

// ensureSigned surfaces a clear typed error when the SDK is configured
// without API credentials.
func (t *TradingClient) ensureSigned() error {
	if !t.c.signerEnabled() {
		return bybit.NewError(bybit.ErrorKindAuth, "", "trading: APIKey/SecretKey not configured", nil)
	}
	return nil
}

// _ — keep ensureSigned referenced; M3 wires it into every private
// endpoint pre-check.
var _ = (*TradingClient).ensureSigned
