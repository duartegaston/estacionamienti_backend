package service

import (
	"estacionamienti/internal/db"
	"estacionamienti/internal/entities"
	"estacionamienti/internal/repository"
	"fmt"
	"log"
	"time"
)

type ReservationService struct {
	Repo *repository.ReservationRepository
}

func NewReservationService(repo *repository.ReservationRepository) *ReservationService {
	return &ReservationService{Repo: repo}
}

func (s *ReservationService) CheckAvailability(req entities.ReservationRequest) (bool, error) {
	available, err := s.Repo.CheckAvailability(req)
	if err != nil {
		return false, err
	}
	return available > 0, nil
}

func (s *ReservationService) CreateReservation(req *entities.ReservationRequest) (string, error) {
	var vehicleTypeID int
	err := s.Repo.GetVehicleTypeIDByName(req.VehicleType, &vehicleTypeID)
	if err != nil {
		return "", fmt.Errorf("vehicle type not found: %w", err)
	}
	//TODO en base al payment_method obtener el ID al igual que tipo de vehiculo

	code := fmt.Sprintf("%08X", time.Now().UnixNano()%100000000)

	reservation := &db.Reservation{
		Code:          code,
		UserName:      req.UserName,
		UserEmail:     req.UserEmail,
		UserPhone:     req.UserPhone,
		VehicleTypeID: vehicleTypeID,
		VehiclePlate:  req.VehiclePlate,
		VehicleModel:  req.VehicleModel,
		PaymentMethod: req.PaymentMethod, // TODO: guardar el ID
		Status:        "active",
		StartTime:     req.StartTime,
		EndTime:       req.EndTime,
	}

	err = s.Repo.CreateReservation(reservation)
	if err != nil {
		return "", err
	}

	return code, nil
}

func (s *ReservationService) GetReservationByCode(code, email string) (*db.Reservation, error) {
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
