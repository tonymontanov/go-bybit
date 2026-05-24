/*
FILE: premarket/contract_test.go

Contract tests for the pre-market profile. Fixtures derived from Bybit V5 docs.
*/

package premarket

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	bybit "github.com/tonymontanov/go-bybit/v2"
	pmtypes "github.com/tonymontanov/go-bybit/v2/premarket/types"
	commontypes "github.com/tonymontanov/go-bybit/v2/types"
)

func mockPreMarket(t *testing.T, routes map[string]string) (*httptest.Server, *Client) {
	t.Helper()

	var srv *httptest.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body string
		var ok bool
		body, ok = routes[r.URL.Path]
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"retCode":404,"retMsg":"no fixture","result":{},"retExtInfo":{},"time":0}`)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)

	var cfg bybit.Config = bybit.DefaultConfig()
	cfg.REST.BaseURL = srv.URL
	cfg.REST.RequestTimeout = 3 * time.Second

	var root *bybit.Client
	var err error
	root, err = bybit.NewClient(cfg)
	if err != nil {
		t.Fatalf("bybit.NewClient: %v", err)
	}
	t.Cleanup(func() { _ = root.Close() })
	return srv, NewClient(root)
}

const fixturePreLaunchInstrument = `{
  "retCode": 0,
  "retMsg": "OK",
  "result": {
    "category": "linear",
    "list": [{
      "symbol": "BIOUSDT",
      "contractType": "LinearPerpetual",
      "status": "PreLaunch",
      "baseCoin": "BIO",
      "quoteCoin": "USDT",
      "launchTime": "1735032510000",
      "deliveryTime": "0",
      "deliveryFeeRate": "",
      "priceScale": "4",
      "leverageFilter": {
        "minLeverage": "1",
        "maxLeverage": "5.00",
        "leverageStep": "0.01"
      },
      "priceFilter": {
        "minPrice": "0.0001",
        "maxPrice": "1999.9998",
        "tickSize": "0.0001"
      },
      "lotSizeFilter": {
        "maxOrderQty": "70000",
        "minOrderQty": "1",
        "qtyStep": "1",
        "postOnlyMaxOrderQty": "70000",
        "maxMktOrderQty": "14000",
        "minNotionalValue": "5"
      },
      "unifiedMarginTrade": true,
      "fundingInterval": 480,
      "settleCoin": "USDT",
      "copyTrading": "none",
      "upperFundingRate": "0.05",
      "lowerFundingRate": "-0.05",
      "isPreListing": true,
      "preListingInfo": {
        "curAuctionPhase": "ContinuousTrading",
        "phases": [{
          "phase": "CallAuction",
          "startTime": "1735113600000",
          "endTime": "1735116600000"
        }, {
          "phase": "CallAuctionNoCancel",
          "startTime": "1735116600000",
          "endTime": "1735116900000"
        }, {
          "phase": "CrossMatching",
          "startTime": "1735116900000",
          "endTime": "1735117200000"
        }, {
          "phase": "ContinuousTrading",
          "startTime": "1735117200000",
          "endTime": ""
        }],
        "auctionFeeInfo": {
          "auctionFeeRate": "0",
          "takerFeeRate": "0.001",
          "makerFeeRate": "0.0004"
        }
      },
      "riskParameters": {
        "priceLimitRatioX": "0.05",
        "priceLimitRatioY": "0.1"
      },
      "symbolType": ""
    }],
    "nextPageCursor": "first%3DBIOUSDT%26last%3DBIOUSDT"
  },
  "retExtInfo": {},
  "time": 1735810114435
}`

const fixtureRiskLimit = `{
  "retCode": 0,
  "retMsg": "OK",
  "result": {
    "category": "inverse",
    "list": [{
      "id": 1,
      "symbol": "BTCUSD",
      "riskLimitValue": "150",
      "maintenanceMargin": 0.5,
      "initialMargin": 1,
      "isLowestRisk": 1,
      "maxLeverage": "100.00",
      "mmDeduction": ""
    }],
    "nextPageCursor": ""
  },
  "retExtInfo": {},
  "time": 1672054488010
}`

const fixtureTicker = `{
  "retCode": 0,
  "retMsg": "OK",
  "result": {
    "category": "linear",
    "list": [{
      "symbol": "BIOUSDT",
      "lastPrice": "0.1234",
      "indexPrice": "0.1200",
      "markPrice": "0.1210",
      "prevPrice24h": "0.1100",
      "price24hPcnt": "0.121818",
      "highPrice24h": "0.1300",
      "lowPrice24h": "0.1000",
      "prevPrice1h": "0.1220",
      "openInterest": "1000000",
      "openInterestValue": "123400",
      "turnover24h": "50000",
      "volume24h": "400000",
      "fundingRate": "0.0001",
      "nextFundingTime": "1760371200000",
      "predictedDeliveryPrice": "",
      "basisRate": "",
      "deliveryFeeRate": "",
      "deliveryTime": "0",
      "ask1Size": "100",
      "bid1Price": "0.1230",
      "ask1Price": "0.1240",
      "bid1Size": "200",
      "basis": "",
      "preOpenPrice": "0.1150",
      "preQty": "50000",
      "curPreListingPhase": "ContinuousTrading",
      "fundingIntervalHour": "8",
      "basisRateYear": "",
      "fundingCap": "0.005"
    }]
  },
  "retExtInfo": {},
  "time": 1760352369814
}`

func TestGetPreLaunchInstruments(t *testing.T) {
	var _, pc = mockPreMarket(t, map[string]string{
		"/v5/market/instruments-info": fixturePreLaunchInstrument,
	})

	var page pmtypes.InstrumentList
	var err error
	page, err = pc.GetPreLaunchInstruments(context.Background(), commontypes.CategoryLinear, "")
	if err != nil {
		t.Fatalf("GetPreLaunchInstruments: %v", err)
	}
	if page.NextPageCursor != "first%3DBIOUSDT%26last%3DBIOUSDT" {
		t.Fatalf("cursor: got %q", page.NextPageCursor)
	}
	if len(page.Instruments) != 1 {
		t.Fatalf("instruments: got %d", len(page.Instruments))
	}

	var inst = page.Instruments[0]
	if inst.Symbol != "BIOUSDT" || inst.Status != "PreLaunch" {
		t.Fatalf("symbol/status: %+v", inst)
	}
	if !inst.IsPreListing || inst.PreListingInfo == nil {
		t.Fatal("expected preListingInfo")
	}
	if inst.PreListingInfo.CurAuctionPhase != pmtypes.AuctionPhaseContinuousTrading {
		t.Fatalf("curAuctionPhase: %q", inst.PreListingInfo.CurAuctionPhase)
	}
	if len(inst.PreListingInfo.Phases) != 4 {
		t.Fatalf("phases: got %d", len(inst.PreListingInfo.Phases))
	}
	if !inst.PreListingInfo.AuctionFeeInfo.TakerFeeRate.Equal(decimal.RequireFromString("0.001")) {
		t.Fatalf("takerFeeRate: %s", inst.PreListingInfo.AuctionFeeInfo.TakerFeeRate)
	}
	if inst.MaxLeverage.String() != "5" {
		t.Fatalf("maxLeverage: %s", inst.MaxLeverage)
	}
}

func TestGetRiskLimit(t *testing.T) {
	var _, pc = mockPreMarket(t, map[string]string{
		"/v5/market/risk-limit": fixtureRiskLimit,
	})

	var page pmtypes.RiskLimitList
	var err error
	page, err = pc.GetRiskLimit(context.Background(), pmtypes.RiskLimitRequest{
		Category: commontypes.CategoryInverse,
		Symbol:   "BTCUSD",
	})
	if err != nil {
		t.Fatalf("GetRiskLimit: %v", err)
	}
	if len(page.Tiers) != 1 {
		t.Fatalf("tiers: got %d", len(page.Tiers))
	}

	var tier = page.Tiers[0]
	if tier.ID != 1 || tier.Symbol != "BTCUSD" {
		t.Fatalf("tier id/symbol: %+v", tier)
	}
	if !tier.IsLowestRisk {
		t.Fatal("expected isLowestRisk")
	}
	if !tier.MaintenanceMargin.Equal(decimal.RequireFromString("0.5")) {
		t.Fatalf("maintenanceMargin: %s", tier.MaintenanceMargin)
	}
	if !tier.InitialMargin.Equal(decimal.RequireFromString("1")) {
		t.Fatalf("initialMargin: %s", tier.InitialMargin)
	}
}

func TestGetTickers(t *testing.T) {
	var _, pc = mockPreMarket(t, map[string]string{
		"/v5/market/tickers": fixtureTicker,
	})

	var page pmtypes.TickerList
	var err error
	page, err = pc.GetTickers(context.Background(), commontypes.CategoryLinear, "BIOUSDT")
	if err != nil {
		t.Fatalf("GetTickers: %v", err)
	}
	if len(page.Tickers) != 1 {
		t.Fatalf("tickers: got %d", len(page.Tickers))
	}

	var tk = page.Tickers[0]
	if tk.Symbol != "BIOUSDT" {
		t.Fatalf("symbol: %q", tk.Symbol)
	}
	if !tk.PreOpenPrice.Equal(decimal.RequireFromString("0.1150")) {
		t.Fatalf("preOpenPrice: %s", tk.PreOpenPrice)
	}
	if !tk.PreQty.Equal(decimal.RequireFromString("50000")) {
		t.Fatalf("preQty: %s", tk.PreQty)
	}
	if tk.CurPreListingPhase != "ContinuousTrading" {
		t.Fatalf("curPreListingPhase: %q", tk.CurPreListingPhase)
	}
}

func TestGetInstrumentsInvalidCategory(t *testing.T) {
	var _, pc = mockPreMarket(t, map[string]string{})

	var _, err = pc.GetInstruments(context.Background(), pmtypes.InstrumentsRequest{
		Category: commontypes.CategorySpot,
	})
	if err == nil {
		t.Fatal("expected error for spot category")
	}
}
