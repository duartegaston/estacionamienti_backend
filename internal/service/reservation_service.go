package service

import (
	"bytes"
	"estacionamienti/internal/db"
	"estacionamienti/internal/entities"
	"estacionamienti/internal/repository"
	"fmt"
	"html/template"
	"log"
	"path/filepath"
	"time"
)

type ReservationService struct {
	Repo *repository.ReservationRepository
}

func NewReservationService(repo *repository.ReservationRepository) *ReservationService {
	return &ReservationService{Repo: repo}
}

func (s *ReservationService) CheckAvailability(req entities.ReservationRequest) (*entities.AvailabilityResponse, error) {
	if !req.EndTime.After(req.StartTime) {
		return &entities.AvailabilityResponse{
			IsOverallAvailable: false,
			RequestedStartTime: req.StartTime,
			RequestedEndTime:   req.EndTime,
		}, nil
	}

	hourlyDetails, err := s.Repo.GetHourlyAvailabilityDetails(req.StartTime, req.EndTime, req.VehicleTypeID)
	if err != nil {
		log.Printf("Error from GetHourlyAvailabilityDetails: %v", err)
		return nil, fmt.Errorf("error interno al verificar disponibilidad: %w", err)
	}

	response := &entities.AvailabilityResponse{
		RequestedStartTime: req.StartTime,
		RequestedEndTime:   req.EndTime,
		IsOverallAvailable: true, // Asumimos que sí hasta que se demuestre lo contrario
	}

	if len(hourlyDetails) == 0 {
		response.IsOverallAvailable = false
		return response, nil
	}

	var firstUnavailableTime *time.Time

	for _, detail := range hourlyDetails {
		availableInSlot := detail.TotalSpaces - detail.BookedSpaces
		isSlotAvailable := availableInSlot > 0

		response.SlotDetails = append(response.SlotDetails, entities.TimeSlotAvailability{
			StartTime:       detail.SlotStart,
			EndTime:         detail.SlotEnd,
			IsAvailable:     isSlotAvailable,
			AvailableSpaces: availableInSlot,
		})

		if !isSlotAvailable {
			response.IsOverallAvailable = false
			if firstUnavailableTime == nil {
				tempTime := detail.SlotStart
				firstUnavailableTime = &tempTime
			}
		}
	}

	return response, nil
}

func (s *ReservationService) CreateReservation(req *entities.ReservationRequest) (string, error) {
	code := fmt.Sprintf("%08X", time.Now().UnixNano()%100000000)

	reservation := &db.Reservation{
		Code:            code,
		UserName:        req.UserName,
		UserEmail:       req.UserEmail,
		UserPhone:       req.UserPhone,
		VehicleTypeID:   req.VehicleTypeID,
		VehiclePlate:    req.VehiclePlate,
		VehicleModel:    req.VehicleModel,
		PaymentMethodID: req.PaymentMethodID,
		Status:          "active",
		StartTime:       req.StartTime,
		EndTime:         req.EndTime,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// Generar cobro stripe

	err := s.Repo.CreateReservation(reservation)
	if err != nil {
		log.Printf("Error creating reservation in repository: %v", err)
		return "", err
	}

	//sendReservationSMS(*reservation)
	//sendReservationEmail(*reservation)

	return code, nil
}

func (s *ReservationService) GetReservationByCode(code, email string) (*entities.ReservationResponse, error) {
	return s.Repo.GetReservationByCode(code, email)
}

func (s *ReservationService) CancelReservation(code string) error {
	_, err := s.Repo.CancelReservation(code)
	return err
}

func (s *ReservationService) GetPrices() ([]entities.PriceResponse, error) {
	return s.Repo.GetPrices()
}

func (s *ReservationService) GetVehicleTypes() ([]db.VehicleType, error) {
	return s.Repo.GetVehicleTypes()
}

// ADMIN FUNCTIONS

func (s *ReservationService) ListReservations(date, vehicleType, status string) ([]db.Reservation, error) {
	return s.Repo.ListReservations(date, vehicleType)
}

func (s *ReservationService) ListVehicleSpaces() ([]db.VehicleSpace, error) {
	return s.Repo.ListVehicleSpaces()
}

func (s *ReservationService) UpdateVehicleSpaces(vehicleType string, totalSpaces, availableSpaces int) error {
	return s.Repo.UpdateVehicleSpaces(vehicleType, totalSpaces, availableSpaces)
}

// JOBS

// UpdateFinishedReservations busca reservas activas que han finalizado y actualiza su estado a "finished".
func (s *ReservationService) UpdateFinishedReservations() error {
	log.Println("Cron Job: Checking for reservations to mark as 'finished'...")

	reservationIDs, err := s.Repo.GetActiveReservationIDsPastEndTime()
	if err != nil {
		return fmt.Errorf("cron job: failed to get active reservations past end time: %w", err)
	}

	if len(reservationIDs) == 0 {
		log.Println("Cron Job: No active reservations found past their end time.")
		return nil
	}

	log.Printf("Cron Job: Found %d reservations to mark as 'finished'. IDs: %v", len(reservationIDs), reservationIDs)

	err = s.Repo.UpdateReservationStatuses(reservationIDs, "finished")
	if err != nil {
		return fmt.Errorf("cron job: failed to update reservation statuses: %w", err)
	}

	log.Printf("Cron Job: Successfully updated %d reservations to 'finished'.", len(reservationIDs))
	return nil
}

func sendReservationEmail(reservation db.Reservation) {
	emailData := entities.ReservationEmailData{
		UserName:           reservation.UserName,
		ReservationCode:    reservation.Code,
		VehicleModel:       reservation.VehicleModel,
		VehiclePlate:       reservation.VehiclePlate,
		StartTimeFormatted: reservation.StartTime.Format("02 Jan 2006 15:04 MST"),
		EndTimeFormatted:   reservation.EndTime.Format("02 Jan 2006 15:04 MST"),
		CurrentYear:        time.Now().Year(),
	}

	emailSubject := fmt.Sprintf("Confirmación de tu Reserva en GreenPark - Código: %s", emailData.ReservationCode)

	plainTextBody := fmt.Sprintf(
		"Hola %s,\n\nTu reserva en GreenPark ha sido confirmada.\n\n"+
			"Detalles de la Reserva:\n"+
			"Código de Reserva: %s\n"+
			"Vehículo: %s (Matrícula: %s)\n"+
			"Entrada: %s\n"+
			"Salida: %s\n\n"+
			"Gracias por elegir GreenPark.\n\n"+
			"© %d GreenPark. Todos los derechos reservados.",
		emailData.UserName, emailData.ReservationCode, emailData.VehicleModel, emailData.VehiclePlate,
		emailData.StartTimeFormatted, emailData.EndTimeFormatted, emailData.CurrentYear,
	)

	tmplPath := filepath.Join("internal", "templates", "reservation_email.html")
	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		log.Printf("ALERTA: Error al parsear la plantilla de correo HTML (%s): %v", tmplPath, err)
	}

	var htmlBodyBuffer bytes.Buffer
	if err := tmpl.Execute(&htmlBodyBuffer, emailData); err != nil {
		log.Printf("ALERTA: Error al ejecutar la plantilla de correo HTML para reserva %s: %v", emailData.ReservationCode, err)
	}
	htmlBody := htmlBodyBuffer.String()

	go func(toEmail, userName, subject, plainBody, htmlBodyContent string) {
		errEmail := SendEmailWithSendGrid(toEmail, userName, subject, plainBody, htmlBodyContent)
		if errEmail != nil {
			log.Printf("ALERTA (asíncrono): Falló envío de correo para reserva %s: %v", emailData.ReservationCode, errEmail)
		}
	}(reservation.UserEmail, emailData.UserName, emailSubject, plainTextBody, htmlBody)
}

func sendReservationSMS(reservation db.Reservation) {
	userPhoneNumber := reservation.UserPhone
	reservationCode := reservation.Code
	smsMessage := fmt.Sprintf("GreenParking: Reserva %s confirmada!\nEntrada: %s.\nMás detalles en tu email.",
		reservationCode,
		reservation.StartTime.Format("02/01 15:04"),
	)

	errSMS := SendSMS(userPhoneNumber, smsMessage)
	if errSMS != nil {
		log.Printf("ALERTA: La reserva %s se creó, pero falló el envío del SMS de confirmación a %s: %v", reservationCode, userPhoneNumber, errSMS)
	}
}
