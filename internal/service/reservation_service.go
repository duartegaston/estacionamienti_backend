package service

import (
	"errors"
	"estacionamienti/internal/db"
	"estacionamienti/internal/repository"
	"fmt"
	"time"
)

type ReservationService struct {
	Repo *repository.ReservationRepository
}

func NewReservationService(repo *repository.ReservationRepository) *ReservationService {
	return &ReservationService{Repo: repo}
}

func (s *ReservationService) ListReservations(date, vehicleType string) ([]db.Reservation, error) {
	return s.Repo.ListReservations(date, vehicleType)
}

func (s *ReservationService) GetReservationByCode(code string) (*db.Reservation, error) {
	return s.Repo.GetReservationByCode(code)
}

func (s *ReservationService) CreateReservation(req *db.Reservation) (string, error) {
	available, err := s.Repo.CheckAvailability(req.VehicleType)
	if err != nil {
		return "", err
	}
	if available <= 0 {
		return "", errors.New("no available spots")
	}

	// Generar código de reserva
	code := fmt.Sprintf("%08X", time.Now().UnixNano()%100000000)
	req.ReservationCode = code
	req.Status = "active"

	// Crear la reserva
	err = s.Repo.CreateReservation(req)
	if err != nil {
		return "", err
	}

	// Obtener vehicle_type_id
	var vehicleTypeID int
	err = s.Repo.DB.QueryRow(`
		SELECT id FROM vehicle_types WHERE name = $1
	`, req.VehicleType).Scan(&vehicleTypeID)
	if err != nil {
		return "", fmt.Errorf("error getting vehicle_type_id: %w", err)
	}

	// Decrementar available_spaces usando vehicle_type_id
	_, err = s.Repo.DB.Exec(`
		UPDATE vehicle_spaces
		SET available_spaces = available_spaces - 1
		WHERE vehicle_type_id = $1
	`, vehicleTypeID)
	if err != nil {
		return "", err
	}

	return code, nil
}

func (s *ReservationService) UpdateReservationByCode(code string, updates map[string]interface{}) error {
	return s.Repo.UpdateReservationByCode(code, updates)
}

func (s *ReservationService) CancelReservation(code string) error {
	_, err := s.Repo.CancelReservation(code)
	return err
}

func (s *ReservationService) UpdateReservationByID(id int, res *db.Reservation) error {
	return s.Repo.UpdateReservationByID(id, res)
}

func (s *ReservationService) DeleteReservationByID(id int) error {
	return s.Repo.DeleteReservationByID(id)
}

func (s *ReservationService) UpdateVehicleSpaces(vehicleType string, totalSpaces, availableSpaces int) error {
	return s.Repo.UpdateVehicleSpaces(vehicleType, totalSpaces, availableSpaces)
}

func (s *ReservationService) CheckAvailability(vehicleType string) (bool, error) {
	available, err := s.Repo.CheckAvailability(vehicleType)
	if err != nil {
		return false, err
	}
	return available > 0, nil
}

func (s *ReservationService) ListVehicleSpaces() ([]db.VehicleSpace, error) {
	return s.Repo.ListVehicleSpaces()
}
