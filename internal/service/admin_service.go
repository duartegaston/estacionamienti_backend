package service

import (
	"estacionamienti/internal/db"
	"estacionamienti/internal/repository"
	"fmt"
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

// Logica para capturar el pago si no se pago con tarjeta
func (s *AdminService) CaptureReservationPayment(code string) error {
	reservation, err := s.reservationRepo.GetReservationByCodeOnly(code)
	if err != nil {
		return err
	}

	// Ver si ademas valido que haya finalizado la reserva.
	if reservation.PaymentStatus != statusRequiresCapture {
		return fmt.Errorf("Payment cannot be captured in status: %s", reservation.PaymentStatus)
	}
	err = s.stripeService.CapturePaymentIntent(reservation.StripePaymentIntentID)
	if err != nil {
		return err
	}

	// Update payment status
	intent, err := s.stripeService.GetPaymentIntent(reservation.StripePaymentIntentID)
	if err != nil {
		return err
	}
	reservation.PaymentStatus = string(intent.Status)
	return s.reservationRepo.UpdatePaymentStatus(reservation.ID, reservation.PaymentStatus)
}
