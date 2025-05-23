package api

import (
	"encoding/json"
	"estacionamienti/internal/entities"
	"estacionamienti/internal/service"
	"net/http"

	"github.com/gorilla/mux"
)

type UserReservationHandler struct {
	Service *service.ReservationService
}

func NewUserReservationHandler(svc *service.ReservationService) *UserReservationHandler {
	return &UserReservationHandler{Service: svc}
}

// TO DO
func (h *UserReservationHandler) CheckAvailability(w http.ResponseWriter, r *http.Request) {
	var req entities.ReservationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	available, err := h.Service.CheckAvailability(req)
	if err != nil {
		http.Error(w, "Error checking availability", http.StatusInternalServerError)
		return
	}
	// TODO: Modificar el response para mostrar los horarios disponibles dentro de lo que el user quiere reserva
	json.NewEncoder(w).Encode(map[string]interface{}{
		"available": available,
	})
}

// TO DO
func (h *UserReservationHandler) CreateReservation(w http.ResponseWriter, r *http.Request) {
	var req entities.ReservationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	code, err := h.Service.CreateReservation(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"reservation_code": code,
		"message":          "Reservation confirmed.",
	})
}

// TO DO en vez de devolver el id del vehicle type, devolver el name
func (h *UserReservationHandler) GetReservation(w http.ResponseWriter, r *http.Request) {
	code := mux.Vars(r)["code"]
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	res, err := h.Service.GetReservationByCode(code, req.Email)
	if err != nil {
		http.Error(w, "Get reservation not found", http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(res)
}

// TO DO
func (h *UserReservationHandler) CancelReservation(w http.ResponseWriter, r *http.Request) {
	code := mux.Vars(r)["code"]
	err := h.Service.CancelReservation(code)
	if err != nil {
		http.Error(w, "Could not cancel reservation", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"message": "Reservation cancelled"})
}

func (h *UserReservationHandler) GetPrices(w http.ResponseWriter, r *http.Request) {
	res, err := h.Service.GetPrices()
	if err != nil {
		http.Error(w, "Could not get prices", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(res)
}

// TO DO
func (h *UserReservationHandler) GetTotalPriceForReservation(w http.ResponseWriter, r *http.Request) {
	return
}

func (h *UserReservationHandler) GetVehicleTypes(w http.ResponseWriter, r *http.Request) {
	res, err := h.Service.GetVehicleTypes()
	if err != nil {
		http.Error(w, "Could not get vehicle types", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(res)
}
