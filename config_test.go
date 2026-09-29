/*
FILE: config_test.go

DESCRIPTION:
Endpoint resolution per environment (mainnet / testnet / Demo Trading)
through the public NewClient + Client.Config path.

Coverage:
  - mainnet / testnet / demo x REST + every public category + private,
    starting from the zero Config (URLs empty) and from DefaultConfig()
    (URLs pre-filled with production values — the pattern the desk and
    examples/internal/exhelp use).
  - Demo Trading keeps public WS on stream.bybit.com: stream-demo.bybit.com
    serves only /v5/private and answers /v5/public/* with 404
    (https://bybit-exchange.github.io/docs/v5/demo).
  - Testnet + Demo together resolve every endpoint to testnet.
  - Explicit non-production URLs (mock server, gateway) are kept as is.

Expected values are string literals, not the package vars, so a typo in
a var is caught as well. No network calls are made.
*/

package bybit

import "testing"

// endpointSet — the resolved endpoint URLs checked by the tests.
type endpointSet struct {
	rest          string
	publicLinear  string
	publicSpot    string
	publicInverse string
	publicOption  string
	private       string
}

// resolvedEndpoints builds a client from cfg and returns its resolved
// endpoints.
func resolvedEndpoints(t *testing.T, cfg Config) endpointSet {
	t.Helper()
	var client *Client
	var err error
	client, err = NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	var got Config = client.Config()
	return endpointSet{
		rest:          got.REST.BaseURL,
		publicLinear:  got.WS.PublicLinearURL,
		publicSpot:    got.WS.PublicSpotURL,
		publicInverse: got.WS.PublicInverseURL,
		publicOption:  got.WS.PublicOptionURL,
		private:       got.WS.PrivateURL,
	}
}

// assertEndpoints compares every field so one failure lists all mismatches.
func assertEndpoints(t *testing.T, got endpointSet, want endpointSet) {
	t.Helper()
	var fields = []struct {
		name string
		got  string
		want string
	}{
		{"REST.BaseURL", got.rest, want.rest},
		{"WS.PublicLinearURL", got.publicLinear, want.publicLinear},
		{"WS.PublicSpotURL", got.publicSpot, want.publicSpot},
		{"WS.PublicInverseURL", got.publicInverse, want.publicInverse},
		{"WS.PublicOptionURL", got.publicOption, want.publicOption},
		{"WS.PrivateURL", got.private, want.private},
	}
	for _, f := range fields {
		if f.got != f.want {
			t.Errorf("%s = %q, want %q", f.name, f.got, f.want)
		}
	}
}

func TestNewClient_EndpointsPerEnvironment(t *testing.T) {
	var mainnet endpointSet = endpointSet{
		rest:          "https://api.bybit.com",
		publicLinear:  "wss://stream.bybit.com/v5/public/linear",
		publicSpot:    "wss://stream.bybit.com/v5/public/spot",
		publicInverse: "wss://stream.bybit.com/v5/public/inverse",
		publicOption:  "wss://stream.bybit.com/v5/public/option",
		private:       "wss://stream.bybit.com/v5/private",
	}
	var testnet endpointSet = endpointSet{
		rest:          "https://api-testnet.bybit.com",
		publicLinear:  "wss://stream-testnet.bybit.com/v5/public/linear",
		publicSpot:    "wss://stream-testnet.bybit.com/v5/public/spot",
		publicInverse: "wss://stream-testnet.bybit.com/v5/public/inverse",
		publicOption:  "wss://stream-testnet.bybit.com/v5/public/option",
		private:       "wss://stream-testnet.bybit.com/v5/private",
	}
	// Demo Trading: REST and the private stream on the demo hosts, public
	// market data on production — there is no stream-demo /v5/public/*.
	var demo endpointSet = endpointSet{
		rest:          "https://api-demo.bybit.com",
		publicLinear:  "wss://stream.bybit.com/v5/public/linear",
		publicSpot:    "wss://stream.bybit.com/v5/public/spot",
		publicInverse: "wss://stream.bybit.com/v5/public/inverse",
		publicOption:  "wss://stream.bybit.com/v5/public/option",
		private:       "wss://stream-demo.bybit.com/v5/private",
	}

	var envs = []struct {
		name    string
		testnet bool
		demo    bool
		want    endpointSet
	}{
		{"mainnet", false, false, mainnet},
		{"testnet", true, false, testnet},
		{"demo", false, true, demo},
		{"testnet and demo", true, true, testnet},
	}
	var bases = []struct {
		name string
		cfg  func() Config
	}{
		{"zero Config", func() Config { return Config{} }},
		{"DefaultConfig", DefaultConfig},
	}

	for _, base := range bases {
		for _, env := range envs {
			t.Run(base.name+"/"+env.name, func(t *testing.T) {
				var cfg Config = base.cfg()
				cfg.Testnet = env.testnet
				cfg.Demo = env.demo
				assertEndpoints(t, resolvedEndpoints(t, cfg), env.want)
			})
		}
	}
}

func TestNewClient_ExplicitEndpointsKept(t *testing.T) {
	var explicit endpointSet = endpointSet{
		rest:          "http://127.0.0.1:18080",
		publicLinear:  "ws://127.0.0.1:18081/v5/public/linear",
		publicSpot:    "ws://127.0.0.1:18081/v5/public/spot",
		publicInverse: "ws://127.0.0.1:18081/v5/public/inverse",
		publicOption:  "ws://127.0.0.1:18081/v5/public/option",
		private:       "ws://127.0.0.1:18081/v5/private",
	}

	var envs = []struct {
		name    string
		testnet bool
		demo    bool
	}{
		{"mainnet", false, false},
		{"testnet", true, false},
		{"demo", false, true},
	}

	for _, env := range envs {
		t.Run(env.name, func(t *testing.T) {
			var cfg Config = DefaultConfig()
			cfg.Testnet = env.testnet
			cfg.Demo = env.demo
			cfg.REST.BaseURL = explicit.rest
			cfg.WS.PublicLinearURL = explicit.publicLinear
			cfg.WS.PublicSpotURL = explicit.publicSpot
			cfg.WS.PublicInverseURL = explicit.publicInverse
			cfg.WS.PublicOptionURL = explicit.publicOption
			cfg.WS.PrivateURL = explicit.private
			assertEndpoints(t, resolvedEndpoints(t, cfg), explicit)
		})
	}
}
