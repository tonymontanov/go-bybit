/*
FILE: internal/ws/conn.go

DESCRIPTION:
A managing wrapper over a single Bybit V5 WebSocket connection. At most two
such objects are created per domain client (linears.Client, spot.Client,
...): one for the public endpoint (no auth, market-data topics) and one
for the private endpoint (auth required, account/position/order topics).

RESPONSIBILITIES:
  - connect / reconnect with exponential backoff + jitter;
  - private-only auth (op=auth, see internal/auth.SignWS);
  - heartbeat (op=ping every PingInterval, application-level);
  - subscribe / unsubscribe with a topic registry that survives reconnects;
  - resubscribe after every successful (re)connect, transparently to caller;
  - dispatch incoming push frames to the per-topic handler;
  - graceful shutdown via Close() or ctx cancellation.

DESIGN NOTES (DIFFERENCES VS. OKX-style WS):
  - Bybit identifies a subscription by a single topic string
    ("orderbook.50.BTCUSDT", "tickers.BTCUSDT", "position",
    "execution", ...). There is no channel+instId tuple.
  - Subscribe / unsubscribe payloads carry a list of topic strings; the SDK
    sends one topic per request to keep error attribution simple
    (Bybit accepts up to 10 per request, but if one is invalid the whole
    request is rejected).
  - Bybit's ping is APPLICATION-level JSON ({"op":"ping"}), NOT the
    WebSocket protocol-level control frame. Sending only a control-frame
    Ping causes Bybit to time out the connection after ~20s. The SDK
    therefore writes JSON pings on a ticker and never relies on
    gorilla/websocket's PingHandler.
  - The SDK does not implement WS Trade API ops here in M0. SendOp-style
    request/response correlation is deferred to a separate file when /
    if needed (Bybit's order entry over WS is independent of the data
    channel and could live in a future ws-trade package).

CONCURRENCY:
  - mu guards subs/socket/closed/cancel.
  - writeMu guards the underlying gorilla/websocket conn writes (gorilla
    requires exclusive writes).
  - Background goroutines (read-loop + ping-loop) are started afresh on
    every connect and torn down on every disconnect; the supervisor runs
    in its own goroutine for the entire lifetime of Conn.

ERROR STRATEGY:
  - Read errors → readLoop exits with that error → supervise reconnects
    after backoff.
  - Application-level "wrong subscription" or "auth failed" replies are
    logged and counted in metrics, but the supervisor keeps trying. This
    avoids the "transient backend hiccup → permanent connector death"
    failure mode.
  - On Conn.Close() the supervisor exits cleanly without further
    reconnect attempts.
*/

package ws

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/tonymontanov/go-bybit/internal/auth"
	"github.com/tonymontanov/go-bybit/internal/bberr"
	"github.com/tonymontanov/go-bybit/internal/bblog"
	"github.com/tonymontanov/go-bybit/internal/bbmet"
	"github.com/tonymontanov/go-bybit/internal/codec"
)

// ErrConnClosed is returned by operations performed on a closed Conn.
var ErrConnClosed = errors.New("ws: connection closed")

// Subscription describes a single topic subscription. The caller (domain
// stream package) constructs it once and passes it to Subscribe; the same
// Subscription is reused on every reconnect via its Reset hook.
type Subscription struct {
	// Topic — full Bybit topic string ("orderbook.50.BTCUSDT",
	// "tickers.BTCUSDT", "publicTrade.BTCUSDT", "position",
	// "execution.linear", ...). Required.
	Topic string
	// Handler is invoked for every push frame whose topic matches. Args
	// are the topic (so the same handler can serve multiple symbols if
	// the caller wants) and the raw bytes of the frame's "data" field.
	// Push frames whose data field is missing or null still call the
	// handler with payload=nil — handlers must be defensive.
	Handler func(topic string, payload []byte)
	// Reset is called once before every (re)subscribe. Used by the
	// orderbook engine to drop any local state so the next snapshot
	// pushed by the server is treated as the new authoritative state.
	// May be nil.
	Reset func()
}

// Config — parameters for a single Bybit WS connection. Populated from the
// public root config via field-by-field copy.
type Config struct {
	// URL — wss://stream.bybit.com/v5/public/linear,
	//       wss://stream.bybit.com/v5/private,
	//       or testnet/demo equivalents.
	URL string
	// IsPrivate — true for the private endpoint (auth required).
	IsPrivate bool
	// HandshakeTimeout — TLS+HTTP upgrade timeout.
	HandshakeTimeout time.Duration
	// ReadTimeout — read deadline used to detect a silent server. Should be
	// >= 1.5 * PingInterval so a single dropped pong does not trigger a
	// reconnect.
	ReadTimeout time.Duration
	// WriteTimeout — write deadline.
	WriteTimeout time.Duration
	// PingInterval — interval between application-level {"op":"ping"} frames.
	// Bybit's default server-side timeout is 20s; recommended values are
	// 15-20s.
	PingInterval time.Duration
	// AuthExpiresWindow — window added to the current time when computing
	// the WS auth `expires` argument. Default 1s. Larger values trade
	// signature replay safety for tolerance to clock skew.
	AuthExpiresWindow time.Duration
	// AuthTimeout — how long to wait for the auth ack before deciding
	// auth failed. Should comfortably exceed RTT * 3. Default 3s.
	AuthTimeout time.Duration
	// ReconnectInitialBackoff — first sleep after a connection failure.
	ReconnectInitialBackoff time.Duration
	// ReconnectMaxBackoff — cap for the exponential backoff.
	ReconnectMaxBackoff time.Duration
	// ReconnectJitter — random multiplier [1-j, 1+j] applied to backoff.
	// 0 disables jitter.
	ReconnectJitter float64
	// ReadBufferSize / WriteBufferSize — gorilla/websocket buffer sizes.
	ReadBufferSize  int
	WriteBufferSize int
}

// Conn — managing wrapper over a single Bybit V5 WS connection.
type Conn struct {
	cfg     Config
	signer  *auth.Signer
	logger  bblog.Logger
	metrics bbmet.CounterFactory

	mu     sync.RWMutex
	subs   map[string]*Subscription
	socket *websocket.Conn
	closed bool
	cancel context.CancelFunc

	writeMu sync.Mutex

	startOnce sync.Once

	cReceived bbmet.Counter
	cDropped  bbmet.Counter
	cReconn   bbmet.Counter
	cSub      bbmet.Counter
	cPingErr  bbmet.Counter
	cAuthOK   bbmet.Counter
	cAuthFail bbmet.Counter
}

// NewConn creates a Conn. No network activity occurs until Start (or the
// first Subscribe) is called. log/mf may be nil.
func NewConn(cfg Config, signer *auth.Signer, log bblog.Logger, mf bbmet.CounterFactory) *Conn {
	if log == nil {
		log = bblog.Noop()
	}
	if mf == nil {
		mf = bbmet.Noop()
	}
	if cfg.AuthExpiresWindow <= 0 {
		cfg.AuthExpiresWindow = time.Second
	}
	if cfg.AuthTimeout <= 0 {
		cfg.AuthTimeout = 3 * time.Second
	}
	return &Conn{
		cfg:       cfg,
		signer:    signer,
		logger:    log,
		metrics:   mf,
		subs:      make(map[string]*Subscription, 16),
		cReceived: mf.Counter("bybit_ws_messages_received_total"),
		cDropped:  mf.Counter("bybit_ws_messages_dropped_total"),
		cReconn:   mf.Counter("bybit_ws_reconnects_total"),
		cSub:      mf.Counter("bybit_ws_subscriptions_total"),
		cPingErr:  mf.Counter("bybit_ws_ping_failed_total"),
		cAuthOK:   mf.Counter("bybit_ws_auth_ok_total"),
		cAuthFail: mf.Counter("bybit_ws_auth_failed_total"),
	}
}

// Start launches the background supervisor (idempotent). It returns
// immediately; the supervisor exits when ctx is cancelled or Close is
// called.
func (c *Conn) Start(ctx context.Context) {
	c.startOnce.Do(func() {
		var supCtx context.Context
		supCtx, c.cancel = context.WithCancel(ctx)
		go c.supervise(supCtx)
	})
}

// Subscribe registers a subscription and, if the socket is up, sends the
// subscribe op immediately. Otherwise the subscription waits in the
// registry and is sent automatically on the next successful (re)connect.
func (c *Conn) Subscribe(sub *Subscription) error {
	if sub == nil || sub.Topic == "" || sub.Handler == nil {
		return bberr.New(bberr.ErrorKindInvalidRequest, "", "ws: invalid subscription", nil)
	}
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return ErrConnClosed
	}
	c.subs[sub.Topic] = sub
	var socket *websocket.Conn = c.socket
	c.mu.Unlock()
	c.cSub.Inc()

	if socket == nil {
		return nil
	}
	return c.sendOp(socket, "subscribe", []any{sub.Topic})
}

// Unsubscribe removes the topic from the registry. If the socket is up,
// an unsubscribe op is sent. No error is returned for already-unknown
// topics — Unsubscribe is idempotent.
func (c *Conn) Unsubscribe(topic string) error {
	c.mu.Lock()
	delete(c.subs, topic)
	var socket *websocket.Conn = c.socket
	c.mu.Unlock()
	if socket == nil {
		return nil
	}
	return c.sendOp(socket, "unsubscribe", []any{topic})
}

// Close stops the supervisor and the underlying socket. Idempotent.
func (c *Conn) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	if c.cancel != nil {
		c.cancel()
	}
	var s *websocket.Conn = c.socket
	c.socket = nil
	c.mu.Unlock()

	if s != nil {
		_ = s.Close()
	}
	return nil
}

// supervise is the connect → run → backoff loop. Exits on ctx.Done.
func (c *Conn) supervise(ctx context.Context) {
	var backoff time.Duration = c.cfg.ReconnectInitialBackoff
	var attempt int
	for {
		if ctx.Err() != nil {
			return
		}
		var err error = c.connectAndRun(ctx)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			c.logger.Warn("ws: connection error, will reconnect",
				bblog.Str("url", c.cfg.URL),
				bblog.Int("attempt", int64(attempt)),
				bblog.Err(err),
			)
		}
		c.cReconn.Inc()
		attempt++

		var sleep time.Duration = applyJitter(backoff, c.cfg.ReconnectJitter)
		select {
		case <-ctx.Done():
			return
		case <-time.After(sleep):
		}
		backoff = nextBackoff(backoff, c.cfg.ReconnectMaxBackoff)
	}
}

// connectAndRun owns one full connection lifecycle: dial → auth (if private)
// → resubscribe → read-loop + ping-loop.
func (c *Conn) connectAndRun(ctx context.Context) error {
	var dialer *websocket.Dialer = &websocket.Dialer{
		HandshakeTimeout: c.cfg.HandshakeTimeout,
		ReadBufferSize:   c.cfg.ReadBufferSize,
		WriteBufferSize:  c.cfg.WriteBufferSize,
	}
	var socket *websocket.Conn
	var err error
	socket, _, err = dialer.DialContext(ctx, c.cfg.URL, nil)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	c.logger.Info("ws: connected", bblog.Str("url", c.cfg.URL))

	// Snapshot the current subscriptions and call Reset BEFORE we publish
	// the new socket — that way a stale push that arrived on the previous
	// socket cannot race with the engine reset.
	c.mu.Lock()
	c.socket = socket
	var subsCopy []*Subscription = make([]*Subscription, 0, len(c.subs))
	var s *Subscription
	for _, s = range c.subs {
		if s.Reset != nil {
			s.Reset()
		}
		subsCopy = append(subsCopy, s)
	}
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		if c.socket == socket {
			c.socket = nil
		}
		c.mu.Unlock()
		_ = socket.Close()
	}()

	if c.cfg.IsPrivate {
		if err = c.performAuth(socket); err != nil {
			c.cAuthFail.Inc()
			return fmt.Errorf("auth: %w", err)
		}
		c.cAuthOK.Inc()
	}

	var i int
	for i = 0; i < len(subsCopy); i++ {
		if err = c.sendOp(socket, "subscribe", []any{subsCopy[i].Topic}); err != nil {
			c.logger.Warn("ws: resubscribe failed",
				bblog.Str("topic", subsCopy[i].Topic),
				bblog.Err(err),
			)
		}
	}

	var loopCtx context.Context
	var loopCancel context.CancelFunc
	loopCtx, loopCancel = context.WithCancel(ctx)
	defer loopCancel()

	var wg sync.WaitGroup
	wg.Add(2)
	var readErr error
	go func() {
		defer wg.Done()
		defer loopCancel()
		readErr = c.readLoop(loopCtx, socket)
	}()
	go func() {
		defer wg.Done()
		c.pingLoop(loopCtx, socket)
	}()
	wg.Wait()

	if readErr != nil {
		return readErr
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return nil
}

// performAuth sends {"op":"auth","args":[apiKey, expires, signature]} and
// waits for the matching {"op":"auth","success":true} reply.
func (c *Conn) performAuth(socket *websocket.Conn) error {
	if c.signer == nil || !c.signer.Enabled() {
		return errors.New("ws: private endpoint requires signer with credentials")
	}
	var nowMs int64 = time.Now().UnixMilli()
	var expiresMs int64 = nowMs + int64(c.cfg.AuthExpiresWindow/time.Millisecond)
	var expiresStr string = strconv.FormatInt(expiresMs, 10)
	var signature string
	var err error
	signature, err = c.signer.SignWS(expiresStr)
	if err != nil {
		return err
	}
	// Bybit expects expires as a JSON number, not a string.
	var op OutboundOp = OutboundOp{
		Op:   "auth",
		Args: []any{c.signer.APIKey(), expiresMs, signature},
	}
	var raw []byte
	raw, err = codec.Marshal(op)
	if err != nil {
		return err
	}
	if err = c.writeFrame(socket, raw); err != nil {
		return err
	}

	var deadline time.Time = time.Now().Add(c.cfg.AuthTimeout)
	_ = socket.SetReadDeadline(deadline)
	defer func() { _ = socket.SetReadDeadline(time.Time{}) }()

	// Read up to 10 frames: pongs / push frames may arrive before the auth
	// ack on a busy connection.
	var i int
	for i = 0; i < 10; i++ {
		var msgType int
		var raw []byte
		msgType, raw, err = socket.ReadMessage()
		if err != nil {
			return err
		}
		if msgType != websocket.TextMessage {
			continue
		}
		var env Envelope
		if err = codec.Unmarshal(raw, &env); err != nil {
			continue
		}
		if env.Op != "auth" {
			continue
		}
		if env.Success != nil && *env.Success {
			c.logger.Info("ws: auth ok", bblog.Str("connId", env.ConnID))
			return nil
		}
		return fmt.Errorf("auth rejected: ret_msg=%q", env.RetMsg)
	}
	return errors.New("ws: auth ack not received")
}

// readLoop reads frames and dispatches push frames to subscription handlers.
// Returns the read error so supervise can decide whether to reconnect.
func (c *Conn) readLoop(ctx context.Context, socket *websocket.Conn) error {
	for {
		if ctx.Err() != nil {
			return nil
		}
		_ = socket.SetReadDeadline(time.Now().Add(c.cfg.ReadTimeout))
		var msgType int
		var raw []byte
		var err error
		msgType, raw, err = socket.ReadMessage()
		if err != nil {
			return err
		}
		if msgType != websocket.TextMessage {
			continue
		}
		c.cReceived.Inc()

		var env Envelope
		if err = codec.Unmarshal(raw, &env); err != nil {
			c.cDropped.Inc()
			c.logger.Warn("ws: failed to parse envelope", bblog.Err(err))
			continue
		}

		if env.IsControl() {
			c.handleControl(&env)
			continue
		}
		if !env.IsPush() {
			c.cDropped.Inc()
			continue
		}

		c.mu.RLock()
		var sub *Subscription = c.subs[env.Topic]
		c.mu.RUnlock()
		if sub == nil {
			// Topic-prefix fallback: private channels send "execution.linear"
			// while the user may have subscribed to "execution". Try one
			// progressively shorter prefix before dropping. We only do one
			// dot strip — deeper structures should subscribe with the exact
			// topic.
			var dot int
			for dot = len(env.Topic) - 1; dot >= 0; dot-- {
				if env.Topic[dot] == '.' {
					break
				}
			}
			if dot > 0 {
				c.mu.RLock()
				sub = c.subs[env.Topic[:dot]]
				c.mu.RUnlock()
			}
		}
		if sub == nil {
			c.cDropped.Inc()
			continue
		}
		sub.Handler(env.Topic, env.Data)
	}
}

// handleControl logs ack frames and counts authentication outcomes that
// were not consumed by the synchronous performAuth path (e.g. auth that
// arrives unexpectedly mid-stream).
func (c *Conn) handleControl(env *Envelope) {
	switch env.Op {
	case "subscribe":
		c.logger.Debug("ws: subscribed",
			bblog.Str("connId", env.ConnID),
			bblog.Str("ret_msg", env.RetMsg),
		)
	case "unsubscribe":
		c.logger.Debug("ws: unsubscribed",
			bblog.Str("connId", env.ConnID),
		)
	case "auth":
		// Stale auth ack arriving outside performAuth — log only.
		c.logger.Debug("ws: auth ack",
			bblog.Str("connId", env.ConnID),
			bblog.Str("ret_msg", env.RetMsg),
		)
	case "pong":
		// Application-level pong; just resets the read deadline implicitly.
	default:
		c.logger.Debug("ws: control",
			bblog.Str("op", env.Op),
			bblog.Str("ret_msg", env.RetMsg),
		)
	}
}

// pingLoop sends {"op":"ping"} on a ticker. Exits on the first write error;
// the read-loop will fail too and supervise will reconnect.
func (c *Conn) pingLoop(ctx context.Context, socket *websocket.Conn) {
	if c.cfg.PingInterval <= 0 {
		return
	}
	var ticker *time.Ticker = time.NewTicker(c.cfg.PingInterval)
	defer ticker.Stop()
	var pingPayload []byte = []byte(`{"op":"ping"}`)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := c.writeFrame(socket, pingPayload); err != nil {
				c.cPingErr.Inc()
				c.logger.Debug("ws: ping write failed", bblog.Err(err))
				return
			}
		}
	}
}

// sendOp marshals an op JSON ({"op":..., "args":[...]}) and writes it.
func (c *Conn) sendOp(socket *websocket.Conn, op string, args []any) error {
	var msg OutboundOp = OutboundOp{Op: op, Args: args}
	var raw []byte
	var err error
	raw, err = codec.Marshal(msg)
	if err != nil {
		return err
	}
	return c.writeFrame(socket, raw)
}

// writeFrame is a thread-safe text-frame write. gorilla/websocket requires
// exclusive writes — the dedicated mutex keeps ping/sub/auth from
// stepping on each other.
func (c *Conn) writeFrame(socket *websocket.Conn, data []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	_ = socket.SetWriteDeadline(time.Now().Add(c.cfg.WriteTimeout))
	return socket.WriteMessage(websocket.TextMessage, data)
}

// nextBackoff doubles cur, capping at max.
func nextBackoff(cur, max time.Duration) time.Duration {
	cur *= 2
	if cur > max {
		cur = max
	}
	return cur
}

// applyJitter multiplies d by a random factor in [1-j, 1+j].
func applyJitter(d time.Duration, jitter float64) time.Duration {
	if jitter <= 0 {
		return d
	}
	var f float64 = 1.0 + (rand.Float64()*2.0-1.0)*jitter
	return time.Duration(float64(d) * f)
}
