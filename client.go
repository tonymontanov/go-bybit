/*
FILE: client.go

DESCRIPTION:
The root SDK client. Holds shared resources (REST transport, signer,
config, logger) and exposes lazy domain sub-clients on demand. Domain
profiles (linears, spot, ...) are implemented in their own packages and
register a factory at init() time so the root package never imports them
directly (avoids a circular dependency: domain packages import the root
for Config/Error/etc.).

USAGE:

	var cfg bybit.Config = bybit.DefaultConfig()
	cfg.APIKey = "..."
	cfg.SecretKey = "..."
	var c, err = bybit.NewClient(cfg)
	if err != nil { panic(err) }
	defer c.Close()

	// Once the linears package is imported (anonymously is fine):
	//   import _ "github.com/tonymontanov/go-bybit/v2/linears"
	var linearsClient = c.Linears().(*linears.Client)

The .(*linears.Client) cast is by design: the root package returns `any`
because it cannot know about the linears.Client type without importing
the linears package (which already imports root). The cast is a single
line and keeps the SDK structure flat.
*/

package bybit

import (
	"sync"

	"github.com/tonymontanov/go-bybit/v2/internal/auth"
	"github.com/tonymontanov/go-bybit/v2/internal/bberr"
	"github.com/tonymontanov/go-bybit/v2/internal/rest"
)

// Client is the root SDK object. Safe for concurrent use; methods on Client
// itself are stateless apart from the lazy sub-client cache.
type Client struct {
	cfg    Config
	signer *auth.Signer
	rest   *rest.Client
	logger Logger

	linearsOnce sync.Once
	linearsVal  any

	spotOnce sync.Once
	spotVal  any

	assetOnce sync.Once
	assetVal  any

	accountOnce sync.Once
	accountVal  any

	brokerOnce sync.Once
	brokerVal  any

	affiliateOnce sync.Once
	affiliateVal  any

	preMarketOnce sync.Once
	preMarketVal  any
}

// NewClient validates cfg, fills defaults, and returns a configured root
// client. Returns an *Error with ErrorKindInvalidRequest on configuration
// problems.
func NewClient(cfg Config) (*Client, error) {
	cfg = cfg.withDefaults()
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	var signer *auth.Signer = auth.NewSigner(cfg.APIKey, cfg.SecretKey)

	var restCfg rest.Config = rest.Config{
		RequestTimeout:      cfg.REST.RequestTimeout,
		MaxIdleConns:        cfg.REST.MaxIdleConns,
		MaxIdleConnsPerHost: cfg.REST.MaxIdleConnsPerHost,
		IdleConnTimeout:     cfg.REST.IdleConnTimeout,
		RecvWindow:          cfg.REST.RecvWindow,
		RateLimitObserver:   cfg.RateLimitObserver,
	}
	// Forward the typed event observer through a thin adapter. The
	// public RateLimitEvent struct lives in the root package and CANNOT
	// be passed directly into internal/rest (import cycle). The transport
	// invokes the callback with flat arguments and we assemble
	// RateLimitEvent here.
	if cfg.RateLimitEventObserver != nil {
		var userObserver = cfg.RateLimitEventObserver
		restCfg.RateLimitEventObserver = func(endpoint, method string, headers map[string]string, meta rest.RequestMeta) {
			userObserver(RateLimitEvent{
				Endpoint:   endpoint,
				Method:     method,
				Headers:    headers,
				OrderCount: meta.OrderCount,
				Symbols:    meta.Symbols,
				Category:   RateLimitCategory(meta.Category),
			})
		}
	}

	// Adapter: bblog.Logger ←→ public Logger. Aliased types make this a
	// pure-cost-free identity at the type level — both are bblog.Logger.
	var transportLogger = cfg.Logger
	var restClient *rest.Client = rest.NewClient(cfg.REST.BaseURL, signer, restCfg, cfg.UserAgent, transportLogger)

	return &Client{
		cfg:    cfg,
		signer: signer,
		rest:   restClient,
		logger: cfg.Logger,
	}, nil
}

// Config returns a copy of the resolved Config (after defaults applied).
func (c *Client) Config() Config { return c.cfg }

// Logger returns the configured logger. Useful for the same logger to be
// reused by a desk-side adapter.
func (c *Client) Logger() Logger { return c.logger }

// Signer is exposed to internal SDK sub-packages (linears, spot) so they
// can sign WS auth payloads. User code SHOULD NOT touch it.
func (c *Client) Signer() *auth.Signer { return c.signer }

// REST is exposed to internal SDK sub-packages.
func (c *Client) REST() *rest.Client { return c.rest }

// Close releases idle HTTP connections. WS connections owned by domain
// sub-clients close on their own contexts.
func (c *Client) Close() error {
	if c == nil {
		return nil
	}
	c.rest.Close()
	return nil
}

// ----------------------------------------------------------------------------
// Sub-client factories (registered by domain packages via init).
// ----------------------------------------------------------------------------

// linearsFactory is set by linears.init() via RegisterLinearsFactory.
var linearsFactory func(c *Client) any

// RegisterLinearsFactory wires the linears.Client builder. Idempotent —
// only the first call is honoured.
func RegisterLinearsFactory(f func(c *Client) any) {
	if linearsFactory == nil {
		linearsFactory = f
	}
}

// Linears returns the *linears.Client (typed as any for import-cycle
// reasons). nil when the linears package has not been imported.
func (c *Client) Linears() any {
	c.linearsOnce.Do(func() {
		if linearsFactory == nil {
			c.logger.Warn(`bybit.Client.Linears: linears factory is not registered; import _ "github.com/tonymontanov/go-bybit/v2/linears"`)
			return
		}
		c.linearsVal = linearsFactory(c)
	})
	return c.linearsVal
}

// spotFactory is set by spot.init() via RegisterSpotFactory.
var spotFactory func(c *Client) any

// RegisterSpotFactory wires the spot.Client builder. Idempotent.
func RegisterSpotFactory(f func(c *Client) any) {
	if spotFactory == nil {
		spotFactory = f
	}
}

// Spot returns the *spot.Client (typed as any). nil when the spot package
// has not been imported.
func (c *Client) Spot() any {
	c.spotOnce.Do(func() {
		if spotFactory == nil {
			c.logger.Warn(`bybit.Client.Spot: spot factory is not registered; import _ "github.com/tonymontanov/go-bybit/v2/spot"`)
			return
		}
		c.spotVal = spotFactory(c)
	})
	return c.spotVal
}

// assetFactory is set by asset.init() via RegisterAssetFactory.
var assetFactory func(c *Client) any

// RegisterAssetFactory wires the asset.Client builder. Idempotent.
func RegisterAssetFactory(f func(c *Client) any) {
	if assetFactory == nil {
		assetFactory = f
	}
}

// Asset returns the *asset.Client (typed as any). nil when the asset
// package has not been imported.
func (c *Client) Asset() any {
	c.assetOnce.Do(func() {
		if assetFactory == nil {
			c.logger.Warn(`bybit.Client.Asset: asset factory is not registered; import _ "github.com/tonymontanov/go-bybit/v2/asset"`)
			return
		}
		c.assetVal = assetFactory(c)
	})
	return c.assetVal
}

// accountFactory is set by account.init() via RegisterAccountFactory.
var accountFactory func(c *Client) any

// RegisterAccountFactory wires the account.Client builder. Idempotent.
func RegisterAccountFactory(f func(c *Client) any) {
	if accountFactory == nil {
		accountFactory = f
	}
}

// Account returns the *account.Client (typed as any). nil when the account
// package has not been imported.
func (c *Client) Account() any {
	c.accountOnce.Do(func() {
		if accountFactory == nil {
			c.logger.Warn(`bybit.Client.Account: account factory is not registered; import _ "github.com/tonymontanov/go-bybit/v2/account"`)
			return
		}
		c.accountVal = accountFactory(c)
	})
	return c.accountVal
}

// brokerFactory is set by broker.init() via RegisterBrokerFactory.
var brokerFactory func(c *Client) any

// RegisterBrokerFactory wires the broker.Client builder. Idempotent.
func RegisterBrokerFactory(f func(c *Client) any) {
	if brokerFactory == nil {
		brokerFactory = f
	}
}

// Broker returns the *broker.Client (typed as any). nil when the broker
// package has not been imported.
func (c *Client) Broker() any {
	c.brokerOnce.Do(func() {
		if brokerFactory == nil {
			c.logger.Warn(`bybit.Client.Broker: broker factory is not registered; import _ "github.com/tonymontanov/go-bybit/v2/broker"`)
			return
		}
		c.brokerVal = brokerFactory(c)
	})
	return c.brokerVal
}

// affiliateFactory is set by affiliate.init() via RegisterAffiliateFactory.
var affiliateFactory func(c *Client) any

// RegisterAffiliateFactory wires the affiliate.Client builder. Idempotent.
func RegisterAffiliateFactory(f func(c *Client) any) {
	if affiliateFactory == nil {
		affiliateFactory = f
	}
}

// Affiliate returns the *affiliate.Client (typed as any). nil when the
// affiliate package has not been imported.
func (c *Client) Affiliate() any {
	c.affiliateOnce.Do(func() {
		if affiliateFactory == nil {
			c.logger.Warn(`bybit.Client.Affiliate: affiliate factory is not registered; import _ "github.com/tonymontanov/go-bybit/v2/affiliate"`)
			return
		}
		c.affiliateVal = affiliateFactory(c)
	})
	return c.affiliateVal
}

// preMarketFactory is set by premarket.init() via RegisterPreMarketFactory.
var preMarketFactory func(c *Client) any

// RegisterPreMarketFactory wires the premarket.Client builder. Idempotent.
func RegisterPreMarketFactory(f func(c *Client) any) {
	if preMarketFactory == nil {
		preMarketFactory = f
	}
}

// PreMarket returns the *premarket.Client (typed as any). nil when the
// premarket package has not been imported.
func (c *Client) PreMarket() any {
	c.preMarketOnce.Do(func() {
		if preMarketFactory == nil {
			c.logger.Warn(`bybit.Client.PreMarket: premarket factory is not registered; import _ "github.com/tonymontanov/go-bybit/v2/premarket"`)
			return
		}
		c.preMarketVal = preMarketFactory(c)
	})
	return c.preMarketVal
}

// Compile-time assertion: *Error implements the error interface.
var _ error = (*bberr.Error)(nil)
