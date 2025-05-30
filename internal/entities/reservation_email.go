package entities

type ReservationEmailData struct {
	UserName           string
	ReservationCode    string
	VehicleModel       string
	VehiclePlate       string
	StartTimeFormatted string
	EndTimeFormatted   string
	CurrentYear        int
}
