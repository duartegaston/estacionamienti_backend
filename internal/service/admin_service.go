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
	s.senderService.SendReservationSMS(*reservationResponse, statusActive)
	s.senderService.SendReservationEmail(*reservationResponse, statusActive)

	return reservationResponse, nil
}

// todo

func (s *AdminService) CancelReservation(code string) error {
	return nil
}

func (s *AdminService) ListVehicleSpaces() ([]db.VehicleSpace, error) {
	return s.adminRepo.ListVehicleSpaces()
}

func (s *AdminService) UpdateVehicleSpaces(vehicleType string, totalSpaces, availableSpaces int) error {
	return s.adminRepo.UpdateVehicleSpaces(vehicleType, totalSpaces, availableSpaces)
}
