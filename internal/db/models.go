package db

import (
	"database/sql"
	"time"
)

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
	Price             float32   `json:"price"`
	CreatedAt         time.Time `json:"created_at"`
}

type VehicleSpace struct {
	VehicleType     string `json:"vehicle_type"`
	TotalSpaces     int    `json:"total_spaces"`
	AvailableSpaces int    `json:"available_spaces"`
}

type VehicleSpaceWithPrices struct {
	VehicleType string             `json:"vehicle_type"`
	Spaces      int                `json:"spaces"`
	Prices      map[string]float32 `json:"prices"`
}

type Reservation struct {
	ID                    int             `json:"id"`
	Code                  string          `json:"code"`
	UserName              string          `json:"user_name"`
	UserEmail             string          `json:"user_email"`
	UserPhone             sql.NullString  `json:"user_phone"`
	VehicleTypeID         int             `json:"vehicle_type_id"`
	VehiclePlate          sql.NullString  `json:"vehicle_plate"`
	VehicleModel          sql.NullString  `json:"vehicle_model"`
	PaymentMethodID       int             `json:"payment_method_id"`
	Status                string          `json:"status"`
	StartTime             time.Time       `json:"start_time"`
	EndTime               time.Time       `json:"end_time"`
	CreatedAt             time.Time       `json:"created_at"`
	UpdatedAt             time.Time       `json:"updated_at"`
	StripeSessionID       sql.NullString  `json:"stripe_session_id,omitempty"`
	PaymentStatus         sql.NullString  `json:"payment_status,omitempty"`
	Language              string          `json:"language"`
	StripePaymentIntentID sql.NullString  `json:"stripe_payment_intent_id,omitempty"`
	TotalPrice            sql.NullFloat64 `json:"total_price,omitempty"`
}
