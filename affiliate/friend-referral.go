/*
FILE: affiliate/friend-referral.go

DESCRIPTION:
GET /v5/user/invitation/referrals — friend invitation referrals.
*/

package affiliate

import (
	"context"
	"net/url"
	"strconv"

	bybit "github.com/tonymontanov/go-bybit/v2"
	"github.com/tonymontanov/go-bybit/v2/affiliate/types"
	"github.com/tonymontanov/go-bybit/v2/internal/rest"
)

type rawFriendReferral struct {
	ID         string `json:"id"`
	InviteeUID string `json:"inviteeUid"`
	Status     int    `json:"status"`
	CreatedAt  string `json:"createdAt"`
	UpdatedAt  string `json:"updatedAt"`
}

type friendReferralPayload struct {
	NextCursor string              `json:"nextCursor"`
	Records    []rawFriendReferral `json:"records"`
}

// GetFriendReferrals returns paginated friend invitation records.
func (c *Client) GetFriendReferrals(ctx context.Context, req types.FriendReferralRequest) (types.FriendReferralList, error) {
	var query url.Values = url.Values{}
	if req.Status != "" {
		query.Set("status", string(req.Status))
	}
	if req.Size > 0 {
		if req.Size > 100 {
			req.Size = 100
		}
		query.Set("size", strconv.Itoa(req.Size))
	}
	if req.Cursor != "" {
		query.Set("cursor", req.Cursor)
	}

	var resp rest.Response
	var err error
	resp, _, err = c.rest().Do(ctx, rest.Options{
		Method: "GET",
		Path:   "/v5/user/invitation/referrals",
		Query:  query,
		Signed: true,
		Meta: rest.RequestMeta{
			Category: string(bybit.RateLimitCategoryQuery),
		},
	})
	if err != nil {
		return types.FriendReferralList{}, err
	}

	var payload friendReferralPayload
	if err = resp.UnmarshalResult(&payload); err != nil {
		return types.FriendReferralList{}, bybit.NewError(bybit.ErrorKindUnknown, "", "affiliate.GetFriendReferrals: parse", err)
	}

	var out types.FriendReferralList
	out.NextCursor = payload.NextCursor
	out.Records = make([]types.FriendReferral, 0, len(payload.Records))
	var i int
	for i = 0; i < len(payload.Records); i++ {
		var row = payload.Records[i]
		out.Records = append(out.Records, types.FriendReferral{
			ID:         row.ID,
			InviteeUID: row.InviteeUID,
			Status:     row.Status,
			CreatedAt:  ms(row.CreatedAt),
			UpdatedAt:  ms(row.UpdatedAt),
		})
	}
	return out, nil
}
