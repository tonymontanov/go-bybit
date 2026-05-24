/*
FILE: affiliate/types/enums.go
*/

package types

// BusinessLine — aff-customer-info business filter on the wire.
type BusinessLine string

const (
	BusinessLineDerivatives BusinessLine = "1"
	BusinessLineSpot        BusinessLine = "2"
	BusinessLineByFi        BusinessLine = "3"
	BusinessLineUSDC        BusinessLine = "4"
	BusinessLineOptions     BusinessLine = "5"
)

// FriendReferralStatus — invitation relationship status filter.
type FriendReferralStatus string

const (
	FriendReferralStatusAlive   FriendReferralStatus = "0"
	FriendReferralStatusInvalid FriendReferralStatus = "1"
)
