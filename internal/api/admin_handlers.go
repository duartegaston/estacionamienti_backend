package api

import (
	"encoding/json"
	"estacionamienti/internal/db"
	"github.com/gorilla/mux"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// GET /admin/reservations?date=YYYY-MM-DD&vehicle_type=car
func ListReservations(w http.ResponseWriter, r *http.Request) {
	query := "SELECT id, reservation_code, entry_time, exit_time, vehicle_type, full_name, email, phone, license_plate, vehicle_model, status, created_at, updated_at FROM reservations WHERE 1=1"
	args := []interface{}{}
	idx := 1

	// Filtering
	date := r.URL.Query().Get("date")
	if date != "" {
		query += " AND DATE(entry_time) = $" + strconv.Itoa(idx)
		args = append(args, date)
		idx++
	}
	vehicleType := r.URL.Query().Get("vehicle_type")
	if vehicleType != "" {
		query += " AND vehicle_type = $" + strconv.Itoa(idx)
		args = append(args, vehicleType)
		idx++
	}
	query += " ORDER BY entry_time DESC"

	rows, err := DB.Query(query, args...)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	reservations := []db.Reservation{}
	for rows.Next() {
		var res db.Reservation
		err := rows.Scan(
			&res.ID, &res.ReservationCode, &res.EntryTime, &res.ExitTime, &res.VehicleType, &res.FullName, &res.Email, &res.Phone, &res.LicensePlate, &res.VehicleModel, &res.Status, &res.CreatedAt, &res.UpdatedAt,
		)
		if err == nil {
			reservations = append(reservations, res)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reservations)
}

// PUT /admin/reservations/{id}
func AdminUpdateReservation(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var req struct {
		EntryTime    string `json:"entry_time"`
		ExitTime     string `json:"exit_time"`
		VehicleType  string `json:"vehicle_type"`
		FullName     string `json:"full_name"`
		Email        string `json:"email"`
		Phone        string `json:"phone"`
		LicensePlate string `json:"license_plate"`
		VehicleModel string `json:"vehicle_model"`
		Status       string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	// Validate input
	entry, err1 := time.Parse(time.RFC3339, req.EntryTime)
	exit, err2 := time.Parse(time.RFC3339, req.ExitTime)
	if err1 != nil || err2 != nil || !exit.After(entry) {
		http.Error(w, "Invalid entry/exit times", http.StatusBadRequest)
		return
	}
	if req.VehicleType == "" || req.FullName == "" || req.Email == "" || req.Phone == "" || req.LicensePlate == "" || req.VehicleModel == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}
	// Only allow certain status
	validStatus := map[string]bool{"active": true, "cancelled": true}
	if !validStatus[strings.ToLower(req.Status)] {
		http.Error(w, "Invalid status", http.StatusBadRequest)
		return
	}
	_, err := DB.Exec(`
        UPDATE reservations SET entry_time=$1, exit_time=$2, vehicle_type=$3, full_name=$4, email=$5, phone=$6, license_plate=$7, vehicle_model=$8, status=$9, updated_at=NOW()
        WHERE id=$10
    `, entry, exit, req.VehicleType, req.FullName, req.Email, req.Phone, req.LicensePlate, req.VehicleModel, req.Status, id)
	if err != nil {
		http.Error(w, "Could not update reservation", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Reservation updated"})
}

// DELETE /admin/reservations/{id}
func AdminDeleteReservation(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	_, err := DB.Exec(`DELETE FROM reservations WHERE id=$1`, id)
	if err != nil {
		http.Error(w, "Could not delete reservation", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Reservation deleted"})
}

// PUT /admin/spaces/{vehicle_type}
func UpdateVehicleSpaces(w http.ResponseWriter, r *http.Request) {
	vehicleType := mux.Vars(r)["vehicle_type"]
	var req struct {
		TotalSpaces     int `json:"total_spaces"`
		AvailableSpaces int `json:"available_spaces"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	if req.TotalSpaces < 0 || req.AvailableSpaces < 0 || req.AvailableSpaces > req.TotalSpaces {
		http.Error(w, "Invalid spaces values", http.StatusBadRequest)
		return
	}
	_, err := DB.Exec(`
        UPDATE vehicle_spaces SET total_spaces=$1, available_spaces=$2 WHERE vehicle_type=$3
    `, req.TotalSpaces, req.AvailableSpaces, vehicleType)
	if err != nil {
		http.Error(w, "Could not update vehicle spaces", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Vehicle spaces updated"})
}
