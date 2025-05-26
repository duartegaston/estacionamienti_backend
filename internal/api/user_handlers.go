package api

import (
	"encoding/json"
	"estacionamienti/internal/entities"
	"estacionamienti/internal/service"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

type UserReservationHandler struct {
	Service *service.ReservationService
}

func NewUserReservationHandler(svc *service.ReservationService) *UserReservationHandler {
	return &UserReservationHandler{Service: svc}
}

func (h *UserReservationHandler) CheckAvailability(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	startTimeStr := queryParams.Get("startTime")
	endTimeStr := queryParams.Get("endTime")
	vehicleTypeIDStr := queryParams.Get("vehicleTypeId")

	if startTimeStr == "" {
		http.Error(w, "Query parameter 'startTime' is required", http.StatusBadRequest)
		return
	}
	startTime, err := time.Parse(time.RFC3339, startTimeStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid 'startTime' format. Please use RFC3339 format (e.g., YYYY-MM-DDTHH:MM:SSZ): %v", err), http.StatusBadRequest)
		return
	}

	if endTimeStr == "" {
		http.Error(w, "Query parameter 'endTime' is required", http.StatusBadRequest)
		return
	}
	endTime, err := time.Parse(time.RFC3339, endTimeStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid 'endTime' format. Please use RFC3339 format (e.g., YYYY-MM-DDTHH:MM:SSZ): %v", err), http.StatusBadRequest)
		return
	}

	if vehicleTypeIDStr == "" {
		http.Error(w, "Query parameter 'vehicleTypeId' is required", http.StatusBadRequest)
		return
	}
	vehicleTypeID, err := strconv.Atoi(vehicleTypeIDStr)
	if err != nil {
		http.Error(w, "Invalid 'vehicleTypeId' format. It must be an integer.", http.StatusBadRequest)
		return
	}
	if vehicleTypeID <= 0 {
		http.Error(w, "'vehicleTypeId' must be a positive integer.", http.StatusBadRequest)
		return
	}

	if !endTime.After(startTime) {
		http.Error(w, "'endTime' must be after 'startTime'.", http.StatusBadRequest)
		return
	}

	minDuration := 1 * time.Hour
	requestedDuration := endTime.Sub(startTime)
	if requestedDuration < minDuration {
		http.Error(w, fmt.Sprintf("Minimum reservation duration is 1 hour. Requested duration is %v.", requestedDuration), http.StatusBadRequest)
		return
	}

	availabilityReq := entities.ReservationRequest{
		StartTime:     startTime,
		EndTime:       endTime,
		VehicleTypeID: vehicleTypeID,
	}

	availabilityResponse, err := h.Service.CheckAvailability(availabilityReq)
	if err != nil {
		log.Printf("Error from CheckAvailability service: %v", err)
		http.Error(w, "An error occurred while checking availability.", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(availabilityResponse); err != nil {
		log.Printf("Error encoding availability response: %v", err)
	}
}

// TO DO hacer cobro, enviar sms y email
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

// TO DO devolver cobro, enviar sms y email
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
