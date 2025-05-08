package db

import "time"

type VehicleSpace struct {
	ID              int
	VehicleType     string
	TotalSpaces     int
	AvailableSpaces int
}

type Reservation struct {
	ID              int
	ReservationCode string
	EntryTime       time.Time
	ExitTime        time.Time
	VehicleType     string
	FullName        string
	Email           string
	Phone           string
	LicensePlate    string
	VehicleModel    string
	Status          string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
