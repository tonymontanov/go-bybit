/*
FILE: internal/rest/client.go

DESCRIPTION:
Low-level Bybit V5 REST client. Stays at the transport / envelope layer:

  - assembles the URL (BaseURL + path + canonical query);
  - signs requests via auth.Signer (X-BAPI-* headers);
  - executes the HTTP call honouring ctx deadline / Config.RequestTimeout;
  - parses the V5 envelope { retCode, retMsg, result, retExtInfo, time };
  - maps non-zero retCode and 4xx/5xx HTTP statuses into *bberr.Error with
    the right Kind via bberr.MapBybitCode / bberr.MapHTTPStatus;
  - notifies the legacy and event-based rate-limit observers with the
    headers Bybit returns (X-Bapi-Limit / X-Bapi-Limit-Status /
    X-Bapi-Limit-Reset-Timestamp).

WHY DOES THE OBSERVER GET ACTUAL HEADERS HERE (DIFFERENT FROM OKX):
Bybit DOES return rate-limit headers on every signed REST response, unlike
OKX (where the equivalent observer always sees an empty map). The headers
are forwarded as-is so an external rate-limiter at the desk level can
reconcile its model with the live remaining budget.

DESIGN:
  - The package does NOT import the public root (which imports rest), so
    everything we need lives in internal/* (auth, codec, bberr, bblog).
  - Domain layers (linears/trading.go, etc.) call Do() with a populated
    Options.Meta describing OrderCount / Symbols / Category. The metadata
    is forwarded to the event-observer 1:1; the legacy observer receives
    only (endpoint, headers) for source-level back-compat.
*/

package rest

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/tonymontanov/go-bybit/v2/internal/auth"
	"github.com/tonymontanov/go-bybit/v2/internal/bberr"
	"github.com/tonymontanov/go-bybit/v2/internal/bblog"
	"github.com/tonymontanov/go-bybit/v2/internal/codec"
)

// Config — REST transport parameters. Populated from public Config.REST in
// the root package via explicit field copy (avoids an import cycle: the
// root package imports internal/rest, not the other way around).
type Config struct {
	// RequestTimeout — global timeout for a single REST call. ctx with its
	// own deadline overrides this for a particular request.
	RequestTimeout time.Duration
	// MaxIdleConns — http.Transport pool size.
	MaxIdleConns int
	// MaxIdleConnsPerHost — http.Transport per-host pool size.
	MaxIdleConnsPerHost int
	// IdleConnTimeout — keep-alive idle timeout.
	IdleConnTimeout time.Duration
	// RecvWindow — value sent in the X-BAPI-RECV-WINDOW header (ms). Bybit
	// rejects requests where (server_time - timestamp) > recvWindow.
	// Default 5000ms is plenty for most scenarios; trading-from-VPN setups
	// may need 10000.
	RecvWindow int
	// RateLimitObserver — legacy callback (kept for source-level back-compat
	// with the OKX-style observer pattern). Receives only (endpoint, headers).
	// nil → no-op.
	RateLimitObserver func(endpoint string, headers map[string]string)
	// RateLimitEventObserver — primary callback. Receives every REST
	// response with the full RequestMeta (OrderCount/Symbols/Category) plus
	// the live X-Bapi-Limit-* headers. nil → no-op.
	//
	// Speed contract: called synchronously from the goroutine that issued
	// the REST request. Implementations must be O(1) — typically a
	// non-blocking send to a buffered channel. Blocking the observer
	// stalls the REST pipeline.
	RateLimitEventObserver func(endpoint, method string, headers map[string]string, meta RequestMeta)
}

// RequestMeta — domain-level information about the request that the
// external rate-limiter needs to model Bybit V5 limits accurately. The
// meta is set by the calling domain method (linears/trading.go etc.) at
// the point where the symbol set, batch size, and category are known.
//
// Fields:
//
//   - OrderCount: 1 for single-order endpoints, len(orders) for batch
//     endpoints, 0 for non-trading. Bybit accounts for batch
//     budgets in orders, not requests.
//   - Symbols:    sorted unique list of symbols affected by the request.
//     Bybit V5 trading limits are per (UID + Symbol) on
//     derivatives; the rate-limiter uses this to debit only
//     the affected symbol(s) instead of blocking all of them.
//   - Category:   "place" | "amend" | "cancel" | "query" | "market" | "".
//     Used by the rate-limiter to model the sub-account-level
//     NEW+AMEND budget separately from cancellations and
//     queries.
//   - Endpoint:   the V5 path in canonical form (e.g. "/v5/order/create").
//     Set by the rest client itself just before invoking the
//     observer; callers do not need to populate it.
type RequestMeta struct {
	OrderCount int
	Symbols    []string
	Category   string
	// Endpoint — populated by Do() before invoking the observer; ignored on
	// input.
	Endpoint string
}

// Options — parameters for a single REST request.
type Options struct {
	// Method — HTTP method, must be upper-case ("GET", "POST", ...). The
	// client also tolerates lower-case and upper-cases internally.
	Method string
	// Path — request path including the leading "/" (e.g. "/v5/order/create").
	Path string
	// Query — query parameters; serialized in URL-encoded form. For signed
	// GET requests the canonical query string is also part of the
	// signature pre-hash.
	Query url.Values
	// Body — JSON body. Marshalled by codec; the resulting bytes are used
	// both for the wire and for the signature pre-hash. Pass nil for GET.
	Body any
	// Signed — true for endpoints that require X-BAPI-SIGN.
	Signed bool
	// Meta — request metadata for the rate-limit observer.
	Meta RequestMeta
}

// Response — Bybit V5 response envelope. The result is kept as raw JSON so
// domain methods can decode into typed structs without re-marshalling.
type Response struct {
	RetCode    int                `json:"retCode"`
	RetMsg     string             `json:"retMsg"`
	Result     jsoniterRawMessage `json:"result"`
	RetExtInfo jsoniterRawMessage `json:"retExtInfo"`
	Time       int64              `json:"time"`
}

// jsoniterRawMessage is a json.RawMessage equivalent that works correctly
// with jsoniter. Same trick as in go-okx/internal/rest.
type jsoniterRawMessage []byte

// MarshalJSON implements json.Marshaler.
func (m jsoniterRawMessage) MarshalJSON() ([]byte, error) {
	if len(m) == 0 {
		return []byte("null"), nil
	}
	return []byte(m), nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (m *jsoniterRawMessage) UnmarshalJSON(data []byte) error {
	*m = append((*m)[:0], data...)
	return nil
}

// UnmarshalResult decodes the Result field into dest. No-op if Result is
// missing or "null".
func (r Response) UnmarshalResult(dest any) error {
	if len(r.Result) == 0 || bytes.Equal(r.Result, []byte("null")) {
		return nil
	}
	return codec.Unmarshal(r.Result, dest)
}

// Client — low-level REST client.
type Client struct {
	httpClient             *http.Client
	signer                 *auth.Signer
	baseURL                string
	userAgent              string
	logger                 bblog.Logger
	recvWindow             string
	rateLimitObserver      func(endpoint string, headers map[string]string)
	rateLimitEventObserver func(endpoint, method string, headers map[string]string, meta RequestMeta)
}

// NewClient creates a REST client. signer may have empty credentials —
// public endpoints will still work, signed calls will surface
// auth.ErrSignerDisabled at call time.
func NewClient(baseURL string, signer *auth.Signer, cfg Config, ua string, log bblog.Logger) *Client {
	if log == nil {
		log = bblog.Noop()
	}
	if cfg.RecvWindow <= 0 {
		cfg.RecvWindow = 5000
	}
	var transport *http.Transport = &http.Transport{
		MaxIdleConns:        cfg.MaxIdleConns,
		MaxIdleConnsPerHost: cfg.MaxIdleConnsPerHost,
		IdleConnTimeout:     cfg.IdleConnTimeout,
		ForceAttemptHTTP2:   true,
	}
	var httpClient *http.Client = &http.Client{
		Timeout:   cfg.RequestTimeout,
		Transport: transport,
	}
	return &Client{
		httpClient:             httpClient,
		signer:                 signer,
		baseURL:                strings.TrimRight(baseURL, "/"),
		userAgent:              ua,
		logger:                 log,
		recvWindow:             strconv.Itoa(cfg.RecvWindow),
		rateLimitObserver:      cfg.RateLimitObserver,
		rateLimitEventObserver: cfg.RateLimitEventObserver,
	}
}

// Close releases idle transport connections.
func (c *Client) Close() {
	if c == nil || c.httpClient == nil {
		return
	}
	if t, ok := c.httpClient.Transport.(*http.Transport); ok {
		t.CloseIdleConnections()
	}
}

/*
Do executes a single REST call. Returns the response envelope, the
collected rate-limit headers, and an error.

Error semantics:
  - context cancel/deadline / network failures → *bberr.Error with
    Kind=Network, Cause=underlying error.
  - HTTP 4xx/5xx without a parseable envelope → *bberr.Error with
    Kind = bberr.MapHTTPStatus(status), HTTPStatus set, Message=truncated body.
  - HTTP 2xx with retCode != 0                → *bberr.Error with
    Kind = bberr.MapBybitCode(retCode, retMsg), BybitCode=retCode,
    Message=retMsg, HTTPStatus=2xx.
  - HTTP 4xx/5xx WITH a parseable envelope (Bybit sometimes wraps errors
    in the standard envelope on 5xx)             → same as above.

The rate-limit headers map is always non-nil but may be empty (e.g. for
public endpoints that do not advertise per-UID limits). It is allocated
fresh on every call so observers may safely retain references.
*/
func (c *Client) Do(ctx context.Context, opts Options) (Response, map[string]string, error) {
	var resp Response
	var rateHeaders map[string]string = map[string]string{}

	// Build URL and body BEFORE signing — the signature must cover the
	// exact bytes that go on the wire.
	var fullURL string
	var bodyStr string
	var canonicalQuery string
	var err error
	fullURL, bodyStr, canonicalQuery, err = c.buildRequest(opts)
	if err != nil {
		return resp, rateHeaders, err
	}

	var method string = strings.ToUpper(opts.Method)

	var req *http.Request
	req, err = http.NewRequestWithContext(ctx, method, fullURL, bytes.NewBufferString(bodyStr))
	if err != nil {
		return resp, rateHeaders, bberr.New(bberr.ErrorKindInvalidRequest, "", "rest: build request", err)
	}

	c.applyHeaders(req, opts, method, bodyStr, canonicalQuery)

	var httpResp *http.Response
	var started time.Time = time.Now()
	httpResp, err = c.httpClient.Do(req)
	if err != nil {
		return resp, rateHeaders, classifyTransportError(err)
	}
	defer func() {
		_ = httpResp.Body.Close()
	}()

	// Collect the rate-limit headers BEFORE notifying observers and
	// BEFORE parsing the body. Even if the body turns out to be malformed,
	// the headers are still meaningful.
	rateHeaders = collectRateLimitHeaders(httpResp.Header)

	// Notify observers. The observer is called with the canonical endpoint
	// from opts.Path (we deliberately do NOT pass the full URL — observers
	// key off endpoints, not URLs).
	if c.rateLimitObserver != nil || c.rateLimitEventObserver != nil {
		var meta RequestMeta = opts.Meta
		meta.Endpoint = opts.Path
		if c.rateLimitObserver != nil {
			c.rateLimitObserver(opts.Path, rateHeaders)
		}
		if c.rateLimitEventObserver != nil {
			c.rateLimitEventObserver(opts.Path, method, rateHeaders, meta)
		}
	}

	var raw []byte
	raw, err = io.ReadAll(httpResp.Body)
	if err != nil {
		return resp, rateHeaders, bberr.New(bberr.ErrorKindNetwork, "", "rest: read body", err)
	}

	c.logger.Debug(
		"rest.Do",
		bblog.Str("method", method),
		bblog.Str("path", opts.Path),
		bblog.Int("status", int64(httpResp.StatusCode)),
		bblog.Int("durationMs", time.Since(started).Milliseconds()),
		bblog.Int("bytes", int64(len(raw))),
	)

	// Try to decode the envelope on every status. Bybit returns the same
	// {retCode, retMsg, result} wrapper on 4xx/5xx for application-level
	// validation errors, and only falls back to plain-text on infra
	// failures (LB returning 502 with HTML, etc.).
	var parseErr error = codec.Unmarshal(raw, &resp)

	if httpResp.StatusCode >= 200 && httpResp.StatusCode < 300 {
		if parseErr != nil {
			return resp, rateHeaders, bberr.New(bberr.ErrorKindUnknown, "", "rest: parse response", parseErr)
		}
		if resp.RetCode != 0 {
			var code string = strconv.Itoa(resp.RetCode)
			return resp, rateHeaders, &bberr.Error{
				Kind:       bberr.MapBybitCode(code, resp.RetMsg),
				HTTPStatus: httpResp.StatusCode,
				BybitCode:  code,
				Message:    resp.RetMsg,
			}
		}
		return resp, rateHeaders, nil
	}

	// Non-2xx path. Prefer the typed envelope when available.
	if parseErr == nil && resp.RetCode != 0 {
		var code string = strconv.Itoa(resp.RetCode)
		return resp, rateHeaders, &bberr.Error{
			Kind:       bberr.MapBybitCode(code, resp.RetMsg),
			HTTPStatus: httpResp.StatusCode,
			BybitCode:  code,
			Message:    resp.RetMsg,
		}
	}
	return resp, rateHeaders, &bberr.Error{
		Kind:       bberr.MapHTTPStatus(httpResp.StatusCode),
		HTTPStatus: httpResp.StatusCode,
		Message:    truncate(string(raw), 256),
	}
}

// buildRequest assembles the URL, the body string, and the canonical
// query string used for signing. The canonical query is the same string
// that is appended to the URL — calling code MUST sign exactly that.
func (c *Client) buildRequest(opts Options) (string, string, string, error) {
	var u *url.URL
	var err error
	u, err = url.Parse(c.baseURL + opts.Path)
	if err != nil {
		return "", "", "", bberr.New(bberr.ErrorKindInvalidRequest, "", "rest: invalid url", err)
	}
	var canonicalQuery string
	if len(opts.Query) > 0 {
		canonicalQuery = opts.Query.Encode()
		u.RawQuery = canonicalQuery
	}

	var bodyStr string
	if opts.Body != nil {
		var raw []byte
		raw, err = codec.Marshal(opts.Body)
		if err != nil {
			return "", "", "", bberr.New(bberr.ErrorKindInvalidRequest, "", "rest: marshal body", err)
		}
		bodyStr = string(raw)
	}
	return u.String(), bodyStr, canonicalQuery, nil
}

// applyHeaders sets common headers and, for signed calls, the V5 X-BAPI-*
// headers.
func (c *Client) applyHeaders(req *http.Request, opts Options, method, body, canonicalQuery string) {
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	if method != http.MethodGet {
		req.Header.Set("Content-Type", "application/json")
	}

	if !opts.Signed {
		return
	}
	if c.signer == nil || !c.signer.Enabled() {
		// Surface signing failure later via auth.ErrSignerDisabled at the
		// call site that builds the error explicitly. The transport layer
		// keeps going so that public-only embedders are not broken.
		return
	}

	// For GET the signature pre-hash uses the canonical query string (the
	// same one already appended to the URL). For POST/PUT/DELETE it uses
	// the JSON body string (which may be empty). Bybit V5 signs whichever
	// payload is sent — exactly one of the two is non-empty for typical
	// signed calls.
	var signPayload string
	if method == http.MethodGet {
		signPayload = canonicalQuery
	} else {
		signPayload = body
	}

	var ts string = c.signer.MillisTimestamp(time.Now())
	var signature string
	var err error
	signature, err = c.signer.SignREST(ts, c.recvWindow, signPayload)
	if err != nil {
		c.logger.Warn("rest: sign skipped", bblog.Err(err))
		return
	}
	req.Header.Set("X-BAPI-API-KEY", c.signer.APIKey())
	req.Header.Set("X-BAPI-TIMESTAMP", ts)
	req.Header.Set("X-BAPI-RECV-WINDOW", c.recvWindow)
	req.Header.Set("X-BAPI-SIGN", signature)
	req.Header.Set("X-BAPI-SIGN-TYPE", "2")
}

// collectRateLimitHeaders extracts the X-Bapi-Limit-* family that Bybit V5
// returns on every signed REST response. The header keys are normalised to
// canonical Go form (http.Header already does that on receive). The
// returned map is fresh on every call.
func collectRateLimitHeaders(h http.Header) map[string]string {
	var out map[string]string = map[string]string{}
	var name string
	var values []string
	for name, values = range h {
		if len(values) == 0 {
			continue
		}
		var lower string = strings.ToLower(name)
		// Bybit ships a small fixed set of rate-limit headers. Hard-code the
		// allow-list to avoid leaking unrelated headers (cookies, auth) into
		// observer maps that may be logged downstream.
		switch lower {
		case "x-bapi-limit",
			"x-bapi-limit-status",
			"x-bapi-limit-reset-timestamp",
			"x-bapi-recv-window-status",
			"retry-after":
			out[name] = values[0]
		}
	}
	return out
}

// classifyTransportError converts a transport error into *bberr.Error
// with Kind=Network. Distinguishes context cancel / deadline so callers
// can use errors.Is(err, context.Canceled) when needed.
func classifyTransportError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) {
		return bberr.New(bberr.ErrorKindNetwork, "", "rest: context canceled", err)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return bberr.New(bberr.ErrorKindNetwork, "", "rest: deadline exceeded", err)
	}
	return bberr.New(bberr.ErrorKindNetwork, "", "rest: transport error", err)
}

// truncate returns at most n bytes of s, appending an ellipsis when cut.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
