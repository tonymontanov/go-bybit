/*
FILE: asset/types/enums.go

DESCRIPTION:
Asset-specific enums not shared with trading profiles.
*/

package types

// TransferStatus — internal / universal transfer lifecycle on the wire.
type TransferStatus string

const (
	TransferStatusUnknown TransferStatus = "STATUS_UNKNOWN"
	TransferStatusSuccess TransferStatus = "SUCCESS"
	TransferStatusPending TransferStatus = "PENDING"
	TransferStatusFailed  TransferStatus = "FAILED"
)

// WithdrawStatus — withdrawal record status strings returned by Bybit.
// The exchange extends this catalogue; unknown values are returned
// verbatim in WithdrawRecord.Status.
type WithdrawStatus string

const (
	WithdrawStatusSecurityCheck   WithdrawStatus = "SecurityCheck"
	WithdrawStatusPending         WithdrawStatus = "Pending"
	WithdrawStatusSuccess         WithdrawStatus = "success"
	WithdrawStatusCancelByUser    WithdrawStatus = "CancelByUser"
	WithdrawStatusReject          WithdrawStatus = "Reject"
	WithdrawStatusFail            WithdrawStatus = "Fail"
	WithdrawStatusBlockchainConfirmed WithdrawStatus = "BlockchainConfirmed"
)

// DepositStatus — deposit record status strings returned by Bybit.
type DepositStatus string

const (
	DepositStatusUnknown   DepositStatus = "0"
	DepositStatusPending   DepositStatus = "1"
	DepositStatusSuccess   DepositStatus = "2"
	DepositStatusFailed    DepositStatus = "3"
)
