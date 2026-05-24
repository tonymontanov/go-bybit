/*
FILE: broker/contract_test.go

Contract tests for the broker profile. Fixtures derived from Bybit V5 docs.
*/

package broker

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	bybit "github.com/tonymontanov/go-bybit/v2"
	brokertypes "github.com/tonymontanov/go-bybit/v2/broker/types"
)

func mockBroker(t *testing.T, routes map[string]string) (*httptest.Server, *Client) {
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
		w.Header().Set("X-Bapi-Limit", "10")
		w.Header().Set("X-Bapi-Limit-Status", "9")
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

const fixtureBrokerAccountInfo = `{
  "retCode": 0,
  "retMsg": "success",
  "result": {
    "subAcctQty": "2",
    "maxSubAcctQty": "20",
    "baseFeeRebateRate": {
      "spot": "10.0%",
      "derivatives": "10.0%"
    },
    "markupFeeRebateRate": {
      "spot": "6.00%",
      "derivatives": "9.00%",
      "convert": "3.00%"
    },
    "ts": "1701395633402"
  },
  "retExtInfo": {},
  "time": 1701395633403
}`

const fixtureBrokerEarnings = `{
  "retCode": 0,
  "retMsg": "success",
  "result": {
    "totalEarningCat": {
      "spot": [],
      "derivatives": [{"coin": "USDT", "earning": "0.00027844"}],
      "options": [],
      "total": [{"coin": "USDT", "earning": "0.00027844"}]
    },
    "details": [{
      "userId": "117894077",
      "bizType": "DERIVATIVES",
      "symbol": "DOGEUSDT",
      "coin": "USDT",
      "earning": "0.00016166",
      "markupEarning": "0.000032332",
      "baseFeeEarning": "0.000129328",
      "orderId": "ec2132f2-a7e0-4a0c-9219-9f3cbcd8e878",
      "execId": "c8f418a0-2ccc-594f-ae72-effedf24d0c4",
      "execTime": "1701275846033"
    }],
    "nextPageCursor": ""
  },
  "retExtInfo": {},
  "time": 1701398193964
}`

const fixtureSubMemberDeposit = `{
  "retCode": 0,
  "retMsg": "success",
  "result": {
    "rows": [{
      "id": "9976",
      "subMemberId": "117894077",
      "coin": "USDT",
      "chain": "ETH",
      "amount": "100",
      "txID": "0xabc",
      "status": 2,
      "toAddress": "0x123",
      "tag": "",
      "depositFee": "0",
      "successAt": "1700000000000",
      "confirmations": "12",
      "txIndex": "1",
      "blockHash": "0xhash",
      "batchReleaseLimit": "-1",
      "depositType": "0",
      "fromAddress": "0xfrom",
      "taxDepositRecordsId": "0",
      "taxStatus": 0
    }],
    "nextPageCursor": ""
  },
  "retExtInfo": {},
  "time": 1700000000000
}`

const fixtureVoucherSpec = `{
  "retCode": 0,
  "retMsg": "",
  "result": {
    "id": "80209",
    "coin": "USDT",
    "amountUnit": "AWARD_AMOUNT_UNIT_USD",
    "productLine": "PRODUCT_LINE_CONTRACT",
    "subProductLine": "SUB_PRODUCT_LINE_CONTRACT_DEFAULT",
    "totalAmount": "10000",
    "usedAmount": "100"
  },
  "retExtInfo": {},
  "time": 1726107086313
}`

const fixtureVoucherDistribution = `{
  "retCode": 0,
  "retMsg": "",
  "result": {
    "accountId": "5714139",
    "awardId": "189528",
    "specCode": "demo000",
    "amount": "1",
    "isClaimed": true,
    "startAt": "1725926400",
    "endAt": "1733788800",
    "effectiveAt": "1726531200",
    "ineffectiveAt": "1733817600",
    "usedAmount": ""
  },
  "retExtInfo": {},
  "time": 1726112099846
}`

func TestGetAccountInfo_ParsesFields(t *testing.T) {
	t.Parallel()
	_, bc := mockBroker(t, map[string]string{
		"/v5/broker/account-info": fixtureBrokerAccountInfo,
	})

	var info brokertypes.AccountInfo
	var err error
	info, err = bc.GetAccountInfo(context.Background())
	if err != nil {
		t.Fatalf("GetAccountInfo: %v", err)
	}
	if info.SubAccountQty != "2" {
		t.Fatalf("subAcctQty: got %q", info.SubAccountQty)
	}
	if info.BaseFeeRebateRate.Spot != "10.0%" {
		t.Fatalf("base spot: got %q", info.BaseFeeRebateRate.Spot)
	}
	if info.TimestampMs != 1701395633402 {
		t.Fatalf("ts: got %d", info.TimestampMs)
	}
}

func TestGetEarnings_ParsesPage(t *testing.T) {
	t.Parallel()
	_, bc := mockBroker(t, map[string]string{
		"/v5/broker/earnings-info": fixtureBrokerEarnings,
	})

	var page brokertypes.EarningsList
	var err error
	page, err = bc.GetEarnings(context.Background(), brokertypes.EarningsRequest{
		Begin: "20231129",
		End:   "20231129",
		UID:   "117894077",
	})
	if err != nil {
		t.Fatalf("GetEarnings: %v", err)
	}
	if len(page.Details) != 1 {
		t.Fatalf("details: got %d", len(page.Details))
	}
	if page.Details[0].BizType != brokertypes.BizTypeDerivatives {
		t.Fatalf("bizType: got %q", page.Details[0].BizType)
	}
	if len(page.CategoryTotals.Derivatives) != 1 {
		t.Fatalf("derivatives totals: got %d", len(page.CategoryTotals.Derivatives))
	}
}

func TestGetEarnings_BeginEndValidation(t *testing.T) {
	t.Parallel()
	_, bc := mockBroker(t, map[string]string{})

	_, err := bc.GetEarnings(context.Background(), brokertypes.EarningsRequest{
		Begin: "20231129",
	})
	if err == nil || !bybit.IsInvalidRequest(err) {
		t.Fatalf("expected invalid request, got %v", err)
	}
}

func TestGetSubMemberDepositRecords_ParsesRows(t *testing.T) {
	t.Parallel()
	_, bc := mockBroker(t, map[string]string{
		"/v5/broker/asset/query-sub-member-deposit-record": fixtureSubMemberDeposit,
	})

	var page brokertypes.SubMemberDepositList
	var err error
	page, err = bc.GetSubMemberDepositRecords(context.Background(), brokertypes.SubMemberDepositRequest{
		Coin: "USDT",
	})
	if err != nil {
		t.Fatalf("GetSubMemberDepositRecords: %v", err)
	}
	if len(page.Records) != 1 || page.Records[0].Coin != "USDT" {
		t.Fatalf("records: %+v", page.Records)
	}
}

func TestGetVoucherSpec_ParsesFields(t *testing.T) {
	t.Parallel()
	_, bc := mockBroker(t, map[string]string{
		"/v5/broker/award/info": fixtureVoucherSpec,
	})

	var spec brokertypes.VoucherSpec
	var err error
	spec, err = bc.GetVoucherSpec(context.Background(), "80209")
	if err != nil {
		t.Fatalf("GetVoucherSpec: %v", err)
	}
	if spec.Coin != "USDT" {
		t.Fatalf("coin: got %q", spec.Coin)
	}
	if !spec.TotalAmount.Equal(decimal.RequireFromString("10000")) {
		t.Fatalf("totalAmount: %v", spec.TotalAmount)
	}
}

func TestGetVoucherDistribution_ParsesFields(t *testing.T) {
	t.Parallel()
	_, bc := mockBroker(t, map[string]string{
		"/v5/broker/award/distribution-record": fixtureVoucherDistribution,
	})

	var dist brokertypes.VoucherDistribution
	var err error
	dist, err = bc.GetVoucherDistribution(context.Background(), brokertypes.VoucherDistributionRequest{
		AccountID: "5714139",
		AwardID:   "189528",
		SpecCode:  "demo000",
	})
	if err != nil {
		t.Fatalf("GetVoucherDistribution: %v", err)
	}
	if !dist.IsClaimed {
		t.Fatal("expected isClaimed=true")
	}
	if dist.StartAtSec != 1725926400 {
		t.Fatalf("startAt: got %d", dist.StartAtSec)
	}
}

func TestDistributeVoucher_Validation(t *testing.T) {
	t.Parallel()
	_, bc := mockBroker(t, map[string]string{})

	err := bc.DistributeVoucher(context.Background(), brokertypes.DistributeVoucherRequest{})
	if err == nil || !bybit.IsInvalidRequest(err) {
		t.Fatalf("expected invalid request, got %v", err)
	}
}

func TestGetVoucherSpec_Validation(t *testing.T) {
	t.Parallel()
	_, bc := mockBroker(t, map[string]string{})

	_, err := bc.GetVoucherSpec(context.Background(), "")
	if err == nil || !bybit.IsInvalidRequest(err) {
		t.Fatalf("expected invalid request, got %v", err)
	}
}
