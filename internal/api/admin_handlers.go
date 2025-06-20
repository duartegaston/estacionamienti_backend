package api

import (
	"encoding/json"
	"estacionamienti/internal/service"
	"net/http"

	"github.com/gorilla/mux"
)

type AdminHandler struct {
	adminService *service.AdminService
}

func NewAdminHandler(svc *service.AdminService) *AdminHandler {
	return &AdminHandler{adminService: svc}
}

func (h *AdminHandler) ListReservations(w http.ResponseWriter, r *http.Request) {
	date := r.URL.Query().Get("date")
	vehicleType := r.URL.Query().Get("vehicle_type")
	status := r.URL.Query().Get("status")
	reservations, err := h.adminService.ListReservations(date, vehicleType, status)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(reservations)
}

// TO DO
func (h *AdminHandler) CreateReservation(w http.ResponseWriter, r *http.Request) {
	return
}

func (h *AdminHandler) AdminDeleteReservation(w http.ResponseWriter, r *http.Request) {
	code := mux.Vars(r)["code"]
	err := h.adminService.CancelReservation(code)
	if err != nil {
		http.Error(w, "Could not delete reservation", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"message": "Reservation deleted"})
}

func (h *AdminHandler) ListVehicleSpaces(w http.ResponseWriter, r *http.Request) {
	spaces, err := h.adminService.ListVehicleSpaces()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(spaces)
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
	err := h.adminService.UpdateVehicleSpaces(vehicleType, req.TotalSpaces, req.AvailableSpaces)
	if err != nil {
		http.Error(w, "Could not update vehicle spaces", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"message": "Vehicle spaces updated"})
}
