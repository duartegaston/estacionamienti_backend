package api

import (
	"encoding/json"
	"estacionamienti/internal/service"
	"io"
	"log"
	"net/http"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/webhook"
)

type StripeWebhookHandler struct {
	StripeSecret       string
	reservationService *service.ReservationService
}

func NewStripeWebhookHandler(stripeSecret string, reservationService *service.ReservationService) *StripeWebhookHandler {
	return &StripeWebhookHandler{
		StripeSecret:       stripeSecret,
		reservationService: reservationService,
	}
}

func (h *StripeWebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	const maxBodyBytes = int64(65536)
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	sigHeader := r.Header.Get("Stripe-Signature")
	event, err := webhook.ConstructEvent(payload, sigHeader, h.StripeSecret)
	if err != nil {
		log.Printf("Webhook signature verification failed: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	switch event.Type {
	case "payment_intent.succeeded":
		var pi stripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
			log.Printf("Error parsing payment_intent: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		err := h.reservationService.UpdatePaymentStatusByStripeID(pi.ID, string(pi.Status))
		log.Printf("PaymentIntent succeeded: %s", pi.ID)
		if err != nil {
			log.Printf("DB error: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

	case "payment_intent.payment_failed":
		var pi stripe.PaymentIntent
		json.Unmarshal(event.Data.Raw, &pi)
		log.Printf("PaymentIntent failed: %s", pi.ID)
		err := h.reservationService.UpdatePaymentStatusByStripeID(pi.ID, "payment_failed")
		if err != nil {
			return
		}

	case "payment_intent.canceled":
		var pi stripe.PaymentIntent
		json.Unmarshal(event.Data.Raw, &pi)
		log.Printf("PaymentIntent canceled: %s", pi.ID)
		err := h.reservationService.UpdatePaymentStatusByStripeID(pi.ID, "canceled")
		if err != nil {
			return
		}

	case "charge.refunded":
		var charge stripe.Charge
		json.Unmarshal(event.Data.Raw, &charge)
		if charge.PaymentIntent != nil {
			log.Printf("Charge refunded for PI %s", charge.PaymentIntent.ID)
			err := h.reservationService.UpdatePaymentStatusByStripeID(charge.PaymentIntent.ID, "refunded")
			if err != nil {
				return
			}
		}

	default:
		log.Printf("ℹUnhandled event type: %s", event.Type)
	}

	w.WriteHeader(http.StatusOK)
}
