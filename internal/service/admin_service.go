package service

import (
	"estacionamienti/internal/db"
	"estacionamienti/internal/repository"
)

type AdminService struct {
	Repo *repository.AdminRepository
}

func NewAdminService(repo *repository.AdminRepository) *AdminService {
	return &AdminService{Repo: repo}
}

func (s *AdminService) ListReservations(date, vehicleType, status string) ([]db.Reservation, error) {
	return s.Repo.ListReservations(date, vehicleType)
}

func (s *AdminService) CancelReservation(code string) error {
	return nil
}

func (s *AdminService) ListVehicleSpaces() ([]db.VehicleSpace, error) {
	return s.Repo.ListVehicleSpaces()
}

func (s *AdminService) UpdateVehicleSpaces(vehicleType string, totalSpaces, availableSpaces int) error {
	return s.Repo.UpdateVehicleSpaces(vehicleType, totalSpaces, availableSpaces)
}
