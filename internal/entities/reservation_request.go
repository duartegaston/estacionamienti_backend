package entities

import "time"

type ReservationRequest struct {
	VehicleTypeID       int       `json:"vehicle_type_id"`
	UserName            string    `json:"user_name"`
	UserEmail           string    `json:"user_email"`
	UserPhone           string    `json:"user_phone"`
	VehiclePlate        string    `json:"vehicle_plate"`
	VehicleModel        string    `json:"vehicle_model"`
	PaymentMethodID     int       `json:"payment_method_id"`
	StripePaymentMethod string    `json:"stripe_payment_method_id"`
	Status              string    `json:"status"`
	StartTime           time.Time `json:"start_time"`
	EndTime             time.Time `json:"end_time"`
}

type ReservationResponse struct {
	ID                int       `json:"id"`
	Code              string    `json:"code"`
	UserName          string    `json:"user_name"`
	UserEmail         string    `json:"user_email"`
	UserPhone         string    `json:"user_phone"`
	VehicleTypeID     int       `json:"vehicle_type_id"`
	VehicleTypeName   string    `json:"vehicle_type_name"`
	VehiclePlate      string    `json:"vehicle_plate"`
	VehicleModel      string    `json:"vehicle_model"`
	PaymentMethodID   int       `json:"payment_method_id"`
	PaymentMethodName string    `json:"payment_method_name"`
	Status            string    `json:"status"`
	StartTime         time.Time `json:"start_time"`
	EndTime           time.Time `json:"end_time"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}
