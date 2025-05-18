package entities

import "time"

type ReservationRequest struct {
	VehicleTypeID int       `json:"vehicle_type_id"`
	UserName      string    `json:"user_name"`
	UserEmail     string    `json:"user_email"`
	UserPhone     string    `json:"user_phone"`
	VehicleType   string    `json:"vehicle_type"`
	VehiclePlate  string    `json:"vehicle_plate"`
	VehicleModel  string    `json:"vehicle_model"`
	PaymentMethod string    `json:"payment_method"`
	Status        string    `json:"status"`
	StartTime     time.Time `json:"start_time"`
	EndTime       time.Time `json:"end_time"`
}
