package service

import (
	"bytes"
	"estacionamienti/internal/db"
	"estacionamienti/internal/entities"
	"estacionamienti/internal/repository"
	"fmt"
	"github.com/stripe/stripe-go/v82"

	"html/template"
	"log"
	"path/filepath"
	"time"
)

const (
	statusActive          = "active"
	confirmed             = "confirmed"
	statusRequiresCapture = "requires_capture"
	statusSucceeded       = "succeeded"
	statusCancel          = "canceled"
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

func (s *ReservationService) CreateReservation(req *entities.ReservationRequest) (reservationResponse *entities.ReservationResponse, err error) {
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
		Status:          statusActive,
		StartTime:       req.StartTime,
		EndTime:         req.EndTime,
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}

	err = s.handlePaymentIntent(req, reservation)
	if err != nil {
		return nil, err
	}

	err = s.Repo.CreateReservation(reservation)
	if err != nil {
		log.Printf("Error creating reservation in repository: %v", err)
		return reservationResponse, err
	}

	reservationResponse, err = s.Repo.GetReservationByCode(code, req.UserEmail)
	if err != nil {
		log.Printf("Error from GetReservationByCode: %v", err)
		return reservationResponse, fmt.Errorf("internal error creating reservation: %w", err)
	}

	return reservationResponse, nil
}

func (s *ReservationService) GetReservationByCode(code, email string) (*entities.ReservationResponse, error) {
	return s.Repo.GetReservationByCode(code, email)
}

func (s *ReservationService) CancelReservation(code string) error {
	reservation, err := s.Repo.GetReservationByCodeOnly(code)
	if err != nil {
		return err
	}

	currentTime := time.Now().UTC()
	if reservation.StartTime.Sub(currentTime) < 12*time.Hour {
		log.Printf("Reservation can only be cancelled more than 12 hours before the start time")
		return fmt.Errorf("Reservations can only be cancelled more than 12 hours before the start time")
	}

	if reservation.PaymentMethodID == 1 {
		// Autorizado, pero no capturado: cancelarlo
		log.Printf("Canceling payment intent: %s", reservation.StripePaymentIntentID)
		err := s.stripeService.CancelPaymentIntent(reservation.StripePaymentIntentID)
		if err != nil {
			return err
		}
	} else if reservation.PaymentMethodID == 2 {
		// Pagado: hacer refund
		err := s.stripeService.RefundPayment(reservation.StripePaymentIntentID)
		if err != nil {
			return err
		}
	}

	_, err = s.Repo.CancelReservation(code)

	s.SendReservationSMS(*reservation, statusCancel)
	s.SendReservationEmail(*reservation, statusCancel)
	return err
}

func (s *ReservationService) UpdateReservationAndPaymentStatusByStripeID(paymentIntentID, reservationStatus, paymentStatus string) error {
	reservation, err := s.Repo.GetReservationByStripePaymentIntentID(paymentIntentID)
	if err != nil {
		return err
	}
	return s.Repo.UpdateReservationAndPaymentStatus(reservation.ID, reservationStatus, paymentStatus)
}

func (s *ReservationService) GerReservationByPaymentIntentID(paymentIntentID string) (*db.Reservation, error) {
	reservation, err := s.Repo.GetReservationByStripePaymentIntentID(paymentIntentID)
	if err != nil {
		return nil, err
	}
	return reservation, nil
}

func (s *ReservationService) SendReservationEmail(reservation db.Reservation, status string) {
	emailData := entities.ReservationEmailData{
		UserName:           reservation.UserName,
		ReservationCode:    reservation.Code,
		VehicleModel:       reservation.VehicleModel,
		VehiclePlate:       reservation.VehiclePlate,
		StartTimeFormatted: reservation.StartTime.Format("02 Jan 2006 15:04 MST"),
		EndTimeFormatted:   reservation.EndTime.Format("02 Jan 2006 15:04 MST"),
		CurrentYear:        time.Now().Year(),
	}

	emailSubject := fmt.Sprintf("Your GreenParking reservation %s - Code: %s", status, emailData.ReservationCode)

	plainTextBody := fmt.Sprintf(
		"Hello %s,\n\nYour reservation at GreenPark has been %s.\n\n"+
			"Reservation Details:\n"+
			"Reservation Code: %s\n"+
			"Vehicle: %s (Plate: %s)\n"+
			"Check-in: %s\n"+
			"Check-out: %s\n\n"+
			"Thank you for choosing GreenParking.\n\n"+
			" GreenParking. All rights reserved.",
		status, emailData.UserName, emailData.ReservationCode, emailData.VehicleModel, emailData.VehiclePlate,
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

func (s *ReservationService) SendReservationSMS(reservation db.Reservation, status string) {
	userPhoneNumber := reservation.UserPhone
	reservationCode := reservation.Code
	smsMessage := fmt.Sprintf("GreenParking: Reservation %s has been %s!\nCheck-in: %s.\nMore details in your email.",
		reservationCode, status,
		reservation.StartTime.Format("02/01 15:04"),
	)

	errSMS := SendSMS(userPhoneNumber, smsMessage)
	if errSMS != nil {
		log.Printf("ALERTA: La reserva %s se creó, pero falló el envío del SMS de confirmación a %s: %v", reservationCode, userPhoneNumber, errSMS)
	}
}

func (s *ReservationService) handlePaymentIntent(req *entities.ReservationRequest, reservation *db.Reservation) error {
	var paymentIntent *stripe.PaymentIntent
	var err error

	if req.PaymentMethodID == 2 { // 2 = online (pay now)
		paymentIntent, err = s.stripeService.CreatePaymentIntent(int64(req.TotalPrice*100), "eur", reservation.Code)
		if err != nil {
			return err
		}
	} else if req.PaymentMethodID == 1 { // 1 = onsite (guarantee)
		paymentIntent, err = s.stripeService.CreatePaymentIntentWithManualCapture(int64(req.TotalPrice*100), "eur", reservation.Code)
		s.SendReservationEmail(*reservation, confirmed)
		s.SendReservationSMS(*reservation, confirmed)
		if err != nil {
			return err
		}
	}

	if paymentIntent != nil {
		reservation.StripePaymentIntentID = paymentIntent.ID
		reservation.PaymentStatus = string(paymentIntent.Status)
		if paymentIntent.Customer != nil {
			reservation.StripeCustomerID = paymentIntent.Customer.ID
		}
		if paymentIntent.PaymentMethod != nil {
			reservation.StripePaymentMethodID = paymentIntent.PaymentMethod.ID
		}
	}
	return nil
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
