package service

import (
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/refund"
)

type StripeService struct{}

func NewStripeService() *StripeService {
	return &StripeService{}
}

// Refund payment
func (s *StripeService) RefundPayment(paymentIntentID string) error {
	params := &stripe.RefundParams{
		PaymentIntent: stripe.String(paymentIntentID),
	}
	_, err := refund.New(params)
	return err
}

// Create checkout session
func (s *StripeService) CreateCheckoutSession(amount int64, currency, description, customerEmail string) (string, string, error) {
	params := &stripe.CheckoutSessionParams{
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String(currency),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name: stripe.String(description),
					},
					UnitAmount: stripe.Int64(amount),
				},
				Quantity: stripe.Int64(1),
			},
		},
		Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
		//TODO: modificar urls
		SuccessURL:    stripe.String("https://tusitio.com/confirmacion?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:     stripe.String("https://tusitio.com/cancelada"),
		CustomerEmail: stripe.String(customerEmail),
	}

	sess, err := session.New(params)
	if err != nil {
		return "", "", err
	}
	return sess.URL, sess.ID, nil
}
