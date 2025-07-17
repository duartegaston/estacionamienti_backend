package repository

import (
	"database/sql"
	"errors"
	"estacionamienti/internal/db"
	"estacionamienti/internal/entities"
	"fmt"
	"strconv"
)

type AdminRepository struct {
	DB *sql.DB
}

func NewAdminRepository(db *sql.DB) *AdminRepository {
	return &AdminRepository{DB: db}
}

func (r *AdminRepository) ListReservationsWithFilters(startTime, endTime, vehicleType, status, limit, offset string) ([]entities.ReservationResponse, error) {
	query := `
	SELECT
		r.code, r.user_name, r.user_email, r.user_phone, r.vehicle_type_id, vt.name AS vehicle_type_name,
		r.vehicle_plate, r.vehicle_model, r.payment_method_id, pm.name AS payment_method_name, r.stripe_session_id, r.payment_status,
		r.status, r.language, r.start_time, r.end_time, r.created_at, r.updated_at
	FROM reservations r
	JOIN vehicle_types vt ON vt.id = r.vehicle_type_id
	JOIN payment_method pm ON pm.id = r.payment_method_id
	WHERE 1=1`
	args := []interface{}{}
	idx := 1

	if startTime != "" {
		query += " AND r.start_time >= $" + strconv.Itoa(idx)
		args = append(args, startTime)
		idx++
	}
	if endTime != "" {
		query += " AND r.end_time <= $" + strconv.Itoa(idx)
		args = append(args, endTime)
		idx++
	}
	if vehicleType != "" {
		query += " AND vt.name = $" + strconv.Itoa(idx)
		args = append(args, vehicleType)
		idx++
	}
	if status != "" {
		query += " AND r.status = $" + strconv.Itoa(idx)
		args = append(args, status)
		idx++
	}
	query += " ORDER BY r.created_at DESC"
	if limit != "" {
		query += " LIMIT " + limit
	}
	if offset != "" {
		query += " OFFSET " + offset
	}

	rows, err := r.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reservations []entities.ReservationResponse
	for rows.Next() {
		var res entities.ReservationResponse
		err := rows.Scan(
			&res.Code, &res.UserName, &res.UserEmail, &res.UserPhone, &res.VehicleTypeID, &res.VehicleTypeName,
			&res.VehiclePlate, &res.VehicleModel, &res.PaymentMethodID, &res.PaymentMethodName, &res.StripeSessionID, &res.PaymentStatus,
			&res.Status, &res.Language, &res.StartTime, &res.EndTime, &res.CreatedAt, &res.UpdatedAt,
		)
		if err == nil {
			reservations = append(reservations, res)
		}
	}
	return reservations, nil
}

// FindReservationByCode returns a reservation by code and maps it to entities.ReservationResponse
func (r *AdminRepository) FindReservationByCode(code string) (*entities.ReservationResponse, error) {
	var res entities.ReservationResponse

	query := `
        SELECT
            r.code, r.user_name, r.user_email, r.user_phone,
            r.vehicle_type_id, vt.name AS vehicle_type_name,
            r.vehicle_plate, r.vehicle_model,
            r.payment_method_id, pm.name AS payment_method_name,
            r.status, r.start_time, r.end_time, r.created_at, r.updated_at, r.language
        FROM reservations r
        JOIN vehicle_types vt ON vt.id = r.vehicle_type_id
        JOIN payment_method pm ON pm.id = r.payment_method_id
        WHERE r.code = $1`

	err := r.DB.QueryRow(query, code).Scan(
		&res.Code, &res.UserName, &res.UserEmail, &res.UserPhone,
		&res.VehicleTypeID, &res.VehicleTypeName,
		&res.VehiclePlate, &res.VehicleModel,
		&res.PaymentMethodID, &res.PaymentMethodName,
		&res.Status, &res.StartTime, &res.EndTime, &res.CreatedAt, &res.UpdatedAt, &res.Language,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("reservation with code '%s' not found: %w", code, err)
		}
		return nil, fmt.Errorf("error querying or scanning reservation: %w", err)
	}
	return &res, nil
}

// --

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
