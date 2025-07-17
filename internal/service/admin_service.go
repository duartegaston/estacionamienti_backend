package service

import (
	"estacionamienti/internal/db"
	"estacionamienti/internal/entities"
	"estacionamienti/internal/repository"
	"fmt"
	"log"
	"time"
)

type AdminService struct {
	adminRepo       *repository.AdminRepository
	reservationRepo *repository.ReservationRepository
	stripeService   *StripeService
	senderService   *SenderService
}

func NewAdminService(adminRepo *repository.AdminRepository, reservationRepo *repository.ReservationRepository, stripeService *StripeService, senderService *SenderService) *AdminService {
	return &AdminService{adminRepo: adminRepo,
		stripeService:   stripeService,
		reservationRepo: reservationRepo,
		senderService:   senderService}
}

func (s *AdminService) ListReservations(startTime, endTime, vehicleType, status, limit, offset string) ([]entities.ReservationResponse, error) {
	return s.adminRepo.ListReservationsWithFilters(startTime, endTime, vehicleType, status, limit, offset)
}

func (s *AdminService) CreateReservation(reservationReq *entities.ReservationRequest) (reservationResponse *entities.ReservationResponse, err error) {
	code := fmt.Sprintf("%08X", time.Now().UnixNano()%100000000)

	reservation := &db.Reservation{
		Code:            code,
		UserName:        reservationReq.UserName,
		UserEmail:       reservationReq.UserEmail,
		UserPhone:       reservationReq.UserPhone,
		VehicleTypeID:   reservationReq.VehicleTypeID,
		VehiclePlate:    reservationReq.VehiclePlate,
		VehicleModel:    reservationReq.VehicleModel,
		PaymentMethodID: reservationReq.PaymentMethodID,
		Status:          statusActive,
		StartTime:       reservationReq.StartTime,
		EndTime:         reservationReq.EndTime,
		Language:        reservationReq.Language,
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}

	err = s.reservationRepo.CreateReservation(reservation)
	if err != nil {
		log.Printf("Error creating reservation in repository: %v", err)
		return nil, err
	}

	reservationResponse, err = s.adminRepo.FindReservationByCode(code)
	if err != nil {
		log.Printf("Error getting reservation from repository: %v", err)
		return nil, err
	}
	statusTraducido := s.senderService.StatusTranslation(statusActive, reservation.Language)
	s.senderService.SendReservationSMS(*reservationResponse, statusTraducido)
	s.senderService.SendReservationEmail(*reservationResponse, statusTraducido)

	return reservationResponse, nil
}

func (s *AdminService) CancelReservation(code string) error {
	reservation, err := s.reservationRepo.GetReservationByCodeOnly(code)
	if err != nil {
		return err
	}
	log.Printf("Canceling reservation with code: %s", code)
	sessionID := reservation.StripeSessionID
	currentTime := time.Now().UTC()

	// Si la session de stripe no está, se puede cancelar (Quiere decir que nunca hubo pago por stripe)
	// O si la reserva ya empezó, se cancela y no se devuelve el dinero
	if sessionID == "" || reservation.StartTime.Before(currentTime) {
		log.Printf("Canceling reservation with code: %s", code)
		_, err = s.reservationRepo.CancelReservation(code)
		if err != nil {
			return err
		}
		return nil
	}
	err = s.stripeService.RefundPaymentBySessionID(reservation.StripeSessionID)
	if err != nil {
		return err
	}

	_, err = s.reservationRepo.CancelReservation(code)

	statusTraducido := s.senderService.StatusTranslation(statusCancel, reservation.Language)
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
	}
	s.senderService.SendReservationSMS(*resp, statusTraducido)
	s.senderService.SendReservationEmail(*resp, statusTraducido)
	return err
}

// todo

func (s *AdminService) ListVehicleSpaces() ([]db.VehicleSpace, error) {
	return s.adminRepo.ListVehicleSpaces()
}

func (s *AdminService) UpdateVehicleSpaces(vehicleType string, totalSpaces, availableSpaces int) error {
	return s.adminRepo.UpdateVehicleSpaces(vehicleType, totalSpaces, availableSpaces)
}
