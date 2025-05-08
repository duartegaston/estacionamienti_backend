package db

import "time"

type VehicleSpace struct {
	ID              int    `json:"id"`
	VehicleType     string `json:"vehicle_type"`
	TotalSpaces     int    `json:"total_spaces"`
	AvailableSpaces int    `json:"available_spaces"`
}

type Reservation struct {
	ID              int       `json:"id"`
	ReservationCode string    `json:"reservation_code"`
	EntryTime       time.Time `json:"entry_time"`
	ExitTime        time.Time `json:"exit_time"`
	VehicleType     string    `json:"vehicle_type"`
	FullName        string    `json:"full_name"`
	Email           string    `json:"email"`
	Phone           string    `json:"phone"`
	LicensePlate    string    `json:"license_plate"`
	VehicleModel    string    `json:"vehicle_model"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
