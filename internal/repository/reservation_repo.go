package repository

import (
	"database/sql"
	"estacionamienti/internal/db"
	"estacionamienti/internal/entities"
	"fmt"
	"strconv"
)

type ReservationRepository struct {
	DB *sql.DB
}

func NewReservationRepository(db *sql.DB) *ReservationRepository {
	return &ReservationRepository{DB: db}
}

func (r *ReservationRepository) CheckAvailability(req entities.ReservationRequest) (int, error) {
	var available int
	query := `
		SELECT 
			vs.spaces - COUNT(r.id) AS available_spaces
		FROM vehicle_spaces vs
		JOIN vehicle_types vt ON vt.id = vs.vehicle_type_id
		LEFT JOIN reservations r 
			ON r.vehicle_type_id = vt.id
			AND r.status = 'active'
			AND r.start_time < $2 AND r.end_time > $1
		WHERE vt.name = $3
		GROUP BY vs.spaces
	`

	err := r.DB.QueryRow(query, req.StartTime, req.EndTime, req.VehicleType).Scan(&available)
	if err != nil {
		return 1, fmt.Errorf("Error check availability repository", err)
	}
	return available, err
}

func (r *ReservationRepository) CreateReservation(res *db.Reservation) error {
	query := `
		INSERT INTO reservations
		(code, user_name, user_email, user_phone, vehicle_type_id, vehicle_plate, vehicle_model, payment_method, status, start_time, end_time)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING id, created_at, updated_at`
	return r.DB.QueryRow(query,
		res.Code,
		res.UserName,
		res.UserEmail,
		res.UserPhone,
		res.VehicleTypeID,
		res.VehiclePlate,
		res.VehicleModel,
		res.PaymentMethod,
		res.Status,
		res.StartTime,
		res.EndTime,
	).Scan(&res.ID, &res.CreatedAt, &res.UpdatedAt)
}

func (r *ReservationRepository) GetVehicleTypeIDByName(name string, id *int) error {
	return r.DB.QueryRow(`SELECT id FROM vehicle_types WHERE name = $1`, name).Scan(id)
}

func (r *ReservationRepository) GetReservationByCode(code, email string) (*db.Reservation, error) {
	var res db.Reservation
	err := r.DB.QueryRow(`
		SELECT r.id, r.code, r.user_name, r.user_email, r.user_phone, r.vehicle_type_id, r.vehicle_plate, r.vehicle_model,
		       r.payment_method, r.status, r.start_time, r.end_time, r.created_at, r.updated_at
		FROM reservations r
		WHERE r.code = $1 AND r.user_email = $2
	`, code, email).Scan(
		&res.ID, &res.Code, &res.UserName, &res.UserEmail, &res.UserPhone, &res.VehicleTypeID, &res.VehiclePlate, &res.VehicleModel,
		&res.PaymentMethod, &res.Status, &res.StartTime, &res.EndTime, &res.CreatedAt, &res.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *ReservationRepository) CancelReservation(code string) (string, error) {
	query := `UPDATE reservations SET status = 'cancelled', updated_at = NOW() WHERE code = $1 RETURNING status`
	var status string
	err := r.DB.QueryRow(query, code).Scan(&status)
	if err != nil {
		return "", err
	}
	return status, nil
}

func (r *ReservationRepository) GetPrices() ([]entities.PriceResponse, error) {
	query := `
	SELECT vt.name as vehicle_type, rt.name as reservation_time, vp.price
	FROM vehicle_prices vp
	JOIN vehicle_types vt ON vp.vehicle_type_id = vt.id
	JOIN reservation_times rt ON vp.reservation_time_id = rt.id
	ORDER BY vt.name, rt.name
	`

	rows, err := r.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prices []entities.PriceResponse
	for rows.Next() {
		var p entities.PriceResponse
		if err := rows.Scan(&p.VehicleType, &p.ReservationTime, &p.Price); err != nil {
			return nil, err
		}
		prices = append(prices, p)
	}

	return prices, nil
}

func (r *ReservationRepository) GetVehicleTypes() ([]db.VehicleType, error) {
	query := `SELECT id, name FROM vehicle_types ORDER BY name`

	rows, err := r.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var types []db.VehicleType
	for rows.Next() {
		var vt db.VehicleType
		if err := rows.Scan(&vt.ID, &vt.Name); err != nil {
			return nil, err
		}
		types = append(types, vt)
	}

	return types, nil
}

// ADMIN FUNCTIONS

func (r *ReservationRepository) ListReservations(date, vehicleType string) ([]db.Reservation, error) {
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

func (r *ReservationRepository) UpdateVehicleSpaces(vehicleType string, totalSpaces, availableSpaces int) error {
	_, err := r.DB.Exec(`
		UPDATE vehicle_spaces vs
		SET total_spaces = $1,
			available_spaces = $2
		FROM vehicle_types vt
		WHERE vs.vehicle_type_id = vt.id AND vt.name = $3
	`, totalSpaces, availableSpaces, vehicleType)
	return err
}

func (r *ReservationRepository) ListVehicleSpaces() ([]db.VehicleSpace, error) {
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
