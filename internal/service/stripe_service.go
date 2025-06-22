package service

import (
	"github.com/stripe/stripe-go/v78"
	"github.com/stripe/stripe-go/v78/paymentintent"
)

type StripeService struct{}

func NewStripeService() *StripeService {
	return &StripeService{}
}

// On line payment
func (s *StripeService) CreatePaymentIntent(amount int64, currency, description string) (*stripe.PaymentIntent, error) {
	params := &stripe.PaymentIntentParams{
		Amount:      stripe.Int64(amount),
		Currency:    stripe.String(currency),
		Description: stripe.String(description),
		AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
			Enabled: stripe.Bool(true),
		},
	}
	return paymentintent.New(params)
}

// On site payment
func (s *StripeService) CreatePaymentIntentWithManualCapture(amount int64, currency, description string) (*stripe.PaymentIntent, error) {
	params := &stripe.PaymentIntentParams{
		Amount:        stripe.Int64(amount),
		Currency:      stripe.String(currency),
		Description:   stripe.String(description),
		CaptureMethod: stripe.String(string(stripe.PaymentIntentCaptureMethodManual)),
		AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
			Enabled: stripe.Bool(true),
		},
	}
	return paymentintent.New(params)
}
