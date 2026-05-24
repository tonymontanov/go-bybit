/*
FILE: account/contract_test.go

Contract tests for the extended account profile. Fixtures derived from Bybit V5 docs.
*/

package account

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	bybit "github.com/tonymontanov/go-bybit/v2"
	accounttypes "github.com/tonymontanov/go-bybit/v2/account/types"
	commontypes "github.com/tonymontanov/go-bybit/v2/types"
)

func mockAccount(t *testing.T, routes map[string]string) (*httptest.Server, *Client) {
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
		w.Header().Set("X-Bapi-Limit", "50")
		w.Header().Set("X-Bapi-Limit-Status", "1")
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)

	var cfg bybit.Config = bybit.DefaultConfig()
	cfg.REST.BaseURL = srv.URL
	cfg.APIKey = "k"
	cfg.SecretKey = "s"
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

const fixtureAccountInfo = `{
  "retCode": 0,
  "retMsg": "OK",
  "result": {
    "marginMode": "REGULAR_MARGIN",
    "updatedTime": "1697078946000",
    "unifiedMarginStatus": 4,
    "dcpStatus": "OFF",
    "timeWindow": 10,
    "smpGroup": 0,
    "isMasterTrader": false,
    "spotHedgingStatus": "OFF"
  },
  "retExtInfo": {},
  "time": 1697078946000
}`

const fixtureFeeRate = `{
  "retCode": 0,
  "retMsg": "OK",
  "result": {
    "list": [{
      "symbol": "ETHUSDT",
      "takerFeeRate": "0.0006",
      "makerFeeRate": "0.0001"
    }]
  },
  "retExtInfo": {},
  "time": 1676360412576
}`

const fixtureTransactionLog = `{
  "retCode": 0,
  "retMsg": "OK",
  "result": {
    "nextPageCursor": "21963%3A1%2C14954%3A1",
    "list": [{
      "transSubType": "",
      "id": "592324_XRPUSDT_161440249321",
      "symbol": "XRPUSDT",
      "side": "Buy",
      "funding": "-0.003676",
      "orderLinkId": "",
      "orderId": "1672128000-8-592324-1-2",
      "fee": "0.00000000",
      "change": "-0.003676",
      "cashFlow": "0",
      "transactionTime": "1672128000000",
      "type": "SETTLEMENT",
      "feeRate": "0.0001",
      "bonusChange": "",
      "size": "100",
      "qty": "100",
      "cashBalance": "5086.55825002",
      "currency": "USDT",
      "category": "linear",
      "tradePrice": "0.3676",
      "tradeId": "534c0003-4bf7-486f-aa02-78cee36825e4",
      "extraFees": ""
    }]
  },
  "retExtInfo": {},
  "time": 1672132481405
}`

const fixtureCollateralInfo = `{
  "retCode": 0,
  "retMsg": "OK",
  "result": {
    "list": [{
      "availableToBorrow": "3",
      "freeBorrowingAmount": "",
      "freeBorrowAmount": "0",
      "maxBorrowingAmount": "3",
      "hourlyBorrowRate": "0.00000147",
      "borrowUsageRate": "0",
      "collateralSwitch": true,
      "borrowAmount": "0",
      "borrowable": true,
      "currency": "BTC",
      "otherBorrowAmount": "0",
      "marginCollateral": true,
      "freeBorrowingLimit": "0",
      "collateralRatio": "0.95"
    }]
  },
  "retExtInfo": {},
  "time": 1691565901952
}`

const fixtureBorrowHistory = `{
  "retCode": 0,
  "retMsg": "OK",
  "result": {
    "nextPageCursor": "2671153%3A1%2C2671153%3A1",
    "list": [{
      "borrowAmount": "1.06333265702840778",
      "costExemption": "0",
      "freeBorrowedAmount": "0",
      "createdTime": 1697439900204,
      "InterestBearingBorrowSize": "1.06333265702840778",
      "currency": "BTC",
      "unrealisedLoss": "0",
      "hourlyBorrowRate": "0.000001216904",
      "borrowCost": "0.00000129"
    }]
  },
  "retExtInfo": {},
  "time": 1697442206478
}`

const fixtureCoinGreeks = `{
  "retCode": 0,
  "retMsg": "OK",
  "result": {
    "list": [{
      "baseCoin": "BTC",
      "totalDelta": "0.00004001",
      "totalGamma": "-0.00000009",
      "totalVega": "-0.00039689",
      "totalTheta": "0.01243824"
    }]
  },
  "retExtInfo": {},
  "time": 1672287887942
}`

const fixtureSetMarginMode = `{
  "retCode": 0,
  "retMsg": "OK",
  "result": {
    "reasons": []
  },
  "retExtInfo": {},
  "time": 1700000000000
}`

func TestGetAccountInfo_ParsesFields(t *testing.T) {
	t.Parallel()
	_, ac := mockAccount(t, map[string]string{
		"/v5/account/info": fixtureAccountInfo,
	})

	var info accounttypes.AccountInfo
	var err error
	info, err = ac.GetAccountInfo(context.Background())
	if err != nil {
		t.Fatalf("GetAccountInfo: %v", err)
	}
	if info.MarginMode != accounttypes.MarginModeRegular {
		t.Fatalf("marginMode: got %q", info.MarginMode)
	}
	if info.UnifiedMarginStatus != 4 {
		t.Fatalf("unifiedMarginStatus: got %d", info.UnifiedMarginStatus)
	}
	if info.UpdatedAtMs != 1697078946000 {
		t.Fatalf("updatedAtMs: got %d", info.UpdatedAtMs)
	}
}

func TestGetFeeRate_ParsesList(t *testing.T) {
	t.Parallel()
	_, ac := mockAccount(t, map[string]string{
		"/v5/account/fee-rate": fixtureFeeRate,
	})

	var list accounttypes.FeeRateList
	var err error
	list, err = ac.GetFeeRate(context.Background(), accounttypes.FeeRateRequest{
		Category: commontypes.CategoryLinear,
		Symbol:   "ETHUSDT",
	})
	if err != nil {
		t.Fatalf("GetFeeRate: %v", err)
	}
	if len(list.List) != 1 {
		t.Fatalf("list: got %d", len(list.List))
	}
	if !list.List[0].MakerFeeRate.Equal(decimal.RequireFromString("0.0001")) {
		t.Fatalf("maker: %v", list.List[0].MakerFeeRate)
	}
}

func TestGetFeeRate_Validation(t *testing.T) {
	t.Parallel()
	_, ac := mockAccount(t, map[string]string{})

	_, err := ac.GetFeeRate(context.Background(), accounttypes.FeeRateRequest{})
	if err == nil || !bybit.IsInvalidRequest(err) {
		t.Fatalf("expected invalid request, got %v", err)
	}
}

func TestGetTransactionLog_ParsesPage(t *testing.T) {
	t.Parallel()
	_, ac := mockAccount(t, map[string]string{
		"/v5/account/transaction-log": fixtureTransactionLog,
	})

	var page accounttypes.TransactionLogList
	var err error
	page, err = ac.GetTransactionLog(context.Background(), accounttypes.TransactionLogRequest{
		AccountType: commontypes.AccountTypeUnified,
		Category:    commontypes.CategoryLinear,
		Currency:    "USDT",
	})
	if err != nil {
		t.Fatalf("GetTransactionLog: %v", err)
	}
	if len(page.Records) != 1 {
		t.Fatalf("records: got %d", len(page.Records))
	}
	if page.Records[0].Type != "SETTLEMENT" {
		t.Fatalf("type: got %q", page.Records[0].Type)
	}
	if page.NextPageCursor == "" {
		t.Fatal("expected nextPageCursor")
	}
}

func TestGetCollateralInfo_ParsesList(t *testing.T) {
	t.Parallel()
	_, ac := mockAccount(t, map[string]string{
		"/v5/account/collateral-info": fixtureCollateralInfo,
	})

	var rows []accounttypes.CollateralInfo
	var err error
	rows, err = ac.GetCollateralInfo(context.Background(), "BTC")
	if err != nil {
		t.Fatalf("GetCollateralInfo: %v", err)
	}
	if len(rows) != 1 || rows[0].Currency != "BTC" {
		t.Fatalf("rows: %+v", rows)
	}
	if !rows[0].CollateralSwitch {
		t.Fatal("expected collateralSwitch=true")
	}
}

func TestGetBorrowHistory_ParsesPage(t *testing.T) {
	t.Parallel()
	_, ac := mockAccount(t, map[string]string{
		"/v5/account/borrow-history": fixtureBorrowHistory,
	})

	var page accounttypes.BorrowHistoryList
	var err error
	page, err = ac.GetBorrowHistory(context.Background(), accounttypes.BorrowHistoryRequest{
		Currency: "BTC",
		Limit:    1,
	})
	if err != nil {
		t.Fatalf("GetBorrowHistory: %v", err)
	}
	if len(page.Records) != 1 {
		t.Fatalf("records: got %d", len(page.Records))
	}
	if page.Records[0].Currency != "BTC" {
		t.Fatalf("currency: got %q", page.Records[0].Currency)
	}
}

func TestGetCoinGreeks_ParsesList(t *testing.T) {
	t.Parallel()
	_, ac := mockAccount(t, map[string]string{
		"/v5/asset/coin-greeks": fixtureCoinGreeks,
	})

	var rows []accounttypes.CoinGreeks
	var err error
	rows, err = ac.GetCoinGreeks(context.Background(), "BTC")
	if err != nil {
		t.Fatalf("GetCoinGreeks: %v", err)
	}
	if len(rows) != 1 || rows[0].BaseCoin != "BTC" {
		t.Fatalf("rows: %+v", rows)
	}
}

func TestSetMarginMode_Success(t *testing.T) {
	t.Parallel()
	_, ac := mockAccount(t, map[string]string{
		"/v5/account/set-margin-mode": fixtureSetMarginMode,
	})

	var res accounttypes.SetMarginModeResult
	var err error
	res, err = ac.SetMarginMode(context.Background(), accounttypes.MarginModeRegular)
	if err != nil {
		t.Fatalf("SetMarginMode: %v", err)
	}
	if len(res.Reasons) != 0 {
		t.Fatalf("reasons: %+v", res.Reasons)
	}
}

func TestSetCollateralCoin_Validation(t *testing.T) {
	t.Parallel()
	_, ac := mockAccount(t, map[string]string{})

	err := ac.SetCollateralCoin(context.Background(), "", accounttypes.CollateralSwitchOn)
	if err == nil || !bybit.IsInvalidRequest(err) {
		t.Fatalf("expected invalid request, got %v", err)
	}
}

func TestSetHedgingMode_Validation(t *testing.T) {
	t.Parallel()
	_, ac := mockAccount(t, map[string]string{})

	err := ac.SetHedgingMode(context.Background(), "")
	if err == nil || !bybit.IsInvalidRequest(err) {
		t.Fatalf("expected invalid request, got %v", err)
	}
}
