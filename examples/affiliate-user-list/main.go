/*
FILE: examples/affiliate-user-list/main.go

DESCRIPTION:
Read-only smoke test for the affiliate profile (C5).

COVERAGE:
  - affiliate.Client.GetAffiliateUserList
  - affiliate.Client.GetFriendReferrals

USAGE:

	# Requires affiliate-permission API key for aff-user-list
	./scripts/run.sh ./examples/affiliate-user-list
*/

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/tonymontanov/go-bybit/v2/affiliate"
	afftypes "github.com/tonymontanov/go-bybit/v2/affiliate/types"
	"github.com/tonymontanov/go-bybit/v2/examples/internal/exhelp"
)

func main() {
	var opt exhelp.Options = exhelp.LoadEnv()
	exhelp.MustHaveKeys(opt)

	var client, ac = exhelp.NewAffiliateClient(opt)
	defer client.Close()

	var ctx, cancel = context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	fmt.Printf("=== Affiliate info (testnet=%v demo=%v) ===\n\n",
		opt.Testnet, opt.Demo)

	dumpAffiliateUsers(ctx, ac)
	dumpFriendReferrals(ctx, ac)
}

func dumpAffiliateUsers(ctx context.Context, ac *affiliate.Client) {
	var page afftypes.AffiliateUserList
	var err error
	page, err = ac.GetAffiliateUserList(ctx, afftypes.AffiliateUserListRequest{
		Size:    5,
		Need30:  true,
		Need365: true,
	})
	if err != nil {
		fmt.Printf("[affiliate-users] error: %s\n\n", exhelp.Classify(err))
		return
	}
	fmt.Printf("[affiliate-users count=%d cursor=%q]\n", len(page.Users), page.NextPageCursor)
	if len(page.Users) == 0 {
		fmt.Println("  (empty)")
		fmt.Println()
		return
	}
	var row = page.Users[0]
	fmt.Printf("  uid=%s kyc=%v tradeVol30Day=%s source=%q\n\n",
		row.UserID, row.IsKYC, row.TradeVol30Day, row.Source)
}

func dumpFriendReferrals(ctx context.Context, ac *affiliate.Client) {
	var page afftypes.FriendReferralList
	var err error
	page, err = ac.GetFriendReferrals(ctx, afftypes.FriendReferralRequest{
		Status: afftypes.FriendReferralStatusAlive,
		Size:   5,
	})
	if err != nil {
		fmt.Printf("[friend-referrals] error: %s\n\n", exhelp.Classify(err))
		return
	}
	fmt.Printf("[friend-referrals count=%d]\n", len(page.Records))
	if len(page.Records) == 0 {
		fmt.Println("  (empty)")
		fmt.Println()
		return
	}
	var row = page.Records[0]
	fmt.Printf("  inviteeUid=%s status=%d createdAt=%d\n\n",
		row.InviteeUID, row.Status, row.CreatedAt)
}
