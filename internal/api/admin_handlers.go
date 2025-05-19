package api

import (
	"encoding/json"
	"estacionamienti/internal/service"
	"github.com/gorilla/mux"
	"net/http"
)

type AdminHandler struct {
	Service *service.ReservationService
}

func NewAdminHandler(svc *service.ReservationService) *AdminHandler {
	return &AdminHandler{Service: svc}
}

func (h *AdminHandler) ListReservations(w http.ResponseWriter, r *http.Request) {
	date := r.URL.Query().Get("date")
	vehicleType := r.URL.Query().Get("vehicle_type")
	status := r.URL.Query().Get("status")
	reservations, err := h.Service.ListReservations(date, vehicleType, status)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(reservations)
}

func (h *AdminHandler) AdminDeleteReservation(w http.ResponseWriter, r *http.Request) {
	code := mux.Vars(r)["code"]
	err := h.Service.CancelReservation(code)
	if err != nil {
		http.Error(w, "Could not delete reservation", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"message": "Reservation deleted"})
}

func (h *AdminHandler) UpdateVehicleSpaces(w http.ResponseWriter, r *http.Request) {
	vehicleType := mux.Vars(r)["vehicle_type"]
	var req struct {
		TotalSpaces     int `json:"total_spaces"`
		AvailableSpaces int `json:"available_spaces"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	err := h.Service.UpdateVehicleSpaces(vehicleType, req.TotalSpaces, req.AvailableSpaces)
	if err != nil {
		http.Error(w, "Could not update vehicle spaces", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"message": "Vehicle spaces updated"})
}

func (h *AdminHandler) ListVehicleSpaces(w http.ResponseWriter, r *http.Request) {
	spaces, err := h.Service.ListVehicleSpaces()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(spaces)
}
