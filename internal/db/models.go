package db

import "time"

type Admin struct {
	ID           int       `json:"id"`
	UserName     string    `json:"user_name"`
	PasswordHash string    `json:"password_hash"`
	CreatedAt    time.Time `json:"created_at"`
}

type VehicleType struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type ReservationTime struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type VehiclePrice struct {
	ID                int       `json:"id"`
	VehicleTypeID     int       `json:"vehicle_type_id"`
	ReservationTimeID int       `json:"reservation_time_id"`
	Price             int       `json:"price"`
	CreatedAt         time.Time `json:"created_at"`
}

type VehicleSpace struct {
	VehicleType     string `json:"vehicle_type"`
	TotalSpaces     int    `json:"total_spaces"`
	AvailableSpaces int    `json:"available_spaces"`
}

type Reservation struct {
	ID                    int       `json:"id"`
	Code                  string    `json:"code"`
	UserName              string    `json:"user_name"`
	UserEmail             string    `json:"user_email"`
	UserPhone             string    `json:"user_phone"`
	VehicleTypeID         int       `json:"vehicle_type_id"`
	VehiclePlate          string    `json:"vehicle_plate"`
	VehicleModel          string    `json:"vehicle_model"`
	PaymentMethodID       int       `json:"payment_method_id"`
	Status                string    `json:"status"`
	StartTime             time.Time `json:"start_time"`
	EndTime               time.Time `json:"end_time"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
	StripeCustomerID      string    `json:"stripe_customer_id,omitempty"`
	StripePaymentIntentID string    `json:"stripe_payment_intent_id,omitempty"`
	StripeSetupIntentID   string    `json:"stripe_setup_intent_id,omitempty"`
	StripePaymentMethodID string    `json:"stripe_payment_method_id,omitempty"`
	PaymentStatus         string    `json:"payment_status,omitempty"`
}
