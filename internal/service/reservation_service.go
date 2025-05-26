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

func (s *ReservationService) CheckAvailability(req entities.ReservationRequest) (*entities.AvailabilityResponse, error) {
	if !req.EndTime.After(req.StartTime) {
		return &entities.AvailabilityResponse{
			IsOverallAvailable: false,
			RequestedStartTime: req.StartTime,
			RequestedEndTime:   req.EndTime,
			Message:            "La fecha/hora de fin debe ser posterior a la fecha/hora de inicio.",
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
		response.Message = "No se pudo determinar la disponibilidad para el tipo de vehículo o rango solicitado. Verifique la configuración o el rango."
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

	if response.IsOverallAvailable {
		response.Message = "El período solicitado está completamente disponible."
	} else {
		response.Message = "Algunos horarios dentro del período solicitado no están disponibles. Por favor, revise los detalles."
		response.FirstUnavailableSlotStart = firstUnavailableTime
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
