package service

import (
	"estacionamienti/internal/db"
	"estacionamienti/internal/repository"
)

type AdminService struct {
	adminRepo       *repository.AdminRepository
	reservationRepo *repository.ReservationRepository
	stripeService   *StripeService
}

func NewAdminService(adminRepo *repository.AdminRepository, reservationRepo *repository.ReservationRepository, stripeService *StripeService) *AdminService {
	return &AdminService{adminRepo: adminRepo,
		stripeService:   stripeService,
		reservationRepo: reservationRepo}
}

func (s *AdminService) ListReservations(date, vehicleType, status string) ([]db.Reservation, error) {
	return s.adminRepo.ListReservations(date, vehicleType)
}

func (s *AdminService) CancelReservation(code string) error {
	return nil
}

func (s *AdminService) ListVehicleSpaces() ([]db.VehicleSpace, error) {
	return s.adminRepo.ListVehicleSpaces()
}

func (s *AdminService) UpdateVehicleSpaces(vehicleType string, totalSpaces, availableSpaces int) error {
	return s.adminRepo.UpdateVehicleSpaces(vehicleType, totalSpaces, availableSpaces)
}
