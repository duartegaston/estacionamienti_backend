package api

// Availability
type AvailabilityRequest struct {
	EntryTime   string `json:"entry_time"`
	ExitTime    string `json:"exit_time"`
	VehicleType string `json:"vehicle_type"`
}
type AvailabilityResponse struct {
	Available bool   `json:"available"`
	Message   string `json:"message"`
}

// Reservation
type CreateReservationRequest struct {
	EntryTime    string `json:"entry_time"`
	ExitTime     string `json:"exit_time"`
	VehicleType  string `json:"vehicle_type"`
	FullName     string `json:"full_name"`
	Email        string `json:"email"`
	Phone        string `json:"phone"`
	LicensePlate string `json:"license_plate"`
	VehicleModel string `json:"vehicle_model"`
}
type CreateReservationResponse struct {
	ReservationCode string `json:"reservation_code"`
	Message         string `json:"message"`
}

// Add more for update, cancel, list, etc.
