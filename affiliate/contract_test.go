/*
FILE: affiliate/contract_test.go

Contract tests for the affiliate profile. Fixtures derived from Bybit V5 docs.
*/

package affiliate

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	bybit "github.com/tonymontanov/go-bybit/v2"
	afftypes "github.com/tonymontanov/go-bybit/v2/affiliate/types"
)

func mockAffiliate(t *testing.T, routes map[string]string) (*httptest.Server, *Client) {
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

const fixtureAffiliateUserList = `{
  "retCode": 0,
  "retMsg": "",
  "result": {
    "list": [{
      "userId": "103895898",
      "registerTime": "2024-10-29",
      "source": "Default",
      "remarks": "",
      "isKyc": true,
      "takerVol30Day": "12861.362976",
      "makerVol30Day": "262.60865",
      "tradeVol30Day": "13123.971626",
      "depositAmount30Day": "",
      "takerVol365Day": "208971.63737375",
      "makerVol365Day": "33392.64275",
      "tradeVol365Day": "242364.28012375",
      "depositAmount365Day": "",
      "takerVol": "194231.4175",
      "makerVol": "32886.108",
      "tradeVol": "227117.5255",
      "startDate": "2025-08-21",
      "endDate": "2025-10-22",
      "tradfiTradeVol": "0",
      "tradfiTradeVol30Day": "0",
      "tradfiTradeVol365Day": "0",
      "commissions30Day": {"USDT": "2.64288011"},
      "commissionsVol": {"USDT": "0.00835765"},
      "commissions365Day": {"USDT": "2.79509816"}
    }],
    "nextPageCursor": "16197"
  },
  "retExtInfo": {},
  "time": 1733205472513
}`

const fixtureAffiliateUserInfo = `{
  "retCode": 0,
  "retMsg": "",
  "result": {
    "uid": "1087997",
    "takerVol30Day": "17061.64983",
    "makerVol30Day": "10756.454142",
    "tradeVol30Day": "27818.103972",
    "depositAmount30Day": "0",
    "takerVol365Day": "1183752.53919162",
    "makerVol365Day": "44349.42819772",
    "tradeVol365Day": "1228101.96738934",
    "depositAmount365Day": "0",
    "totalWalletBalance": "4",
    "depositUpdateTime": "2026-02-04 00:00:00",
    "vipLevel": "5",
    "volUpdateTime": "2026-02-04 00:00:00",
    "KycLevel": 0,
    "tradfiTradeVol30Day": "1828890.6352",
    "tradfiTradeVol365Day": "1828890.6352",
    "commissions30Day": {"USDT": "17.0461748"},
    "commissions365Day": {"USDT": "130.48078429"},
    "paySendAmount30Day": "1000.00",
    "payFtt": "100.00",
    "cardFtt": "50.00"
  },
  "retExtInfo": {},
  "time": 1770197061282
}`

const fixtureFriendReferrals = `{
  "retCode": 0,
  "retMsg": "",
  "result": {
    "nextCursor": "",
    "records": [{
      "id": "6866",
      "inviteeUid": "1447787",
      "status": 0,
      "createdAt": "1681206247",
      "updatedAt": "1681206247"
    }]
  },
  "retExtInfo": {},
  "time": 1772095760428
}`

func TestGetAffiliateUserList_ParsesPage(t *testing.T) {
	t.Parallel()
	_, ac := mockAffiliate(t, map[string]string{
		"/v5/affiliate/aff-user-list": fixtureAffiliateUserList,
	})

	var page afftypes.AffiliateUserList
	var err error
	page, err = ac.GetAffiliateUserList(context.Background(), afftypes.AffiliateUserListRequest{
		Size:      2,
		Need30:    true,
		Need365:   true,
		StartDate: "2025-10-21",
		EndDate:   "2025-10-22",
	})
	if err != nil {
		t.Fatalf("GetAffiliateUserList: %v", err)
	}
	if len(page.Users) != 1 {
		t.Fatalf("users: got %d", len(page.Users))
	}
	if page.Users[0].UserID != "103895898" {
		t.Fatalf("userId: got %q", page.Users[0].UserID)
	}
	if !page.Users[0].TradeVol30Day.Equal(decimal.RequireFromString("13123.971626")) {
		t.Fatalf("tradeVol30Day: %v", page.Users[0].TradeVol30Day)
	}
	if page.NextPageCursor != "16197" {
		t.Fatalf("cursor: got %q", page.NextPageCursor)
	}
}

func TestGetAffiliateUserList_DateValidation(t *testing.T) {
	t.Parallel()
	_, ac := mockAffiliate(t, map[string]string{})

	_, err := ac.GetAffiliateUserList(context.Background(), afftypes.AffiliateUserListRequest{
		StartDate: "2025-10-21",
	})
	if err == nil || !bybit.IsInvalidRequest(err) {
		t.Fatalf("expected invalid request, got %v", err)
	}
}

func TestGetAffiliateUserInfo_ParsesFields(t *testing.T) {
	t.Parallel()
	_, ac := mockAffiliate(t, map[string]string{
		"/v5/user/aff-customer-info": fixtureAffiliateUserInfo,
	})

	var info afftypes.AffiliateUserInfo
	var err error
	info, err = ac.GetAffiliateUserInfo(context.Background(), afftypes.AffiliateUserInfoRequest{
		UID: "1087997",
	})
	if err != nil {
		t.Fatalf("GetAffiliateUserInfo: %v", err)
	}
	if info.VIPLevel != "5" {
		t.Fatalf("vipLevel: got %q", info.VIPLevel)
	}
	if info.Commissions30Day["USDT"].Equal(decimal.RequireFromString("17.0461748")) == false {
		t.Fatalf("commissions30Day: %v", info.Commissions30Day)
	}
}

func TestGetAffiliateUserInfo_Validation(t *testing.T) {
	t.Parallel()
	_, ac := mockAffiliate(t, map[string]string{})

	_, err := ac.GetAffiliateUserInfo(context.Background(), afftypes.AffiliateUserInfoRequest{})
	if err == nil || !bybit.IsInvalidRequest(err) {
		t.Fatalf("expected invalid request, got %v", err)
	}
}

func TestGetFriendReferrals_ParsesPage(t *testing.T) {
	t.Parallel()
	_, ac := mockAffiliate(t, map[string]string{
		"/v5/user/invitation/referrals": fixtureFriendReferrals,
	})

	var page afftypes.FriendReferralList
	var err error
	page, err = ac.GetFriendReferrals(context.Background(), afftypes.FriendReferralRequest{
		Status: afftypes.FriendReferralStatusAlive,
		Size:   5,
	})
	if err != nil {
		t.Fatalf("GetFriendReferrals: %v", err)
	}
	if len(page.Records) != 1 || page.Records[0].InviteeUID != "1447787" {
		t.Fatalf("records: %+v", page.Records)
	}
}
