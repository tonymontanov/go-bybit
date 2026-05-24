/*
FILE: asset/contract_test.go

Contract tests for the asset profile. Fixtures derived from Bybit V5 docs.
*/

package asset

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	bybit "github.com/tonymontanov/go-bybit/v2"
	assettypes "github.com/tonymontanov/go-bybit/v2/asset/types"
	commontypes "github.com/tonymontanov/go-bybit/v2/types"
)

func mockAsset(t *testing.T, routes map[string]string) (*httptest.Server, *Client) {
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

const fixtureCoinInfo = `{
  "retCode": 0,
  "retMsg": "OK",
  "result": {
    "rows": [{
      "name": "USDT",
      "coin": "USDT",
      "chains": [{
        "chainType": "Ethereum",
        "confirmation": "12",
        "withdrawFee": "10",
        "depositMin": "0",
        "withdrawMin": "10",
        "chain": "ETH",
        "chainDeposit": "1",
        "chainWithdraw": "1",
        "minAccuracy": "8",
        "withdrawPercentageFee": "0",
        "contractAddress": "0x123",
        "safeConfirmNumber": "64",
        "withdrawMax": "10000000"
      }]
    }]
  },
  "retExtInfo": {},
  "time": 1700000000000
}`

const fixtureAllCoinsBalance = `{
  "retCode": 0,
  "retMsg": "OK",
  "result": {
    "balance": [{
      "coin": "USDT",
      "walletBalance": "1000.5",
      "transferBalance": "900",
      "bonus": "0"
    }]
  },
  "retExtInfo": {},
  "time": 1700000000000
}`

const fixtureInternalTransferCreate = `{
  "retCode": 0,
  "retMsg": "success",
  "result": {
    "transferId": "42c0cfb0-6bca-c242-bc76-4e6df6cbcb16",
    "status": "SUCCESS"
  },
  "retExtInfo": {},
  "time": 1700000000000
}`

const fixtureInternalTransferList = `{
  "retCode": 0,
  "retMsg": "success",
  "result": {
    "list": [{
      "transferId": "selfTransfer_a1091cc7-9364-4b74-8de1-18f02c6f2d5c",
      "coin": "USDT",
      "amount": "5000",
      "fromAccountType": "SPOT",
      "toAccountType": "UNIFIED",
      "timestamp": "1667283263000",
      "status": "SUCCESS"
    }],
    "nextPageCursor": "cursor1"
  },
  "retExtInfo": {},
  "time": 1700000000000
}`

const fixtureDepositAddress = `{
  "retCode": 0,
  "retMsg": "OK",
  "result": {
    "chains": [{
      "chainType": "Ethereum",
      "addressDeposit": "",
      "chain": "ETH",
      "coin": "USDT",
      "address": "0xabc",
      "tag": ""
    }]
  },
  "retExtInfo": {},
  "time": 1700000000000
}`

const fixtureWithdrawCreate = `{
  "retCode": 0,
  "retMsg": "success",
  "result": { "id": "9976" },
  "retExtInfo": {},
  "time": 1700000000000
}`

const fixtureWithdrawableAmount = `{
  "retCode": 0,
  "retMsg": "OK",
  "result": {
    "limitAmountUsd": "10000",
    "withdrawableAmount": "5000",
    "withdrawableAmountUsd": "5000"
  },
  "retExtInfo": {},
  "time": 1700000000000
}`

func TestGetCoinInfo_ParsesChains(t *testing.T) {
	t.Parallel()
	_, ac := mockAsset(t, map[string]string{
		"/v5/asset/coin/query-info": fixtureCoinInfo,
	})

	var rows []assettypes.CoinInfo
	var err error
	rows, err = ac.GetCoinInfo(context.Background(), "USDT")
	if err != nil {
		t.Fatalf("GetCoinInfo: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows: got %d want 1", len(rows))
	}
	if rows[0].Coin != "USDT" {
		t.Fatalf("coin: got %q", rows[0].Coin)
	}
	if len(rows[0].Chains) != 1 || rows[0].Chains[0].Chain != "ETH" {
		t.Fatalf("chains: %+v", rows[0].Chains)
	}
	if !rows[0].Chains[0].WithdrawFee.Equal(decimal.RequireFromString("10")) {
		t.Fatalf("withdrawFee: %v", rows[0].Chains[0].WithdrawFee)
	}
}

func TestGetAllCoinsBalance_ParsesBalance(t *testing.T) {
	t.Parallel()
	_, ac := mockAsset(t, map[string]string{
		"/v5/asset/transfer/query-account-coins-balance": fixtureAllCoinsBalance,
	})

	var bal []assettypes.AccountCoinBalance
	var err error
	bal, err = ac.GetAllCoinsBalance(context.Background(), commontypes.AccountTypeUnified, "")
	if err != nil {
		t.Fatalf("GetAllCoinsBalance: %v", err)
	}
	if len(bal) != 1 || bal[0].Coin != "USDT" {
		t.Fatalf("balance: %+v", bal)
	}
	if !bal[0].WalletBalance.Equal(decimal.RequireFromString("1000.5")) {
		t.Fatalf("walletBalance: %v", bal[0].WalletBalance)
	}
}

func TestCreateInternalTransfer_Success(t *testing.T) {
	t.Parallel()
	_, ac := mockAsset(t, map[string]string{
		"/v5/asset/transfer/inter-transfer": fixtureInternalTransferCreate,
	})

	var res assettypes.InternalTransferResult
	var err error
	res, err = ac.CreateInternalTransfer(context.Background(), assettypes.CreateInternalTransferRequest{
		TransferID:      "42c0cfb0-6bca-c242-bc76-4e6df6cbcb16",
		Coin:            "BTC",
		Amount:          decimal.RequireFromString("0.05"),
		FromAccountType: commontypes.AccountTypeUnified,
		ToAccountType:   commontypes.AccountTypeContract,
	})
	if err != nil {
		t.Fatalf("CreateInternalTransfer: %v", err)
	}
	if res.Status != assettypes.TransferStatusSuccess {
		t.Fatalf("status: got %q", res.Status)
	}
}

func TestGetInternalTransferRecords_ParsesList(t *testing.T) {
	t.Parallel()
	_, ac := mockAsset(t, map[string]string{
		"/v5/asset/transfer/query-inter-transfer-list": fixtureInternalTransferList,
	})

	var list assettypes.InternalTransferList
	var err error
	list, err = ac.GetInternalTransferRecords(context.Background(), InternalTransferListRequest{Coin: "USDT"})
	if err != nil {
		t.Fatalf("GetInternalTransferRecords: %v", err)
	}
	if len(list.Records) != 1 {
		t.Fatalf("records: got %d", len(list.Records))
	}
	if list.Records[0].FromAccountType != commontypes.AccountTypeSpot {
		t.Fatalf("from: got %q", list.Records[0].FromAccountType)
	}
	if list.NextPageCursor != "cursor1" {
		t.Fatalf("cursor: got %q", list.NextPageCursor)
	}
}

func TestGetDepositAddress_ParsesChains(t *testing.T) {
	t.Parallel()
	_, ac := mockAsset(t, map[string]string{
		"/v5/asset/deposit/query-address": fixtureDepositAddress,
	})

	var addrs []assettypes.DepositAddress
	var err error
	addrs, err = ac.GetDepositAddress(context.Background(), "USDT", "ETH")
	if err != nil {
		t.Fatalf("GetDepositAddress: %v", err)
	}
	if len(addrs) != 1 || addrs[0].Address != "0xabc" {
		t.Fatalf("address: %+v", addrs)
	}
}

func TestCreateWithdraw_Success(t *testing.T) {
	t.Parallel()
	_, ac := mockAsset(t, map[string]string{
		"/v5/asset/withdraw/create": fixtureWithdrawCreate,
	})

	var res assettypes.CreateWithdrawResult
	var err error
	res, err = ac.CreateWithdraw(context.Background(), assettypes.CreateWithdrawRequest{
		Coin:        "USDT",
		Chain:       "ETH",
		Address:     "0x99ced129603abc771c0dabe935c326ff6c86645d",
		Amount:      decimal.RequireFromString("24"),
		TimestampMs: 1672196561407,
		AccountType: commontypes.AccountTypeFund,
	})
	if err != nil {
		t.Fatalf("CreateWithdraw: %v", err)
	}
	if res.ID != "9976" {
		t.Fatalf("id: got %q", res.ID)
	}
}

func TestGetWithdrawableAmount_ParsesFields(t *testing.T) {
	t.Parallel()
	_, ac := mockAsset(t, map[string]string{
		"/v5/asset/withdraw/withdrawable-amount": fixtureWithdrawableAmount,
	})

	var amt assettypes.WithdrawableAmount
	var err error
	amt, err = ac.GetWithdrawableAmount(context.Background(), "USDT")
	if err != nil {
		t.Fatalf("GetWithdrawableAmount: %v", err)
	}
	if !amt.WithdrawableAmount.Equal(decimal.RequireFromString("5000")) {
		t.Fatalf("withdrawable: %v", amt.WithdrawableAmount)
	}
}

func TestCreateInternalTransfer_Validation(t *testing.T) {
	t.Parallel()
	_, ac := mockAsset(t, map[string]string{})

	_, err := ac.CreateInternalTransfer(context.Background(), assettypes.CreateInternalTransferRequest{})
	if err == nil || !bybit.IsInvalidRequest(err) {
		t.Fatalf("expected invalid request, got %v", err)
	}
}
