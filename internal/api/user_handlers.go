package api

import (
	"database/sql"
	"encoding/json"
	"estacionamienti/internal/db"
	"estacionamienti/internal/service"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"strings"
	"time"
)

// Assume you have a global DB connection pool
var DB *sql.DB

// Helper: Parse ISO8601 time
func parseTime(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}

// POST /api/availability
func CheckAvailability(w http.ResponseWriter, r *http.Request) {
	var req AvailabilityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	entry, err1 := parseTime(req.EntryTime)
	exit, err2 := parseTime(req.ExitTime)
	if err1 != nil || err2 != nil || !exit.After(entry) || strings.TrimSpace(req.VehicleType) == "" {
		http.Error(w, "Invalid entry/exit times or vehicle type", http.StatusBadRequest)
		return
	}

	// Query available spaces
	var available int
	err := DB.QueryRow(
		"SELECT available_spaces FROM vehicle_spaces WHERE vehicle_type = $1",
		req.VehicleType,
	).Scan(&available)
	if err != nil {
		http.Error(w, "Vehicle type not found", http.StatusNotFound)
		return
	}

	resp := AvailabilityResponse{
		Available: available > 0,
		Message:   "Spot available",
	}
	if available <= 0 {
		resp.Message = "No spots available for this vehicle type."
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// POST /api/reservations
func CreateReservation(w http.ResponseWriter, r *http.Request) {
	var req CreateReservationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	entry, err1 := parseTime(req.EntryTime)
	exit, err2 := parseTime(req.ExitTime)
	if err1 != nil || err2 != nil || !exit.After(entry) {
		http.Error(w, "Invalid entry/exit times", http.StatusBadRequest)
		return
	}
	if req.FullName == "" || req.Email == "" || req.Phone == "" || req.LicensePlate == "" || req.VehicleModel == "" || req.VehicleType == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	// Check availability again
	var available int
	err := DB.QueryRow(
		"SELECT available_spaces FROM vehicle_spaces WHERE vehicle_type = $1",
		req.VehicleType,
	).Scan(&available)
	if err != nil || available <= 0 {
		http.Error(w, "No available spots for this vehicle type", http.StatusConflict)
		return
	}

	// Generate reservation code (simple random, improve for prod)
	code := fmt.Sprintf("%08X", time.Now().UnixNano()%100000000)

	// Insert reservation
	_, err = DB.Exec(`
        INSERT INTO reservations
        (reservation_code, entry_time, exit_time, vehicle_type, full_name, email, phone, license_plate, vehicle_model, status)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,'active')
    `,
		code, entry, exit, req.VehicleType, req.FullName, req.Email, req.Phone, req.LicensePlate, req.VehicleModel,
	)
	if err != nil {
		http.Error(w, "Could not create reservation", http.StatusInternalServerError)
		return
	}

	// Decrement available spaces
	_, err = DB.Exec(
		"UPDATE vehicle_spaces SET available_spaces = available_spaces - 1 WHERE vehicle_type = $1",
		req.VehicleType,
	)
	if err != nil {
		// Optionally: Rollback reservation
		http.Error(w, "Could not update available spaces", http.StatusInternalServerError)
		return
	}

	// Send email & SMS (mocked)
	go service.SendEmail(req.Email, "Your Reservation Code", "Your code: "+code)
	go service.SendSMS(req.Phone, "Your parking reservation code: "+code)

	resp := CreateReservationResponse{
		ReservationCode: code,
		Message:         "Reservation confirmed. Code sent via email and SMS.",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// GET /api/reservations/{code}
func GetReservation(w http.ResponseWriter, r *http.Request) {
	code := mux.Vars(r)["code"]
	var res db.Reservation
	err := DB.QueryRow(`
        SELECT id, reservation_code, entry_time, exit_time, vehicle_type, full_name, email, phone, license_plate, vehicle_model, status, created_at, updated_at
        FROM reservations WHERE reservation_code = $1
    `, code).Scan(
		&res.ID, &res.ReservationCode, &res.EntryTime, &res.ExitTime, &res.VehicleType, &res.FullName, &res.Email, &res.Phone, &res.LicensePlate, &res.VehicleModel, &res.Status, &res.CreatedAt, &res.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		http.Error(w, "Reservation not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	// Return reservation info (sanitize as needed)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

// PUT /api/reservations/{code}
func UpdateReservation(w http.ResponseWriter, r *http.Request) {
	code := mux.Vars(r)["code"]
	var req CreateReservationRequest // reuse struct for update
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	// Only allow updating personal info, not times or vehicle type
	_, err := DB.Exec(`
        UPDATE reservations SET full_name=$1, email=$2, phone=$3, license_plate=$4, vehicle_model=$5, updated_at=NOW()
        WHERE reservation_code=$6 AND status='active'
    `, req.FullName, req.Email, req.Phone, req.LicensePlate, req.VehicleModel, code)
	if err != nil {
		http.Error(w, "Could not update reservation", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Reservation updated"})
}

// DELETE /api/reservations/{code}
func CancelReservation(w http.ResponseWriter, r *http.Request) {
	code := mux.Vars(r)["code"]

	// Mark reservation as cancelled and increment available spaces
	tx, err := DB.Begin()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var vehicleType string
	err = tx.QueryRow(`SELECT vehicle_type FROM reservations WHERE reservation_code=$1 AND status='active'`, code).Scan(&vehicleType)
	if err == sql.ErrNoRows {
		http.Error(w, "Reservation not found or already cancelled", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec(`UPDATE reservations SET status='cancelled', updated_at=NOW() WHERE reservation_code=$1`, code)
	if err != nil {
		http.Error(w, "Could not cancel reservation", http.StatusInternalServerError)
		return
	}
	_, err = tx.Exec(`UPDATE vehicle_spaces SET available_spaces=available_spaces+1 WHERE vehicle_type=$1`, vehicleType)
	if err != nil {
		http.Error(w, "Could not update spaces", http.StatusInternalServerError)
		return
	}
	if err := tx.Commit(); err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Reservation cancelled"})
}
