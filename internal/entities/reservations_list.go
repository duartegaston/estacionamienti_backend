package entities

type ReservationsList struct {
	Total        int64                 `json:"total"`
	Limit        int                   `json:"limit"`
	Offset       int                   `json:"offset"`
	Reservations []ReservationResponse `json:"reservations"`
}
