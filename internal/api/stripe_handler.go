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

const (
	active        = "active"
	canceled      = "canceled"
	refunded      = "refunded"
	paymentFailed = "payment_failed"
	confirmed     = "confirmed"
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
		err := h.reservationService.UpdateReservationAndPaymentStatusByStripeID(pi.ID, active, string(pi.Status))
		log.Printf("PaymentIntent succeeded: %s", pi.ID)
		if err != nil {
			log.Printf("DB error: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		reservation, err := h.reservationService.GerReservationByPaymentIntentID(pi.ID)
		if err != nil {
			log.Printf("DB error: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		h.reservationService.SendReservationSMS(*reservation, confirmed)
		h.reservationService.SendReservationEmail(*reservation, confirmed)

	case "payment_intent.payment_failed":
		var pi stripe.PaymentIntent
		json.Unmarshal(event.Data.Raw, &pi)
		log.Printf("PaymentIntent failed: %s", pi.ID)
		// Payment failed: (tarjeta vencida o sin fondos)
		// ver de dar solucion al usuario en este caso, reintentar pago o algo asi.
		err := h.reservationService.UpdateReservationAndPaymentStatusByStripeID(pi.ID, active, paymentFailed)
		if err != nil {
			return
		}
		// Op 1: dejar la reserva activa avisandole al usuario que debe pagar en el lugar o la puede cancelar.
		// Msj: Tu pago no se pudo procesar. Podés pagar al llegar al estacionamiento o cancelar sin cargo hasta X horas antes.
		// Contra: el usuario si no va no tenemos forma de cobrarle.
		// Op 2: cancelar la reserva avisando al usuario.
		// Evitamos reserva fantasma.
		// Op 3: ver si podemos darle la posibilidad de reintentar el pago. Pero tendria que implementar un cron job para cancelarla si pasaron X horas

	case "charge.refunded":
		var charge stripe.Charge
		json.Unmarshal(event.Data.Raw, &charge)
		if charge.PaymentIntent != nil {
			log.Printf("Charge refunded for PI %s", charge.PaymentIntent.ID)
			err := h.reservationService.UpdateReservationAndPaymentStatusByStripeID(charge.PaymentIntent.ID, canceled, refunded)
			if err != nil {
				return
			}
		}

	default:
		log.Printf("Unhandled event type: %s", event.Type)
	}

	w.WriteHeader(http.StatusOK)
}
