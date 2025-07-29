package service

import (
	"estacionamienti/internal/db"
	"estacionamienti/internal/entities"
	"estacionamienti/internal/repository"
	"fmt"
	"log"
	"strings"
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

func (s *AdminService) ListReservations(startTime, endTime, code, vehicleType, status, limit, offset string) (entities.ReservationsList, error) {
	reservationList, err := s.adminRepo.ListReservationsWithFilters(startTime, endTime, code, vehicleType, status, limit, offset)
	if err != nil {
		log.Printf("Error listing reservations: %v", err)
		return entities.ReservationsList{}, err
	}
	return reservationList, nil
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
		TotalPrice:      reservationReq.TotalPrice,
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

func (s *AdminService) CancelReservation(code string, refund bool) error {
	reservation, err := s.reservationRepo.GetReservationByCodeOnly(code)
	if err != nil {
		log.Printf("Error canceling reservation: %v", err)
		return err
	}
	sessionID := reservation.StripeSessionID
	// Si la session de stripe no está, se puede cancelar (Quiere decir que nunca hubo pago por stripe)
	if sessionID == "" {
		_, err = s.reservationRepo.CancelReservation(code)
		if err != nil {
			return err
		}
		return nil
	}
	if refund {
		err = s.stripeService.RefundPaymentBySessionID(sessionID)
		if err != nil {
			if strings.Contains(err.Error(), "charge_already_refunded") {
				log.Printf("Charge already refunded: %v", err)
			} else {
				return err
			}
		}
	}
	_, err = s.reservationRepo.CancelReservation(code)

	return err
}

func (s *AdminService) ListVehicleSpaces() ([]db.VehicleSpaceWithPrices, error) {
	return s.adminRepo.ListVehicleSpaces()
}

func (s *AdminService) UpdateVehicleSpacesAndPrices(vehicleType string, spaces int, prices map[string]float32) error {
	err := s.adminRepo.UpdateVehicleSpaces(vehicleType, spaces)
	if err != nil {
		return err
	}
	for timeName, price := range prices {
		err := s.adminRepo.UpdateVehiclePrice(vehicleType, timeName, price)
		if err != nil {
			log.Printf("[AdminService] Error updating price for vehicleType '%s', time '%s': %v", vehicleType, timeName, err)
			return err
		}
	}
	return nil
}
