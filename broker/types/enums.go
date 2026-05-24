/*
FILE: broker/types/enums.go
*/

package types

// BizType — broker rebate business category on the wire.
type BizType string

const (
	BizTypeSpot        BizType = "SPOT"
	BizTypeDerivatives BizType = "DERIVATIVES"
	BizTypeOptions     BizType = "OPTIONS"
	BizTypeConvert     BizType = "CONVERT"
)
