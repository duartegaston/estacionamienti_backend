package service

import (
	"github.com/stripe/stripe-go/v78"
	"github.com/stripe/stripe-go/v78/paymentintent"
	"github.com/stripe/stripe-go/v78/refund"
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

// Get payment
func (s *StripeService) GetPaymentIntent(paymentIntentID string) (*stripe.PaymentIntent, error) {
	return paymentintent.Get(paymentIntentID, nil)
}

// Refund payment
func (s *StripeService) RefundPayment(paymentIntentID string) error {
	params := &stripe.RefundParams{
		PaymentIntent: stripe.String(paymentIntentID),
	}
	_, err := refund.New(params)
	return err
}

// Cancel payment
func (s *StripeService) CancelPaymentIntent(paymentIntentID string) error {
	_, err := paymentintent.Cancel(paymentIntentID, nil)
	return err
}

// Capture payment
func (s *StripeService) CapturePaymentIntent(paymentIntentID string) error {
	_, err := paymentintent.Capture(paymentIntentID, nil)
	return err
}
