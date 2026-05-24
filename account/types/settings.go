/*
FILE: account/types/settings.go
*/

package types

// SetMarginModeReason — failure reason from POST /v5/account/set-margin-mode.
type SetMarginModeReason struct {
	ReasonCode string
	ReasonMsg  string
}

// SetMarginModeResult — POST /v5/account/set-margin-mode result.
type SetMarginModeResult struct {
	Reasons []SetMarginModeReason
}
