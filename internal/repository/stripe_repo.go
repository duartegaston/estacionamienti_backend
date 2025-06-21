package repository

import (
	"database/sql"
	"fmt"
	"time"
)

type StripeRepository struct {
	DB *sql.DB
}

func NewStripeRepository(db *sql.DB) *StripeRepository {
	return &StripeRepository{DB: db}
}
func (r *StripeRepository) UpdateReservationStripeInfo(reservationID int, stripeCustomerID, stripePaymentIntentID,
	stripeSetupIntentID, stripeUsedPMID, newStatus, newPaymentStatus string) error {
	query := `
		UPDATE reservations
		SET
			stripe_customer_id = $2,
			stripe_payment_intent_id = $3,
			stripe_setup_intent_id = $4,
			stripe_payment_method_id = $5,
			status = $6,
			payment_status = $7,
			updated_at = $8
		WHERE id = $1`

	_, err := r.DB.Exec(query,
		reservationID,
		stripeCustomerID,
		stripePaymentIntentID,
		stripeSetupIntentID,
		stripeUsedPMID,
		newStatus,
		newPaymentStatus,
		time.Now(),
	)
	if err != nil {
		return fmt.Errorf("error actualizando reserva %d con info de Stripe: %w", reservationID, err)
	}
	return nil
}
