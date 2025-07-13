package service

import (
	"estacionamienti/internal/repository"
	"fmt"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/refund"
	"log"
)

type StripeService struct {
	Repo *repository.ReservationRepository
}

func NewStripeService(Repo *repository.ReservationRepository) *StripeService {
	return &StripeService{Repo: Repo}
}

func (s *StripeService) RefundPaymentBySessionID(sessionID string) error {
	reservation, err := s.Repo.GetReservationByStripeSessionID(sessionID)
	if err != nil {
		return err
	}
	log.Printf("reservation: %v", reservation)
	if reservation.StripePaymentIntentID == "" {
		return fmt.Errorf("No PaymentIntent found for session %s", sessionID)
	}
	log.Printf("Refunding payment for session %s", sessionID)
	params := &stripe.RefundParams{
		PaymentIntent: stripe.String(reservation.StripePaymentIntentID),
	}
	_, err = refund.New(params)
	return err
}

// Create checkout session
func (s *StripeService) CreateCheckoutSession(amount int64, currency, description, customerEmail string, language string) (string, string, error) {
	params := &stripe.CheckoutSessionParams{
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String(currency),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name: stripe.String("GreenParking: " + description),
					},
					UnitAmount: stripe.Int64(amount),
				},
				Quantity: stripe.Int64(1),
			},
		},
		Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
		//TODO: modificar urls
		SuccessURL:    stripe.String("http://localhost:3000/" + language + "/reservations/create/confirmation/?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:     stripe.String("http://localhost:3000/" + language + "/reservations/create/failed"),
		CustomerEmail: stripe.String(customerEmail),
	}

	sess, err := session.New(params)
	if err != nil {
		return "", "", err
	}
	return sess.URL, sess.ID, nil
}
