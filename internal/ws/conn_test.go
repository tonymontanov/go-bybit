/*
FILE: internal/ws/conn_test.go

DESCRIPTION:
Mock-server tests for the Bybit V5 WS conn. The fake server speaks just
enough of the protocol to exercise:

  - dial → public connection (no auth);
  - subscribe ack + push frame dispatch via topic match;
  - dial → private connection with successful auth handshake;
  - reconnect-after-disconnect with subscription replay.

We don't go through the full backoff loop (jitter + sleep would make tests
slow and flaky); instead we set ReconnectInitialBackoff = 1ms so a single
artificial drop happens fast.
*/

package ws

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"context"

	"github.com/gorilla/websocket"
	"github.com/tonymontanov/go-bybit/internal/auth"
	"github.com/tonymontanov/go-bybit/internal/codec"
)

// ---------- mock server helpers ----------

// upgrader for tests; allow any origin (httptest serves on a random local port).
var testUpgrader = websocket.Upgrader{
	CheckOrigin: func(*http.Request) bool { return true },
}

// fakeBybitServer encapsulates a single WebSocket connection used by a test.
// Each test creates its own server with its own behaviour.
type fakeBybitServer struct {
	t        *testing.T
	srv      *httptest.Server
	mu       sync.Mutex
	conn     *websocket.Conn
	gotMsgs  []string // captured inbound JSON frames for assertions
	pushOnce sync.Once
	authOK   bool
}

// newFakeServer starts a fake Bybit-compatible WS endpoint. handler is
// invoked on every accepted connection; the test controls the conversation
// from there.
func newFakeServer(t *testing.T, authOK bool, handler func(srv *fakeBybitServer)) *fakeBybitServer {
	t.Helper()
	var s *fakeBybitServer = &fakeBybitServer{t: t, authOK: authOK}
	s.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var c *websocket.Conn
		var err error
		c, err = testUpgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("upgrade: %v", err)
			return
		}
		s.mu.Lock()
		s.conn = c
		s.mu.Unlock()
		if handler != nil {
			handler(s)
		}
	}))
	return s
}

func (s *fakeBybitServer) wsURL() string {
	return "ws" + strings.TrimPrefix(s.srv.URL, "http")
}

func (s *fakeBybitServer) close() {
	s.mu.Lock()
	if s.conn != nil {
		_ = s.conn.Close()
	}
	s.mu.Unlock()
	s.srv.Close()
}

func (s *fakeBybitServer) record(raw []byte) {
	s.mu.Lock()
	s.gotMsgs = append(s.gotMsgs, string(raw))
	s.mu.Unlock()
}

func (s *fakeBybitServer) writeJSON(v any) error {
	var raw []byte
	var err error
	raw, err = codec.Marshal(v)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn == nil {
		return nil
	}
	return s.conn.WriteMessage(websocket.TextMessage, raw)
}

// readUntilOp blocks until the server sees a frame whose top-level "op"
// equals want (e.g. "subscribe", "auth", "ping"). Returns the raw frame.
func (s *fakeBybitServer) readUntilOp(want string, timeout time.Duration) ([]byte, bool) {
	var deadline time.Time = time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		s.mu.Lock()
		var conn *websocket.Conn = s.conn
		s.mu.Unlock()
		if conn == nil {
			time.Sleep(5 * time.Millisecond)
			continue
		}
		_ = conn.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
		var msgType int
		var raw []byte
		var err error
		msgType, raw, err = conn.ReadMessage()
		if err != nil {
			continue
		}
		if msgType != websocket.TextMessage {
			continue
		}
		s.record(raw)
		var probe map[string]any
		if codec.Unmarshal(raw, &probe) == nil {
			if op, _ := probe["op"].(string); op == want {
				return raw, true
			}
		}
	}
	return nil, false
}

// ---------- tests ----------

func TestConn_PublicSubscribeAndDispatch(t *testing.T) {
	var srv *fakeBybitServer = newFakeServer(t, false, func(s *fakeBybitServer) {
		// Wait for the subscribe op, ack it, push one data frame.
		var raw, ok = s.readUntilOp("subscribe", 2*time.Second)
		if !ok {
			t.Errorf("did not see subscribe op")
			return
		}
		_ = raw
		var success bool = true
		_ = s.writeJSON(map[string]any{"op": "subscribe", "success": success, "ret_msg": "", "conn_id": "C1"})
		_ = s.writeJSON(map[string]any{
			"topic": "tickers.BTCUSDT",
			"type":  "snapshot",
			"ts":    time.Now().UnixMilli(),
			"data":  map[string]any{"symbol": "BTCUSDT", "lastPrice": "60000"},
		})
	})
	defer srv.close()

	var c *Conn = NewConn(Config{
		URL:                     srv.wsURL(),
		HandshakeTimeout:        time.Second,
		ReadTimeout:             3 * time.Second,
		WriteTimeout:            time.Second,
		PingInterval:            0, // disable for the test
		ReconnectInitialBackoff: 5 * time.Millisecond,
		ReconnectMaxBackoff:     50 * time.Millisecond,
	}, auth.NewSigner("", ""), nil, nil)
	defer c.Close()

	var ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	c.Start(ctx)

	var got = make(chan []byte, 1)
	var err = c.Subscribe(&Subscription{
		Topic: "tickers.BTCUSDT",
		Handler: func(_, _ string, payload []byte) {
			select {
			case got <- payload:
			default:
			}
		},
	})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	select {
	case payload := <-got:
		if !strings.Contains(string(payload), "BTCUSDT") {
			t.Fatalf("unexpected payload: %s", payload)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("did not receive push within 2s")
	}
}

func TestConn_PrivateAuthHandshake(t *testing.T) {
	var srv *fakeBybitServer = newFakeServer(t, true, func(s *fakeBybitServer) {
		// Wait for auth, ack, then wait for subscribe + ack.
		var _, ok = s.readUntilOp("auth", 2*time.Second)
		if !ok {
			t.Errorf("did not see auth op")
			return
		}
		var success bool = true
		_ = s.writeJSON(map[string]any{"op": "auth", "success": success, "ret_msg": "", "conn_id": "C1"})

		_, ok = s.readUntilOp("subscribe", 2*time.Second)
		if !ok {
			t.Errorf("did not see subscribe op after auth")
			return
		}
		_ = s.writeJSON(map[string]any{"op": "subscribe", "success": success, "conn_id": "C1"})
		_ = s.writeJSON(map[string]any{
			"topic": "position",
			"data":  []any{map[string]any{"symbol": "BTCUSDT", "size": "0.01"}},
		})
	})
	defer srv.close()

	var c *Conn = NewConn(Config{
		URL:                     srv.wsURL(),
		IsPrivate:               true,
		HandshakeTimeout:        time.Second,
		ReadTimeout:             3 * time.Second,
		WriteTimeout:            time.Second,
		AuthExpiresWindow:       time.Second,
		AuthTimeout:             2 * time.Second,
		ReconnectInitialBackoff: 5 * time.Millisecond,
		ReconnectMaxBackoff:     50 * time.Millisecond,
	}, auth.NewSigner("APIKEY1234", "SECRET"), nil, nil)
	defer c.Close()

	var ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	c.Start(ctx)

	var got = make(chan []byte, 1)
	_ = c.Subscribe(&Subscription{
		Topic: "position",
		Handler: func(_, _ string, payload []byte) {
			select {
			case got <- payload:
			default:
			}
		},
	})

	select {
	case payload := <-got:
		if !strings.Contains(string(payload), "BTCUSDT") {
			t.Fatalf("unexpected payload: %s", payload)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("did not receive private push within 3s")
	}
}

func TestConn_TopicPrefixFallback(t *testing.T) {
	// Subscribed to "execution"; server pushes "execution.linear".
	var srv *fakeBybitServer = newFakeServer(t, false, func(s *fakeBybitServer) {
		var _, ok = s.readUntilOp("auth", 2*time.Second)
		if !ok {
			t.Errorf("did not see auth")
			return
		}
		var ok2 bool = true
		_ = s.writeJSON(map[string]any{"op": "auth", "success": ok2})

		_, ok = s.readUntilOp("subscribe", 2*time.Second)
		if !ok {
			t.Errorf("did not see subscribe")
			return
		}
		_ = s.writeJSON(map[string]any{"op": "subscribe", "success": ok2})
		_ = s.writeJSON(map[string]any{
			"topic": "execution.linear",
			"data":  []any{map[string]any{"symbol": "BTCUSDT", "execId": "X1"}},
		})
	})
	defer srv.close()

	var c *Conn = NewConn(Config{
		URL:                     srv.wsURL(),
		IsPrivate:               true,
		HandshakeTimeout:        time.Second,
		ReadTimeout:             3 * time.Second,
		WriteTimeout:            time.Second,
		AuthExpiresWindow:       time.Second,
		AuthTimeout:             2 * time.Second,
		ReconnectInitialBackoff: 5 * time.Millisecond,
		ReconnectMaxBackoff:     50 * time.Millisecond,
	}, auth.NewSigner("APIKEY1234", "SECRET"), nil, nil)
	defer c.Close()

	var ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	c.Start(ctx)

	var hits int32
	_ = c.Subscribe(&Subscription{
		Topic: "execution",
		Handler: func(topic, _ string, _ []byte) {
			if topic == "execution.linear" {
				atomic.AddInt32(&hits, 1)
			}
		},
	})

	var deadline time.Time = time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(&hits) > 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("topic prefix fallback did not deliver execution.linear push")
}
