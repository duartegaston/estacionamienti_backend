package service

import (
	"estacionamienti/internal/db"
	"estacionamienti/internal/entities"
	"estacionamienti/internal/errors"
	"estacionamienti/internal/repository"
	"fmt"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"log"
	"time"
)

const (
	statusActive  = "active"
	statusPending = "pending"
	statusCancel  = "canceled"
)

type ReservationService struct {
	stripeService *StripeService
	Repo          *repository.ReservationRepository
	senderService *SenderService
}

func NewReservationService(repo *repository.ReservationRepository, stripeService *StripeService, senderService *SenderService) *ReservationService {
	return &ReservationService{Repo: repo,
		stripeService: stripeService,
		senderService: senderService}
}

func (s *ReservationService) GetPrices() ([]entities.PriceResponse, error) {
	return s.Repo.GetPrices()
}

func (s *ReservationService) GetVehicleTypes() ([]db.VehicleType, error) {
	return s.Repo.GetVehicleTypes()
}

func (s *ReservationService) CheckAvailability(req entities.ReservationRequest) (*entities.AvailabilityResponse, error) {
	// You need vehicle type name for mapping
	var vehicleTypeName string
	vehicleTypes, err := s.Repo.GetVehicleTypes()
	if err == nil {
		for _, vt := range vehicleTypes {
			if vt.ID == req.VehicleTypeID {
				vehicleTypeName = vt.Name
				break
			}
		}
	}

	hourlyDetails, err := s.Repo.GetHourlyAvailabilityDetails(req.StartTime, req.EndTime, req.VehicleTypeID, vehicleTypeName)
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
		TotalPrice:      req.TotalPrice,
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
	log.Printf("Canceling reservation with code: %s", code)
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
		return errors.ErrUnauthorized("Reservations can only be cancelled more than 12 hours before the start time")
	}

	err = s.stripeService.RefundPaymentBySessionID(reservation.StripeSessionID)
	if err != nil {
		return err
	}

	_, err = s.Repo.CancelReservation(code)

	statusTraducido := s.senderService.StatusTranslation(statusCancel, reservationResp.Language)
	s.senderService.SendReservationSMS(*reservationResp, statusTraducido)
	s.senderService.SendReservationEmail(*reservationResp, statusTraducido)
	return err
}

func (s *ReservationService) GetReservationBySessionID(sessionID string) (*entities.ReservationResponse, error) {
	reservation, err := s.Repo.GetReservationByStripeSessionID(sessionID)
	if err != nil {
		return nil, err
	}
	resp := &entities.ReservationResponse{
		Code:            reservation.Code,
		UserName:        reservation.UserName,
		UserEmail:       reservation.UserEmail,
		UserPhone:       reservation.UserPhone,
		VehicleTypeID:   reservation.VehicleTypeID,
		VehiclePlate:    reservation.VehiclePlate,
		VehicleModel:    reservation.VehicleModel,
		PaymentMethodID: reservation.PaymentMethodID,
		Status:          reservation.Status,
		StartTime:       reservation.StartTime,
		EndTime:         reservation.EndTime,
		CreatedAt:       reservation.CreatedAt,
		UpdatedAt:       reservation.UpdatedAt,
		PaymentStatus:   reservation.PaymentStatus,
		Language:        reservation.Language,
		TotalPrice:      reservation.TotalPrice,
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

func (s *ReservationService) UpdateReservationStatusPaymentAndIntentBySessionID(sessionID, reservationStatus, paymentStatus, paymentIntentID string) error {
	reservation, err := s.Repo.GetReservationByStripeSessionID(sessionID)
	if err != nil {
		return err
	}
	return s.Repo.UpdateReservationStatusPaymentAndIntent(reservation.ID, reservationStatus, paymentStatus, paymentIntentID)
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
