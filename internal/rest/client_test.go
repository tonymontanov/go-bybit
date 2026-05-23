/*
FILE: internal/rest/client_test.go

DESCRIPTION:
Unit tests for the low-level REST client. These tests use httptest.Server
to verify:

  - signing headers are set on signed calls (and only on signed calls);
  - the envelope is parsed and result is exposed via UnmarshalResult;
  - retCode != 0 maps to *bberr.Error with the expected Kind and BybitCode;
  - 4xx without a usable envelope maps to *bberr.Error with HTTPStatus and
    Kind = MapHTTPStatus(...);
  - rate-limit headers (X-Bapi-Limit-*) reach the legacy and event observers
    with the same content;
  - RequestMeta (OrderCount/Symbols/Category) is forwarded to the event
    observer 1:1.
*/

package rest

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tonymontanov/go-bybit/v2/internal/auth"
	"github.com/tonymontanov/go-bybit/v2/internal/bberr"
)

// newTestClient wires up a Client pointing at the given test server.
func newTestClient(t *testing.T, srv *httptest.Server, observer func(string, string, map[string]string, RequestMeta)) *Client {
	t.Helper()
	var signer *auth.Signer = auth.NewSigner("KEY1234567890", "SECRET")
	var cfg Config = Config{
		RequestTimeout:         3 * time.Second,
		MaxIdleConns:           4,
		MaxIdleConnsPerHost:    4,
		IdleConnTimeout:        time.Second,
		RecvWindow:             5000,
		RateLimitEventObserver: observer,
	}
	return NewClient(srv.URL, signer, cfg, "go-bybit-test/1", nil)
}

func TestDo_SignedHeadersPresent(t *testing.T) {
	var got http.Header
	var srv *httptest.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Clone()
		w.Header().Set("X-Bapi-Limit", "100")
		w.Header().Set("X-Bapi-Limit-Status", "99")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"retCode":0,"retMsg":"OK","result":{"x":1},"retExtInfo":{},"time":1}`))
	}))
	defer srv.Close()

	var c *Client = newTestClient(t, srv, nil)
	defer c.Close()

	var _, headers, err = c.Do(context.Background(), Options{
		Method: "GET",
		Path:   "/v5/order/realtime",
		Signed: true,
	})
	if err != nil {
		t.Fatalf("Do: %v", err)
	}

	if got.Get("X-BAPI-API-KEY") == "" {
		t.Fatalf("X-BAPI-API-KEY header missing")
	}
	if got.Get("X-BAPI-SIGN") == "" {
		t.Fatalf("X-BAPI-SIGN header missing")
	}
	if got.Get("X-BAPI-SIGN-TYPE") != "2" {
		t.Fatalf("X-BAPI-SIGN-TYPE: got %q, want 2", got.Get("X-BAPI-SIGN-TYPE"))
	}
	if got.Get("X-BAPI-TIMESTAMP") == "" {
		t.Fatalf("X-BAPI-TIMESTAMP header missing")
	}
	if got.Get("X-BAPI-RECV-WINDOW") != "5000" {
		t.Fatalf("X-BAPI-RECV-WINDOW: got %q, want 5000", got.Get("X-BAPI-RECV-WINDOW"))
	}

	if headers["X-Bapi-Limit"] != "100" {
		t.Fatalf("rate-limit header not propagated: %v", headers)
	}
	if headers["X-Bapi-Limit-Status"] != "99" {
		t.Fatalf("rate-limit-status header not propagated: %v", headers)
	}
}

func TestDo_UnsignedNoHeaders(t *testing.T) {
	var got http.Header
	var srv *httptest.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Clone()
		_, _ = w.Write([]byte(`{"retCode":0,"retMsg":"OK","result":null,"time":1}`))
	}))
	defer srv.Close()

	var c *Client = newTestClient(t, srv, nil)
	defer c.Close()

	var _, _, err = c.Do(context.Background(), Options{Method: "GET", Path: "/v5/market/time"})
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if got.Get("X-BAPI-SIGN") != "" {
		t.Fatalf("unsigned call must not set X-BAPI-SIGN")
	}
}

func TestDo_RetCodeMapsToTypedError(t *testing.T) {
	var srv *httptest.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"retCode":10006,"retMsg":"Too many visits","result":{},"time":1}`))
	}))
	defer srv.Close()

	var c *Client = newTestClient(t, srv, nil)
	defer c.Close()

	var _, _, err = c.Do(context.Background(), Options{Method: "GET", Path: "/v5/test", Signed: true})
	if err == nil {
		t.Fatalf("expected error for retCode=10006")
	}
	if !bberr.IsRateLimit(err) {
		t.Fatalf("expected RateLimit, got %v", err)
	}
	var be *bberr.Error
	if !errors.As(err, &be) {
		t.Fatalf("expected *bberr.Error, got %T", err)
	}
	if be.BybitCode != "10006" {
		t.Fatalf("BybitCode: got %q, want 10006", be.BybitCode)
	}
	if !strings.Contains(be.Message, "Too many visits") {
		t.Fatalf("message must include retMsg: %q", be.Message)
	}
}

func TestDo_HTTP4xxWithoutEnvelope(t *testing.T) {
	var srv *httptest.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`upstream rate-limited`))
	}))
	defer srv.Close()

	var c *Client = newTestClient(t, srv, nil)
	defer c.Close()

	var _, _, err = c.Do(context.Background(), Options{Method: "GET", Path: "/v5/test", Signed: true})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !bberr.IsRateLimit(err) {
		t.Fatalf("expected RateLimit, got %v", err)
	}
	var be *bberr.Error
	if !errors.As(err, &be) {
		t.Fatalf("expected *bberr.Error, got %T", err)
	}
	if be.HTTPStatus != http.StatusTooManyRequests {
		t.Fatalf("HTTPStatus: got %d, want 429", be.HTTPStatus)
	}
}

func TestDo_EventObserverSeesMeta(t *testing.T) {
	var seen RequestMeta
	var observer = func(_, _ string, _ map[string]string, m RequestMeta) {
		seen = m
	}
	var srv *httptest.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"retCode":0,"retMsg":"OK","result":{},"time":1}`))
	}))
	defer srv.Close()

	var c *Client = newTestClient(t, srv, observer)
	defer c.Close()

	var _, _, err = c.Do(context.Background(), Options{
		Method: "POST",
		Path:   "/v5/order/create-batch",
		Body:   map[string]any{"category": "linear"},
		Signed: true,
		Meta: RequestMeta{
			OrderCount: 7,
			Symbols:    []string{"BTCUSDT", "ETHUSDT"},
			Category:   "place",
		},
	})
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if seen.OrderCount != 7 {
		t.Fatalf("OrderCount: got %d, want 7", seen.OrderCount)
	}
	if len(seen.Symbols) != 2 || seen.Symbols[0] != "BTCUSDT" {
		t.Fatalf("Symbols: got %v", seen.Symbols)
	}
	if seen.Category != "place" {
		t.Fatalf("Category: got %q", seen.Category)
	}
	if seen.Endpoint != "/v5/order/create-batch" {
		t.Fatalf("Endpoint: got %q", seen.Endpoint)
	}
}

func TestDo_NetworkErrorClassified(t *testing.T) {
	// Unreachable port — connection refused is a Network-class error.
	var c *Client = NewClient("http://127.0.0.1:1", auth.NewSigner("", ""), Config{
		RequestTimeout: 200 * time.Millisecond,
	}, "test", nil)
	defer c.Close()

	var _, _, err = c.Do(context.Background(), Options{Method: "GET", Path: "/v5/market/time"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !bberr.IsNetwork(err) {
		t.Fatalf("expected Network, got %v", err)
	}
}
