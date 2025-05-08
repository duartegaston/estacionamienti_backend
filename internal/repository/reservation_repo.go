package repository

import (
	"database/sql"
	"estacionamienti/internal/db"
	"strconv"
)

type ReservationRepository struct {
	DB *sql.DB
}

func NewReservationRepository(db *sql.DB) *ReservationRepository {
	return &ReservationRepository{DB: db}
}

func (r *ReservationRepository) ListReservations(date, vehicleType string) ([]db.Reservation, error) {
	query := "SELECT id, reservation_code, entry_time, exit_time, vehicle_type, full_name, email, phone, license_plate, vehicle_model, status, created_at, updated_at FROM reservations WHERE 1=1"
	args := []interface{}{}
	idx := 1

	if date != "" {
		query += " AND DATE(entry_time) = $" + strconv.Itoa(idx)
		args = append(args, date)
		idx++
	}
	if vehicleType != "" {
		query += " AND vehicle_type = $" + strconv.Itoa(idx)
		args = append(args, vehicleType)
		idx++
	}
	query += " ORDER BY entry_time DESC"

	rows, err := r.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reservations []db.Reservation
	for rows.Next() {
		var res db.Reservation
		err := rows.Scan(
			&res.ID, &res.ReservationCode, &res.EntryTime, &res.ExitTime, &res.VehicleType, &res.FullName, &res.Email, &res.Phone, &res.LicensePlate, &res.VehicleModel, &res.Status, &res.CreatedAt, &res.UpdatedAt,
		)
		if err == nil {
			reservations = append(reservations, res)
		}
	}
	return reservations, nil
}

func (r *ReservationRepository) GetReservationByCode(code string) (*db.Reservation, error) {
	var res db.Reservation
	err := r.DB.QueryRow(`SELECT id, reservation_code, entry_time, exit_time, vehicle_type, full_name, email, phone, license_plate, vehicle_model, status, created_at, updated_at FROM reservations WHERE reservation_code = $1`, code).
		Scan(&res.ID, &res.ReservationCode, &res.EntryTime, &res.ExitTime, &res.VehicleType, &res.FullName, &res.Email, &res.Phone, &res.LicensePlate, &res.VehicleModel, &res.Status, &res.CreatedAt, &res.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *ReservationRepository) CreateReservation(res *db.Reservation) error {
	_, err := r.DB.Exec(`
        INSERT INTO reservations
        (reservation_code, entry_time, exit_time, vehicle_type, full_name, email, phone, license_plate, vehicle_model, status)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,'active')
    `,
		res.ReservationCode, res.EntryTime, res.ExitTime, res.VehicleType, res.FullName, res.Email, res.Phone, res.LicensePlate, res.VehicleModel,
	)
	return err
}

func (r *ReservationRepository) UpdateReservationByCode(code string, updates map[string]interface{}) error {
	// Build query dynamically or use a fixed set of fields for update
	// For brevity, only user info fields updated here
	_, err := r.DB.Exec(`
        UPDATE reservations SET full_name=$1, email=$2, phone=$3, license_plate=$4, vehicle_model=$5, updated_at=NOW()
        WHERE reservation_code=$6 AND status='active'
    `, updates["full_name"], updates["email"], updates["phone"], updates["license_plate"], updates["vehicle_model"], code)
	return err
}

func (r *ReservationRepository) CancelReservation(code string) (string, error) {
	tx, err := r.DB.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	var vehicleType string
	err = tx.QueryRow(`SELECT vehicle_type FROM reservations WHERE reservation_code=$1 AND status='active'`, code).Scan(&vehicleType)
	if err != nil {
		return "", err
	}
	_, err = tx.Exec(`UPDATE reservations SET status='cancelled', updated_at=NOW() WHERE reservation_code=$1`, code)
	if err != nil {
		return "", err
	}
	_, err = tx.Exec(`UPDATE vehicle_spaces SET available_spaces=available_spaces+1 WHERE vehicle_type=$1`, vehicleType)
	if err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return vehicleType, nil
}

func (r *ReservationRepository) UpdateReservationByID(id int, res *db.Reservation) error {
	_, err := r.DB.Exec(`
        UPDATE reservations SET entry_time=$1, exit_time=$2, vehicle_type=$3, full_name=$4, email=$5, phone=$6, license_plate=$7, vehicle_model=$8, status=$9, updated_at=NOW()
        WHERE id=$10
    `, res.EntryTime, res.ExitTime, res.VehicleType, res.FullName, res.Email, res.Phone, res.LicensePlate, res.VehicleModel, res.Status, id)
	return err
}

func (r *ReservationRepository) DeleteReservationByID(id int) error {
	_, err := r.DB.Exec(`DELETE FROM reservations WHERE id=$1`, id)
	return err
}

func (r *ReservationRepository) UpdateVehicleSpaces(vehicleType string, totalSpaces, availableSpaces int) error {
	_, err := r.DB.Exec(`UPDATE vehicle_spaces SET total_spaces=$1, available_spaces=$2 WHERE vehicle_type=$3`, totalSpaces, availableSpaces, vehicleType)
	return err
}

func (r *ReservationRepository) CheckAvailability(vehicleType string) (int, error) {
	var available int
	err := r.DB.QueryRow(`SELECT available_spaces FROM vehicle_spaces WHERE vehicle_type = $1`, vehicleType).Scan(&available)
	return available, err
}

func (r *ReservationRepository) ListVehicleSpaces() ([]db.VehicleSpace, error) {
	rows, err := r.DB.Query(`SELECT id, vehicle_type, total_spaces, available_spaces FROM vehicle_spaces ORDER BY vehicle_type`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var spaces []db.VehicleSpace
	for rows.Next() {
		var vs db.VehicleSpace
		if err := rows.Scan(&vs.ID, &vs.VehicleType, &vs.TotalSpaces, &vs.AvailableSpaces); err == nil {
			spaces = append(spaces, vs)
		}
	}
	return spaces, nil
}
