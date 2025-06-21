package repository

import (
	"database/sql"
	"estacionamienti/internal/db"
	"strconv"
)

type AdminRepository struct {
	DB *sql.DB
}

func NewAdminRepository(db *sql.DB) *AdminRepository {
	return &AdminRepository{DB: db}
}

func (r *AdminRepository) ListReservations(date, vehicleType string) ([]db.Reservation, error) {
	query := `
	SELECT
		r.id, r.code, r.user_name, r.user_email, r.user_phone, r.vehicle_type_id, r.vehicle_plate, r.vehicle_model,
		r.status, r.start_time, r.end_time, r.created_at, r.updated_at
	FROM reservations r
	JOIN vehicle_types vt ON vt.id = r.vehicle_type_id
	WHERE 1=1`
	args := []interface{}{}
	idx := 1

	if date != "" {
		query += " AND DATE(r.start_time) = $" + strconv.Itoa(idx)
		args = append(args, date)
		idx++
	}
	if vehicleType != "" {
		query += " AND vt.name = $" + strconv.Itoa(idx)
		args = append(args, vehicleType)
		idx++
	}
	query += " ORDER BY r.start_time DESC"

	rows, err := r.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reservations []db.Reservation
	for rows.Next() {
		var res db.Reservation
		err := rows.Scan(
			&res.ID, &res.Code, &res.UserName, &res.UserEmail, &res.UserPhone, &res.VehicleTypeID, &res.VehiclePlate, &res.VehicleModel,
			&res.Status, &res.StartTime, &res.EndTime, &res.CreatedAt, &res.UpdatedAt,
		)
		if err == nil {
			reservations = append(reservations, res)
		}
	}
	return reservations, nil
}

func (r *AdminRepository) ListVehicleSpaces() ([]db.VehicleSpace, error) {
	rows, err := r.DB.Query(`
		SELECT vt.name, vs.total_spaces, vs.available_spaces
		FROM vehicle_spaces vs
		JOIN vehicle_types vt ON vs.vehicle_type_id = vt.id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var spaces []db.VehicleSpace
	for rows.Next() {
		var vs db.VehicleSpace
		err := rows.Scan(&vs.VehicleType, &vs.TotalSpaces, &vs.AvailableSpaces)
		if err != nil {
			continue
		}
		spaces = append(spaces, vs)
	}
	return spaces, nil
}

func (r *AdminRepository) UpdateVehicleSpaces(vehicleType string, totalSpaces, availableSpaces int) error {
	_, err := r.DB.Exec(`
		UPDATE vehicle_spaces vs
		SET total_spaces = $1,
			available_spaces = $2
		FROM vehicle_types vt
		WHERE vs.vehicle_type_id = vt.id AND vt.name = $3
	`, totalSpaces, availableSpaces, vehicleType)
	return err
}
