/*
FILE: affiliate/types/friend-referral.go
*/

package types

// FriendReferralRequest — filters for GET /v5/user/invitation/referrals.
type FriendReferralRequest struct {
	Status FriendReferralStatus
	Size   int
	Cursor string
}

// FriendReferral — one friend-invitation record.
type FriendReferral struct {
	ID         string
	InviteeUID string
	Status     int
	CreatedAt  int64
	UpdatedAt  int64
}

// FriendReferralList — paginated friend referral page.
type FriendReferralList struct {
	Records    []FriendReferral
	NextCursor string
}
