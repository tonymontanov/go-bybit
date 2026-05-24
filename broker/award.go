/*
FILE: broker/award.go

DESCRIPTION:
Broker voucher / award endpoints under /v5/broker/award/*.
*/

package broker

import (
	"context"
	"strconv"

	bybit "github.com/tonymontanov/go-bybit/v2"
	"github.com/tonymontanov/go-bybit/v2/broker/types"
	"github.com/tonymontanov/go-bybit/v2/internal/rest"
)

type rawVoucherSpec struct {
	ID             string `json:"id"`
	Coin           string `json:"coin"`
	AmountUnit     string `json:"amountUnit"`
	ProductLine    string `json:"productLine"`
	SubProductLine string `json:"subProductLine"`
	TotalAmount    string `json:"totalAmount"`
	UsedAmount     string `json:"usedAmount"`
}

type rawVoucherDistribution struct {
	AccountID     string `json:"accountId"`
	AwardID       string `json:"awardId"`
	SpecCode      string `json:"specCode"`
	Amount        string `json:"amount"`
	IsClaimed     bool   `json:"isClaimed"`
	StartAt       string `json:"startAt"`
	EndAt         string `json:"endAt"`
	EffectiveAt   string `json:"effectiveAt"`
	IneffectiveAt string `json:"ineffectiveAt"`
	UsedAmount    string `json:"usedAmount"`
}

func parseSec(s string) int64 {
	if s == "" {
		return 0
	}
	var v, err = strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return v
}

// GetVoucherSpec returns voucher metadata for a broker award ID.
func (c *Client) GetVoucherSpec(ctx context.Context, voucherID string) (types.VoucherSpec, error) {
	if voucherID == "" {
		return types.VoucherSpec{}, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "broker.GetVoucherSpec: id is empty", nil)
	}

	var resp rest.Response
	var err error
	resp, _, err = c.rest().Do(ctx, rest.Options{
		Method: "POST",
		Path:   "/v5/broker/award/info",
		Body: map[string]any{
			"id": voucherID,
		},
		Signed: true,
		Meta: rest.RequestMeta{
			Category: string(bybit.RateLimitCategoryQuery),
		},
	})
	if err != nil {
		return types.VoucherSpec{}, err
	}

	var raw rawVoucherSpec
	if err = resp.UnmarshalResult(&raw); err != nil {
		return types.VoucherSpec{}, bybit.NewError(bybit.ErrorKindUnknown, "", "broker.GetVoucherSpec: parse", err)
	}

	return types.VoucherSpec{
		ID:             raw.ID,
		Coin:           raw.Coin,
		AmountUnit:     raw.AmountUnit,
		ProductLine:    raw.ProductLine,
		SubProductLine: raw.SubProductLine,
		TotalAmount:    dec(raw.TotalAmount),
		UsedAmount:     dec(raw.UsedAmount),
	}, nil
}

// DistributeVoucher issues a voucher to a user account.
func (c *Client) DistributeVoucher(ctx context.Context, req types.DistributeVoucherRequest) error {
	if req.AccountID == "" || req.AwardID == "" || req.SpecCode == "" || req.BrokerID == "" {
		return bybit.NewError(bybit.ErrorKindInvalidRequest, "", "broker.DistributeVoucher: accountId, awardId, specCode and brokerId are required", nil)
	}
	if req.Amount.IsZero() || req.Amount.IsNegative() {
		return bybit.NewError(bybit.ErrorKindInvalidRequest, "", "broker.DistributeVoucher: amount must be positive", nil)
	}

	_, _, err := c.rest().Do(ctx, rest.Options{
		Method: "POST",
		Path:   "/v5/broker/award/distribute-award",
		Body: map[string]any{
			"accountId": req.AccountID,
			"awardId":   req.AwardID,
			"specCode":  req.SpecCode,
			"amount":    req.Amount.String(),
			"brokerId":  req.BrokerID,
		},
		Signed: true,
		Meta: rest.RequestMeta{
			Category: string(bybit.RateLimitCategoryQuery),
		},
	})
	return err
}

// GetVoucherDistribution returns the state of an issued voucher.
func (c *Client) GetVoucherDistribution(ctx context.Context, req types.VoucherDistributionRequest) (types.VoucherDistribution, error) {
	if req.AccountID == "" || req.AwardID == "" || req.SpecCode == "" {
		return types.VoucherDistribution{}, bybit.NewError(bybit.ErrorKindInvalidRequest, "", "broker.GetVoucherDistribution: accountId, awardId and specCode are required", nil)
	}

	var body map[string]any = map[string]any{
		"accountId": req.AccountID,
		"awardId":   req.AwardID,
		"specCode":  req.SpecCode,
	}
	if req.WithUsedAmount {
		body["withUsedAmount"] = true
	}

	var resp rest.Response
	var err error
	resp, _, err = c.rest().Do(ctx, rest.Options{
		Method: "POST",
		Path:   "/v5/broker/award/distribution-record",
		Body:   body,
		Signed: true,
		Meta: rest.RequestMeta{
			Category: string(bybit.RateLimitCategoryQuery),
		},
	})
	if err != nil {
		return types.VoucherDistribution{}, err
	}

	var raw rawVoucherDistribution
	if err = resp.UnmarshalResult(&raw); err != nil {
		return types.VoucherDistribution{}, bybit.NewError(bybit.ErrorKindUnknown, "", "broker.GetVoucherDistribution: parse", err)
	}

	return types.VoucherDistribution{
		AccountID:        raw.AccountID,
		AwardID:          raw.AwardID,
		SpecCode:         raw.SpecCode,
		Amount:           dec(raw.Amount),
		IsClaimed:        raw.IsClaimed,
		StartAtSec:       parseSec(raw.StartAt),
		EndAtSec:         parseSec(raw.EndAt),
		EffectiveAtSec:   parseSec(raw.EffectiveAt),
		IneffectiveAtSec: parseSec(raw.IneffectiveAt),
		UsedAmount:       dec(raw.UsedAmount),
	}, nil
}
