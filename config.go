/*
FILE: config.go

DESCRIPTION:
Public SDK configuration — REST/WS endpoints, timeouts, reconnect policy,
orderbook tuning, observer hooks. Default values match the production
Bybit V5 endpoints and conservative HFT-friendly timeouts.

ENDPOINTS (defaults; Config.Testnet / Config.Demo pick the environment):

  Mainnet:
    REST:                 https://api.bybit.com
    WS public (linears):  wss://stream.bybit.com/v5/public/linear
    WS public (spot):     wss://stream.bybit.com/v5/public/spot
    WS public (inverse):  wss://stream.bybit.com/v5/public/inverse
    WS public (option):   wss://stream.bybit.com/v5/public/option
    WS private:           wss://stream.bybit.com/v5/private

  Testnet: the same paths on api-testnet.bybit.com (REST) and
  stream-testnet.bybit.com (every public category and the private stream).

  Demo Trading: REST https://api-demo.bybit.com, WS private
  wss://stream-demo.bybit.com/v5/private. Demo has NO public stream of its
  own — stream-demo.bybit.com answers a handshake on /v5/public/* with
  404 — so the public WS stays on the production host stream.bybit.com
  (https://bybit-exchange.github.io/docs/v5/demo).

A SINGLE WS PRIVATE ENDPOINT serves every category (linear/spot/inverse/
option) — auth is per-UID, not per-category, so we do not split it by
profile. The public endpoints differ per category, which is why the
domain client (linears.Client / spot.Client) picks the right URL when it
opens the public connection.

URL RESOLUTION (NewClient): a URL field that is empty, or still holds the
production default DefaultConfig put there, gets the selected
environment's default; any other value is an explicit override (mock
server, custom gateway) and is kept verbatim. So Config{Demo: true} and
DefaultConfig() with Demo = true resolve to the same demo endpoints.
*/

package bybit

import "time"

// Bybit V5 endpoints. Declared as vars so tests can override them
// (e.g. point at a mock server).
var (
	// DefaultRestBaseURL — production REST endpoint.
	DefaultRestBaseURL string = "https://api.bybit.com"

	// TestnetRestBaseURL — public testnet REST endpoint.
	TestnetRestBaseURL string = "https://api-testnet.bybit.com"

	// DemoRestBaseURL — paper-trading (Demo) REST endpoint. Demo and
	// production keys are NOT interchangeable; demo keys are issued
	// separately in the Bybit web UI.
	DemoRestBaseURL string = "https://api-demo.bybit.com"

	// DefaultWsPublicLinearURL — production public WS for linear category.
	DefaultWsPublicLinearURL string = "wss://stream.bybit.com/v5/public/linear"

	// DefaultWsPublicSpotURL — production public WS for spot category.
	DefaultWsPublicSpotURL string = "wss://stream.bybit.com/v5/public/spot"

	// DefaultWsPublicInverseURL — production public WS for inverse category.
	DefaultWsPublicInverseURL string = "wss://stream.bybit.com/v5/public/inverse"

	// DefaultWsPublicOptionURL — production public WS for option category.
	DefaultWsPublicOptionURL string = "wss://stream.bybit.com/v5/public/option"

	// DefaultWsPrivateURL — production private WS (auth required).
	DefaultWsPrivateURL string = "wss://stream.bybit.com/v5/private"

	// TestnetWsPublicLinearURL — testnet public WS for linear category.
	TestnetWsPublicLinearURL string = "wss://stream-testnet.bybit.com/v5/public/linear"

	// TestnetWsPublicSpotURL — testnet public WS for spot category.
	TestnetWsPublicSpotURL string = "wss://stream-testnet.bybit.com/v5/public/spot"

	// TestnetWsPublicInverseURL — testnet public WS for inverse category.
	TestnetWsPublicInverseURL string = "wss://stream-testnet.bybit.com/v5/public/inverse"

	// TestnetWsPublicOptionURL — testnet public WS for option category.
	TestnetWsPublicOptionURL string = "wss://stream-testnet.bybit.com/v5/public/option"

	// TestnetWsPrivateURL — testnet private WS.
	TestnetWsPrivateURL string = "wss://stream-testnet.bybit.com/v5/private"

	// DemoWsPrivateURL — Demo Trading private WS. Demo does NOT have a
	// public stream: stream-demo.bybit.com answers /v5/public/* with 404,
	// and public market data for demo accounts comes from the production
	// hosts (DefaultWsPublic*URL).
	DemoWsPrivateURL string = "wss://stream-demo.bybit.com/v5/private"
)

// Config — public SDK configuration. Pass to NewClient.
type Config struct {
	// APIKey — Bybit V5 public key. Required for signed endpoints; safe
	// to leave empty for public-only access.
	APIKey string
	// SecretKey — Bybit V5 secret used to compute X-BAPI-SIGN.
	SecretKey string

	// REST — REST transport settings. Empty fields fall back to defaults.
	REST RestConfig
	// WS — WebSocket transport settings. Empty fields fall back to defaults.
	WS WsConfig
	// Orderbook — orderbook engine settings. Empty fields fall back to
	// defaults.
	Orderbook OrderbookConfig

	// Logger — optional logger. NoopLogger if nil.
	Logger Logger
	// Metrics — optional counter factory. NoopMetrics if nil.
	Metrics CounterFactory

	// UserAgent — User-Agent value sent on REST requests. Default
	// "go-bybit/1".
	UserAgent string

	// RateLimitObserver — legacy observer (endpoint, headers). Kept for
	// source-level back-compat with the OKX-style pattern. nil → no-op.
	RateLimitObserver func(endpoint string, headers map[string]string)

	// RateLimitEventObserver — primary observer. Receives the full
	// RateLimitEvent with OrderCount/Symbols/Category/Headers.
	//
	// Speed contract: called synchronously from the goroutine that issued
	// the REST call. Implementations must be O(1) (typically a
	// non-blocking send to a buffered channel).
	//
	// nil → no-op.
	RateLimitEventObserver func(RateLimitEvent)

	// Testnet — switches the default REST host and every WS host (all
	// public categories and the private stream) to testnet. URLs set
	// explicitly to a non-production value are kept. Takes precedence
	// over Demo when both are set. Default false.
	Testnet bool

	// Demo — switches the default REST host to api-demo.bybit.com and the
	// private WS host to stream-demo.bybit.com. Public WS stays on the
	// production host stream.bybit.com: Demo Trading has no public stream
	// of its own. URLs set explicitly to a non-production value are kept.
	// Default false.
	Demo bool
}

// RestConfig — REST transport parameters.
type RestConfig struct {
	// BaseURL — REST host. Default DefaultRestBaseURL.
	BaseURL string
	// RequestTimeout — global timeout for one REST call. Default 10s.
	// A ctx with its own deadline overrides this for a single request.
	RequestTimeout time.Duration
	// MaxIdleConns — http.Transport pool size. Default 100.
	MaxIdleConns int
	// MaxIdleConnsPerHost — per-host pool size. Default 100.
	MaxIdleConnsPerHost int
	// IdleConnTimeout — keep-alive idle timeout. Default 90s.
	IdleConnTimeout time.Duration
	// RecvWindow — value for X-BAPI-RECV-WINDOW (ms). Bybit rejects
	// requests where (server_time - timestamp) > RecvWindow. Default 5000.
	// Increase to ~10000 for trading from a high-latency VPN setup.
	RecvWindow int
}

// WsConfig — WebSocket transport parameters.
type WsConfig struct {
	// PublicLinearURL / PublicSpotURL / PublicInverseURL / PublicOptionURL /
	// PrivateURL — endpoint URLs. Empty values, and values equal to the
	// production defaults, resolve to the production / testnet / demo
	// defaults based on Config.Testnet / Config.Demo (see the ENDPOINTS
	// table at the top of this file).
	PublicLinearURL  string
	PublicSpotURL    string
	PublicInverseURL string
	PublicOptionURL  string
	PrivateURL       string

	// HandshakeTimeout — TLS+HTTP upgrade timeout. Default 10s.
	HandshakeTimeout time.Duration
	// ReadTimeout — read deadline. Default 35s; the server's idle
	// timeout is 20s, so this gives one full ping cycle of slack.
	ReadTimeout time.Duration
	// WriteTimeout — write deadline. Default 5s.
	WriteTimeout time.Duration
	// PingInterval — interval between application-level {"op":"ping"}
	// frames. Default 20s. Bybit's server-side idle timeout is 20s by
	// default; pinging at exactly the same period is the documented
	// recipe.
	PingInterval time.Duration

	// AuthExpiresWindow — `expires` window in the WS auth payload.
	// Default 1s.
	AuthExpiresWindow time.Duration
	// AuthTimeout — how long to wait for the auth ack. Default 3s.
	AuthTimeout time.Duration

	// ReconnectInitialBackoff — first sleep after a connection failure.
	// Default 200ms.
	ReconnectInitialBackoff time.Duration
	// ReconnectMaxBackoff — backoff cap. Default 10s.
	ReconnectMaxBackoff time.Duration
	// ReconnectJitter — relative jitter [0..1] applied to backoff.
	// Default 0.2.
	ReconnectJitter float64

	// ReadBufferSize / WriteBufferSize — gorilla/websocket buffer sizes.
	// Defaults: 64KB / 16KB.
	ReadBufferSize  int
	WriteBufferSize int
}

// OrderbookConfig — orderbook engine parameters. Used by the M2 engine
// (added in a later milestone); settings are exposed in M0 so the public
// surface is stable from the start.
type OrderbookConfig struct {
	// MaxDepth — depth of the local order book per side. Default 200
	// (Bybit linear publishes orderbook.50 / orderbook.200 / orderbook.500).
	MaxDepth int
}

// DefaultConfig returns a Config pre-populated with production endpoints
// and HFT-friendly timeouts. Callers can override individual fields and
// pass the result to NewClient — empty sub-fields fall back to these
// defaults. Setting Testnet / Demo on the result switches the endpoints
// too: NewClient treats the production URLs left here as defaults, not as
// explicit overrides.
func DefaultConfig() Config {
	return Config{
		REST: RestConfig{
			BaseURL:             DefaultRestBaseURL,
			RequestTimeout:      10 * time.Second,
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout:     90 * time.Second,
			RecvWindow:          5000,
		},
		WS: WsConfig{
			PublicLinearURL:         DefaultWsPublicLinearURL,
			PublicSpotURL:           DefaultWsPublicSpotURL,
			PublicInverseURL:        DefaultWsPublicInverseURL,
			PublicOptionURL:         DefaultWsPublicOptionURL,
			PrivateURL:              DefaultWsPrivateURL,
			HandshakeTimeout:        10 * time.Second,
			ReadTimeout:             35 * time.Second,
			WriteTimeout:            5 * time.Second,
			PingInterval:            20 * time.Second,
			AuthExpiresWindow:       time.Second,
			AuthTimeout:             3 * time.Second,
			ReconnectInitialBackoff: 200 * time.Millisecond,
			ReconnectMaxBackoff:     10 * time.Second,
			ReconnectJitter:         0.2,
			ReadBufferSize:          64 * 1024,
			WriteBufferSize:         16 * 1024,
		},
		Orderbook: OrderbookConfig{
			MaxDepth: 200,
		},
		Logger:    NoopLogger(),
		Metrics:   NoopMetrics(),
		UserAgent: "go-bybit/1",
	}
}

// withDefaults returns a copy of c with empty fields filled from
// DefaultConfig. The Testnet / Demo flags pick the environment whose
// endpoints fill the URL fields; a URL still equal to the production
// default counts as unset (see resolveEndpoint), so DefaultConfig() with
// Demo = true resolves exactly like Config{Demo: true}. Explicit
// non-production URLs are preserved.
func (c Config) withDefaults() Config {
	var def Config = DefaultConfig()
	var env environmentEndpoints = endpointsFor(c.Testnet, c.Demo)

	// REST.
	c.REST.BaseURL = resolveEndpoint(c.REST.BaseURL, def.REST.BaseURL, env.rest)
	if c.REST.RequestTimeout == 0 {
		c.REST.RequestTimeout = def.REST.RequestTimeout
	}
	if c.REST.MaxIdleConns == 0 {
		c.REST.MaxIdleConns = def.REST.MaxIdleConns
	}
	if c.REST.MaxIdleConnsPerHost == 0 {
		c.REST.MaxIdleConnsPerHost = def.REST.MaxIdleConnsPerHost
	}
	if c.REST.IdleConnTimeout == 0 {
		c.REST.IdleConnTimeout = def.REST.IdleConnTimeout
	}
	if c.REST.RecvWindow == 0 {
		c.REST.RecvWindow = def.REST.RecvWindow
	}

	// WS.
	c.WS.PublicLinearURL = resolveEndpoint(c.WS.PublicLinearURL, def.WS.PublicLinearURL, env.publicLinear)
	c.WS.PublicSpotURL = resolveEndpoint(c.WS.PublicSpotURL, def.WS.PublicSpotURL, env.publicSpot)
	c.WS.PublicInverseURL = resolveEndpoint(c.WS.PublicInverseURL, def.WS.PublicInverseURL, env.publicInverse)
	c.WS.PublicOptionURL = resolveEndpoint(c.WS.PublicOptionURL, def.WS.PublicOptionURL, env.publicOption)
	c.WS.PrivateURL = resolveEndpoint(c.WS.PrivateURL, def.WS.PrivateURL, env.private)
	if c.WS.HandshakeTimeout == 0 {
		c.WS.HandshakeTimeout = def.WS.HandshakeTimeout
	}
	if c.WS.ReadTimeout == 0 {
		c.WS.ReadTimeout = def.WS.ReadTimeout
	}
	if c.WS.WriteTimeout == 0 {
		c.WS.WriteTimeout = def.WS.WriteTimeout
	}
	if c.WS.PingInterval == 0 {
		c.WS.PingInterval = def.WS.PingInterval
	}
	if c.WS.AuthExpiresWindow == 0 {
		c.WS.AuthExpiresWindow = def.WS.AuthExpiresWindow
	}
	if c.WS.AuthTimeout == 0 {
		c.WS.AuthTimeout = def.WS.AuthTimeout
	}
	if c.WS.ReconnectInitialBackoff == 0 {
		c.WS.ReconnectInitialBackoff = def.WS.ReconnectInitialBackoff
	}
	if c.WS.ReconnectMaxBackoff == 0 {
		c.WS.ReconnectMaxBackoff = def.WS.ReconnectMaxBackoff
	}
	if c.WS.ReconnectJitter == 0 {
		c.WS.ReconnectJitter = def.WS.ReconnectJitter
	}
	if c.WS.ReadBufferSize == 0 {
		c.WS.ReadBufferSize = def.WS.ReadBufferSize
	}
	if c.WS.WriteBufferSize == 0 {
		c.WS.WriteBufferSize = def.WS.WriteBufferSize
	}

	if c.Orderbook.MaxDepth == 0 {
		c.Orderbook.MaxDepth = def.Orderbook.MaxDepth
	}

	if c.Logger == nil {
		c.Logger = NoopLogger()
	}
	if c.Metrics == nil {
		c.Metrics = NoopMetrics()
	}
	if c.UserAgent == "" {
		c.UserAgent = def.UserAgent
	}

	return c
}

// environmentEndpoints — the default REST/WS endpoints of one Bybit
// environment (mainnet, testnet or Demo Trading).
type environmentEndpoints struct {
	rest          string
	publicLinear  string
	publicSpot    string
	publicInverse string
	publicOption  string
	private       string
}

// endpointsFor returns the default endpoints of the environment selected
// by the Testnet / Demo flags. Testnet takes precedence when both are set,
// so REST and WS always come from the same environment.
func endpointsFor(testnet bool, demo bool) environmentEndpoints {
	switch {
	case testnet:
		return environmentEndpoints{
			rest:          TestnetRestBaseURL,
			publicLinear:  TestnetWsPublicLinearURL,
			publicSpot:    TestnetWsPublicSpotURL,
			publicInverse: TestnetWsPublicInverseURL,
			publicOption:  TestnetWsPublicOptionURL,
			private:       TestnetWsPrivateURL,
		}
	case demo:
		// Demo Trading has only a private stream: stream-demo.bybit.com
		// answers /v5/public/* with 404, so public market data stays on
		// the production hosts. REST and private WS are demo-specific.
		return environmentEndpoints{
			rest:          DemoRestBaseURL,
			publicLinear:  DefaultWsPublicLinearURL,
			publicSpot:    DefaultWsPublicSpotURL,
			publicInverse: DefaultWsPublicInverseURL,
			publicOption:  DefaultWsPublicOptionURL,
			private:       DemoWsPrivateURL,
		}
	default:
		return environmentEndpoints{
			rest:          DefaultRestBaseURL,
			publicLinear:  DefaultWsPublicLinearURL,
			publicSpot:    DefaultWsPublicSpotURL,
			publicInverse: DefaultWsPublicInverseURL,
			publicOption:  DefaultWsPublicOptionURL,
			private:       DefaultWsPrivateURL,
		}
	}
}

// resolveEndpoint returns envDefault when current is empty or still equal
// to the production default (the value DefaultConfig pre-populates), and
// current otherwise. Without the second case, DefaultConfig() with
// Testnet / Demo = true kept every production URL and silently talked to
// mainnet with testnet / demo keys.
func resolveEndpoint(current string, productionDefault string, envDefault string) string {
	if current == "" || current == productionDefault {
		return envDefault
	}
	return current
}

// validate ensures the minimal set of required fields is present.
// Credentials are NOT enforced here — public endpoints work without keys
// and the signer surfaces auth.ErrSignerDisabled at call time.
func (c Config) validate() error {
	if c.REST.BaseURL == "" {
		return NewError(ErrorKindInvalidRequest, "", "config: REST.BaseURL is empty", nil)
	}
	return nil
}
