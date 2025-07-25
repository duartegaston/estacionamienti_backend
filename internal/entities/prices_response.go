package entities

type PriceResponse struct {
	VehicleType     string  `json:"vehicle_type"`
	ReservationTime string  `json:"reservation_time"`
	Price           float32 `json:"price"`
}
