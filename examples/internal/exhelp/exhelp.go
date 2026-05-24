/*
FILE: examples/internal/exhelp/exhelp.go

DESCRIPTION:
Tiny shared helpers for the go-bybit examples — environment loading,
SDK config bootstrap, error formatting. Kept under examples/internal/ so
external users do not depend on it.

Functions:

  - Classify(err)        : human-readable rendering of *bybit.Error.
  - LoadEnv(...)          : reads BYBIT_* env vars, applies defaults.
  - NewClient(opt) (...)  : builds a fully-configured bybit.Client + the
                            linears.Client cast in one call.

These three building blocks let every example focus on the actual SDK
demonstration code and stay short.
*/

package exhelp

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/shopspring/decimal"
	bybit "github.com/tonymontanov/go-bybit/v2"
	"github.com/tonymontanov/go-bybit/v2/account"
	"github.com/tonymontanov/go-bybit/v2/affiliate"
	"github.com/tonymontanov/go-bybit/v2/asset"
	"github.com/tonymontanov/go-bybit/v2/broker"
	"github.com/tonymontanov/go-bybit/v2/linears"
	"github.com/tonymontanov/go-bybit/v2/premarket"
	"github.com/tonymontanov/go-bybit/v2/spot"
)

// Options — flat input for NewClient. The fields mirror the public
// BYBIT_* env vars so the examples stay self-documenting.
type Options struct {
	APIKey    string
	APISecret string
	Symbol    string
	Quantity  decimal.Decimal
	Testnet   bool
	Demo      bool
	AllowLive bool
	// HoldSeconds — used only by inventory-tracker; harmless elsewhere.
	HoldSeconds int64
}

// LoadEnv reads BYBIT_* variables and returns Options. Uses provided
// defaults when env values are missing or empty.
//
// Defaults:
//
//   - Symbol      : "BTCUSDT"
//   - Quantity    : 0.001
//   - HoldSeconds : 5
//
// LoadEnv does not validate — examples decide which fields are mandatory.
func LoadEnv() Options {
	var opt Options

	opt.APIKey = os.Getenv("BYBIT_API_KEY")
	opt.APISecret = os.Getenv("BYBIT_API_SECRET")

	opt.Symbol = os.Getenv("BYBIT_SYMBOL")
	if opt.Symbol == "" {
		opt.Symbol = "BTCUSDT"
	}

	opt.Quantity = decimal.RequireFromString("0.001")
	if v := os.Getenv("BYBIT_QUANTITY"); v != "" {
		var q, err = decimal.NewFromString(v)
		if err != nil {
			log.Fatalf("BYBIT_QUANTITY=%q: %v", v, err)
		}
		opt.Quantity = q
	}

	opt.HoldSeconds = 5
	if v := os.Getenv("BYBIT_HOLD_SECONDS"); v != "" {
		var n, err = strconv.ParseInt(v, 10, 64)
		if err != nil {
			log.Fatalf("BYBIT_HOLD_SECONDS=%q: %v", v, err)
		}
		opt.HoldSeconds = n
	}

	opt.Testnet = os.Getenv("BYBIT_TESTNET") == "1"
	opt.Demo = os.Getenv("BYBIT_DEMO") == "1"
	opt.AllowLive = os.Getenv("BYBIT_ALLOW_LIVE") == "1"

	return opt
}

// MustHaveKeys terminates the process with a clear message when API
// credentials are not configured. Used by every example that needs to
// sign anything.
func MustHaveKeys(opt Options) {
	if opt.APIKey == "" || opt.APISecret == "" {
		log.Fatal("BYBIT_API_KEY and BYBIT_API_SECRET must be set (see .env.example)")
	}
}

// MustAllowLive forbids running trading-side examples against production
// without an explicit opt-in. Bypassed when Testnet or Demo is enabled.
func MustAllowLive(opt Options) {
	if opt.Testnet || opt.Demo {
		return
	}
	if !opt.AllowLive {
		log.Fatal("refusing to trade against PRODUCTION: set BYBIT_ALLOW_LIVE=1 (or use BYBIT_TESTNET=1 / BYBIT_DEMO=1)")
	}
}

// NewClient constructs a *bybit.Client using opt and returns the
// linears.Client cast as well, since every example uses both.
//
// Endpoint resolution: if Testnet or Demo is set, Config.{Testnet,Demo}
// is forwarded — the SDK's withDefaults switches REST/WS hosts
// accordingly.
func NewClient(opt Options) (*bybit.Client, *linears.Client) {
	var cfg bybit.Config = bybit.DefaultConfig()
	cfg.APIKey = opt.APIKey
	cfg.SecretKey = opt.APISecret
	cfg.Testnet = opt.Testnet
	cfg.Demo = opt.Demo

	var client, err = bybit.NewClient(cfg)
	if err != nil {
		log.Fatalf("bybit.NewClient: %v", err)
	}

	var lc *linears.Client = client.Linears().(*linears.Client)
	return client, lc
}

// NewSpotClient mirrors NewClient but returns the spot.Client cast.
// Use this in examples that demonstrate the spot profile.
func NewSpotClient(opt Options) (*bybit.Client, *spot.Client) {
	var cfg bybit.Config = bybit.DefaultConfig()
	cfg.APIKey = opt.APIKey
	cfg.SecretKey = opt.APISecret
	cfg.Testnet = opt.Testnet
	cfg.Demo = opt.Demo

	var client, err = bybit.NewClient(cfg)
	if err != nil {
		log.Fatalf("bybit.NewClient: %v", err)
	}
	var sc *spot.Client = client.Spot().(*spot.Client)
	return client, sc
}

// NewAssetClient mirrors NewSpotClient but returns the asset.Client cast.
func NewAssetClient(opt Options) (*bybit.Client, *asset.Client) {
	var cfg bybit.Config = bybit.DefaultConfig()
	cfg.APIKey = opt.APIKey
	cfg.SecretKey = opt.APISecret
	cfg.Testnet = opt.Testnet
	cfg.Demo = opt.Demo

	var client, err = bybit.NewClient(cfg)
	if err != nil {
		log.Fatalf("bybit.NewClient: %v", err)
	}
	var ac *asset.Client = client.Asset().(*asset.Client)
	return client, ac
}

// NewExtendedAccountClient mirrors NewAssetClient but returns account.Client.
func NewExtendedAccountClient(opt Options) (*bybit.Client, *account.Client) {
	var cfg bybit.Config = bybit.DefaultConfig()
	cfg.APIKey = opt.APIKey
	cfg.SecretKey = opt.APISecret
	cfg.Testnet = opt.Testnet
	cfg.Demo = opt.Demo

	var client, err = bybit.NewClient(cfg)
	if err != nil {
		log.Fatalf("bybit.NewClient: %v", err)
	}
	var ac *account.Client = client.Account().(*account.Client)
	return client, ac
}

// NewAccountClient is an alias for NewExtendedAccountClient.
func NewAccountClient(opt Options) (*bybit.Client, *account.Client) {
	return NewExtendedAccountClient(opt)
}

// NewBrokerClient mirrors NewAssetClient but returns broker.Client.
func NewBrokerClient(opt Options) (*bybit.Client, *broker.Client) {
	var cfg bybit.Config = bybit.DefaultConfig()
	cfg.APIKey = opt.APIKey
	cfg.SecretKey = opt.APISecret
	cfg.Testnet = opt.Testnet
	cfg.Demo = opt.Demo

	var client, err = bybit.NewClient(cfg)
	if err != nil {
		log.Fatalf("bybit.NewClient: %v", err)
	}
	var bc *broker.Client = client.Broker().(*broker.Client)
	return client, bc
}

// NewAffiliateClient mirrors NewBrokerClient but returns affiliate.Client.
func NewAffiliateClient(opt Options) (*bybit.Client, *affiliate.Client) {
	var cfg bybit.Config = bybit.DefaultConfig()
	cfg.APIKey = opt.APIKey
	cfg.SecretKey = opt.APISecret
	cfg.Testnet = opt.Testnet
	cfg.Demo = opt.Demo

	var client, err = bybit.NewClient(cfg)
	if err != nil {
		log.Fatalf("bybit.NewClient: %v", err)
	}
	var ac *affiliate.Client = client.Affiliate().(*affiliate.Client)
	return client, ac
}

// NewPreMarketClient mirrors NewClient but returns the premarket.Client cast.
// Public endpoints — API keys are optional.
func NewPreMarketClient(opt Options) (*bybit.Client, *premarket.Client) {
	var cfg bybit.Config = bybit.DefaultConfig()
	cfg.APIKey = opt.APIKey
	cfg.SecretKey = opt.APISecret
	cfg.Testnet = opt.Testnet
	cfg.Demo = opt.Demo

	var client, err = bybit.NewClient(cfg)
	if err != nil {
		log.Fatalf("bybit.NewClient: %v", err)
	}
	var pc *premarket.Client = client.PreMarket().(*premarket.Client)
	return client, pc
}

// Classify renders any error as "[kind retCode=… status=…] message: cause".
// Falls back to err.Error() for non-SDK errors.
func Classify(err error) string {
	if err == nil {
		return "<nil>"
	}
	var e *bybit.Error
	if !errors.As(err, &e) {
		return err.Error()
	}
	if e.Cause != nil {
		return fmt.Sprintf("[%s retCode=%s status=%d] %s: %v", e.Kind, e.BybitCode, e.HTTPStatus, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s retCode=%s status=%d] %s", e.Kind, e.BybitCode, e.HTTPStatus, e.Message)
}
