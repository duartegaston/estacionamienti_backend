package service

import (
	"bytes"
	"estacionamienti/internal/db"
	"estacionamienti/internal/entities"
	"estacionamienti/internal/repository"
	"fmt"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"html/template"
	"log"
	"path/filepath"
	"time"
)

const (
	statusPending = "pending"
	statusCancel  = "canceled"
)

type ReservationService struct {
	stripeService *StripeService
	Repo          *repository.ReservationRepository
}

func NewReservationService(repo *repository.ReservationRepository, stripeService *StripeService) *ReservationService {
	return &ReservationService{Repo: repo,
		stripeService: stripeService}
}

func (s *ReservationService) GetPrices() ([]entities.PriceResponse, error) {
	return s.Repo.GetPrices()
}

func (s *ReservationService) GetVehicleTypes() ([]db.VehicleType, error) {
	return s.Repo.GetVehicleTypes()
}

func (s *ReservationService) CheckAvailability(req entities.ReservationRequest) (*entities.AvailabilityResponse, error) {
	hourlyDetails, err := s.Repo.GetHourlyAvailabilityDetails(req.StartTime, req.EndTime, req.VehicleTypeID)
	if err != nil {
		log.Printf("Error from GetHourlyAvailabilityDetails: %v", err)
		return nil, fmt.Errorf("internal error checking availability: %w", err)
	}

	response := &entities.AvailabilityResponse{
		RequestedStartTime: req.StartTime,
		RequestedEndTime:   req.EndTime,
		IsOverallAvailable: true,
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

func (s *ReservationService) GetTotalPriceForReservation(vehicleTypeID int, startTime, endTime time.Time) (int, error) {
	if !endTime.After(startTime) {
		return 0, fmt.Errorf("end_time must be after start_time")
	}
	unit, count, reservationTimeID := getBestUnitAndCount(startTime, endTime)
	pricePerUnit, err := s.Repo.GetPriceForUnit(vehicleTypeID, reservationTimeID)
	if err != nil {
		return 0, fmt.Errorf("could not get price per %s: %w", unit, err)
	}
	return pricePerUnit * count, nil
}

func (s *ReservationService) CreateReservation(req *entities.ReservationRequest) (*entities.StripeSessionResponse, error) {
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
		Status:          statusPending,
		StartTime:       req.StartTime,
		EndTime:         req.EndTime,
		Language:        req.Language,
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}

	sessionURL, err := s.handlePaymentIntent(req, reservation)
	if err != nil {
		return nil, err
	}

	err = s.Repo.CreateReservation(reservation)
	if err != nil {
		log.Printf("Error creating reservation in repository: %v", err)
		return nil, err
	}

	return &entities.StripeSessionResponse{
		Code:      code,
		URL:       sessionURL,
		SessionID: reservation.StripeSessionID}, nil
}

func (s *ReservationService) GetReservationByCode(code, email string) (*entities.ReservationResponse, error) {
	return s.Repo.GetReservationByCode(code, email)
}

func (s *ReservationService) CancelReservation(code string) error {
	reservation, err := s.Repo.GetReservationByCodeOnly(code)
	if err != nil {
		return err
	}
	sessionID := reservation.StripeSessionID
	if sessionID == "" {
		return fmt.Errorf("No Stripe session ID found for reservation code: %s", code)
	}
	reservationResp, err := s.GetReservationBySessionID(sessionID)
	if err != nil {
		return err
	}

	currentTime := time.Now().UTC()
	if reservation.StartTime.Sub(currentTime) < 12*time.Hour {
		log.Printf("Reservation can only be cancelled more than 12 hours before the start time")
		return fmt.Errorf("Reservations can only be cancelled more than 12 hours before the start time")
	}

	err = s.stripeService.RefundPaymentBySessionID(reservation.StripeSessionID)
	if err != nil {
		return err
	}

	_, err = s.Repo.CancelReservation(code)

	statusTraducido := statusTranslation(statusCancel, reservationResp.Language)
	s.SendReservationSMS(*reservationResp, statusTraducido)
	s.SendReservationEmail(*reservationResp, statusTraducido)
	return err
}

func (s *ReservationService) GetReservationBySessionID(sessionID string) (*entities.ReservationResponse, error) {
	reservation, err := s.Repo.GetReservationByStripeSessionID(sessionID)
	if err != nil {
		return nil, err
	}
	resp := &entities.ReservationResponse{
		Code:          reservation.Code,
		UserName:      reservation.UserName,
		UserEmail:     reservation.UserEmail,
		UserPhone:     reservation.UserPhone,
		VehicleTypeID: reservation.VehicleTypeID,
		VehiclePlate:  reservation.VehiclePlate,
		VehicleModel:  reservation.VehicleModel,
		Status:        reservation.Status,
		StartTime:     reservation.StartTime,
		EndTime:       reservation.EndTime,
		CreatedAt:     reservation.CreatedAt,
		UpdatedAt:     reservation.UpdatedAt,
		PaymentStatus: reservation.PaymentStatus,
		Language:      reservation.Language,
	}
	return resp, nil
}

func (s *ReservationService) UpdateReservationAndPaymentStatusBySessionID(sessionID, reservationStatus, paymentStatus string) error {
	reservation, err := s.Repo.GetReservationByStripeSessionID(sessionID)
	if err != nil {
		return err
	}
	return s.Repo.UpdateReservationAndPaymentStatus(reservation.ID, reservationStatus, paymentStatus)
}

// GetSessionIDByPaymentIntentID busca el session_id en Stripe a partir de un PaymentIntentID
func (s *ReservationService) GetSessionIDByPaymentIntentID(paymentIntentID string) (string, error) {
	params := &stripe.CheckoutSessionListParams{
		PaymentIntent: &paymentIntentID,
	}
	params.Limit = stripe.Int64(1)
	it := session.List(params)
	for it.Next() {
		sess := it.CheckoutSession()
		if sess != nil && sess.ID != "" {
			return sess.ID, nil
		}
	}
	return "", fmt.Errorf("No session_id found for PaymentIntentID %s", paymentIntentID)
}

func (s *ReservationService) SendReservationEmail(reservation entities.ReservationResponse, status string) {
	italyLoc, errLoc := time.LoadLocation("Europe/Rome")
	if errLoc != nil {
		italyLoc = time.FixedZone("CET", 1*60*60) // fallback CET
	}

	emailData := entities.ReservationEmailData{
		UserName:           reservation.UserName,
		ReservationCode:    reservation.Code,
		VehicleModel:       reservation.VehicleModel,
		VehiclePlate:       reservation.VehiclePlate,
		StartTimeFormatted: reservation.StartTime.In(italyLoc).Format("02 Jan 2006 15:04 MST"),
		EndTimeFormatted:   reservation.EndTime.In(italyLoc).Format("02 Jan 2006 15:04 MST"),
		CurrentYear:        time.Now().In(italyLoc).Year(),
		Language:           reservation.Language,
		Status:             status,
	}

	var emailSubject, plainTextBody string
	switch reservation.Language {
	case "es":
		emailSubject = fmt.Sprintf("Tu reserva en GreenParking está %s - Código: %s", status, emailData.ReservationCode)
		plainTextBody = fmt.Sprintf(
			"Hola %s,\n\nTu reserva en GreenParking está %s.\n\n"+
				"Detalles de la reserva:\n"+
				"Código de Reserva: %s\n"+
				"Vehículo: %s (Patente: %s)\n"+
				"Check-in: %s\n"+
				"Check-out: %s\n\n"+
				"Gracias por elegir GreenParking.\n\n"+
				"GreenParking. Todos los derechos reservados.",
			emailData.UserName, status, emailData.ReservationCode, emailData.VehicleModel, emailData.VehiclePlate,
			emailData.StartTimeFormatted, emailData.EndTimeFormatted, emailData.CurrentYear,
		)
	case "it":
		emailSubject = fmt.Sprintf("La tua prenotazione GreenParking è %s - Codice: %s", status, emailData.ReservationCode)
		plainTextBody = fmt.Sprintf(
			"Ciao %s,\n\nLa tua prenotazione presso GreenParking è %s.\n\n"+
				"Dettagli della prenotazione:\n"+
				"Codice prenotazione: %s\n"+
				"Veicolo: %s (Targa: %s)\n"+
				"Check-in: %s\n"+
				"Check-out: %s\n\n"+
				"Grazie per aver scelto GreenParking.\n\n"+
				"GreenParking. Tutti i diritti riservati.",
			emailData.UserName, status, emailData.ReservationCode, emailData.VehicleModel, emailData.VehiclePlate,
			emailData.StartTimeFormatted, emailData.EndTimeFormatted, emailData.CurrentYear,
		)
	default:
		emailSubject = fmt.Sprintf("Your GreenParking reservation is %s - Code: %s", status, emailData.ReservationCode)
		plainTextBody = fmt.Sprintf(
			"Hello %s,\n\nYour reservation at GreenPark is %s.\n\n"+
				"Reservation Details:\n"+
				"Reservation Code: %s\n"+
				"Vehicle: %s (Plate: %s)\n"+
				"Check-in: %s\n"+
				"Check-out: %s\n\n"+
				"Thank you for choosing GreenParking.\n\n"+
				"GreenParking. All rights reserved.",
			emailData.UserName, status, emailData.ReservationCode, emailData.VehicleModel, emailData.VehiclePlate,
			emailData.StartTimeFormatted, emailData.EndTimeFormatted, emailData.CurrentYear,
		)
	}

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

func (s *ReservationService) SendReservationSMS(reservation entities.ReservationResponse, status string) {
	italyLoc, errLoc := time.LoadLocation("Europe/Rome")
	if errLoc != nil {
		italyLoc = time.FixedZone("CET", 1*60*60)
	}

	userPhoneNumber := reservation.UserPhone
	reservationCode := reservation.Code

	var smsMessage string
	switch reservation.Language {
	case "es":
		smsMessage = fmt.Sprintf("GreenParking: ¡Tu reserva %s está %s!\nCheck-in: %s.\nMás detalles en tu correo.",
			reservationCode, status,
			reservation.StartTime.In(italyLoc).Format("02/01 15:04"),
		)
	case "it":
		smsMessage = fmt.Sprintf("GreenParking: La tua prenotazione %s è stata %s!\nCheck-in: %s.\nAltri dettagli nella tua email.",
			reservationCode, status,
			reservation.StartTime.In(italyLoc).Format("02/01 15:04"),
		)
	default:
		smsMessage = fmt.Sprintf("GreenParking: Reservation %s has been %s!\nCheck-in: %s.\nMore details in your email.",
			reservationCode, status,
			reservation.StartTime.In(italyLoc).Format("02/01 15:04"),
		)
	}

	errSMS := SendSMS(userPhoneNumber, smsMessage)
	if errSMS != nil {
		log.Printf("ALERTA: La reserva %s se creó, pero falló el envío del SMS de confirmación a %s: %v", reservationCode, userPhoneNumber, errSMS)
	}
}

func (s *ReservationService) handlePaymentIntent(req *entities.ReservationRequest, reservation *db.Reservation) (string, error) {
	var amount int64
	if req.PaymentMethodID == 2 { // online
		amount = int64(req.TotalPrice * 100)
	} else if req.PaymentMethodID == 1 { // onsite
		amount = int64(float64(req.TotalPrice) * 0.3 * 100)
	} else {
		return "", fmt.Errorf("Método de pago no soportado")
	}

	sessionURL, sessionID, err := s.stripeService.CreateCheckoutSession(amount, "eur", reservation.Code, req.UserEmail, reservation.Language)
	if err != nil {
		return "", err
	}

	reservation.StripeSessionID = sessionID
	reservation.PaymentStatus = statusPending

	return sessionURL, nil
}

func getBestUnitAndCount(startTime, endTime time.Time) (unit string, count int, reservationTimeID int) {
	d := endTime.Sub(startTime)
	if d.Hours() < 24 {
		// Less than 1 day, use hours
		count = int(d.Hours())
		if d.Minutes() > float64(count*60) {
			count++
		}
		if count == 0 {
			count = 1
		}
		return "hour", count, 1
	} else if d.Hours() < 24*7 {
		// Less than 1 week, use days
		count = int(d.Hours() / 24)
		if d.Hours() > float64(count*24) {
			count++
		}
		return "day", count, 2
	} else if d.Hours() < 24*30 {
		// Less than 1 month, use weeks
		count = int(d.Hours() / (24 * 7))
		if d.Hours() > float64(count*24*7) {
			count++
		}
		return "week", count, 3
	} else {
		// 1 month or more, use months
		count = int(d.Hours() / (24 * 30))
		if d.Hours() > float64(count*24*30) {
			count++
		}
		return "month", count, 4
	}
}

// statusTranslation traduce el status según idioma.
func statusTranslation(status, lang string) string {
	switch lang {
	case "es":
		switch status {
		case "pending":
			return "pendiente"
		case "active":
			return "activa"
		case "finished":
			return "finalizada"
		case "canceled", "cancelled":
			return "cancelada"
		}
	case "it":
		switch status {
		case "pending":
			return "in attesa"
		case "active":
			return "attiva"
		case "finished":
			return "finito"
		case "canceled", "cancelled":
			return "annullata"
		}
	}
	// Default: English
	return status
}
