package repository

import (
	"database/sql"
	"errors"
	"estacionamienti/internal/db"
	"estacionamienti/internal/entities"
	"fmt"
	"log"
	"time"
)

type SlotOccupationInfo struct {
	SlotStart    time.Time
	SlotEnd      time.Time
	TotalSpaces  int
	BookedSpaces int
}

type ReservationRepository struct {
	DB *sql.DB
}

func NewReservationRepository(db *sql.DB) *ReservationRepository {
	return &ReservationRepository{DB: db}
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

func (r *ReservationRepository) GetHourlyAvailabilityDetails(startTime, endTime time.Time, vehicleTypeID int) ([]SlotOccupationInfo, error) {
	if !endTime.After(startTime) {
		return nil, fmt.Errorf("end time must be after start time")
	}

	query := `
		WITH requested_slots AS (
			SELECT
				gs.slot_hour_start,
				gs.slot_hour_start + interval '1 hour' AS slot_hour_end
			FROM generate_series(
				$1::timestamptz, -- startTime
				$2::timestamptz - interval '1 hour', -- endTime (para generar slots HASTA justo antes de endTime)
				interval '1 hour'
			) AS gs(slot_hour_start)
		),
		total_spaces_for_type AS (
		  SELECT COALESCE(spaces, 0) AS spaces -- COALESCE para manejar si no hay entrada para el tipo
		  FROM vehicle_spaces
		  WHERE vehicle_type_id = $3 -- vehicleTypeID
		)
		SELECT
			rs.slot_hour_start,
			rs.slot_hour_end,
			COALESCE((SELECT spaces FROM total_spaces_for_type), 0) AS total_spaces,
			COUNT(r.id) AS booked_spaces
		FROM requested_slots rs
		LEFT JOIN reservations r
			ON r.vehicle_type_id = $3
			AND r.status = 'active'
			AND r.start_time < rs.slot_hour_end
			AND r.end_time > rs.slot_hour_start
		GROUP BY rs.slot_hour_start, rs.slot_hour_end
		ORDER BY rs.slot_hour_start;
    `

	rows, err := r.DB.Query(query, startTime, endTime, vehicleTypeID)
	if err != nil {
		return nil, fmt.Errorf("error querying hourly availability: %w", err)
	}
	defer rows.Close()

	var results []SlotOccupationInfo
	var hasTotalSpacesEntry bool

	for rows.Next() {
		var soi SlotOccupationInfo
		err := rows.Scan(&soi.SlotStart, &soi.SlotEnd, &soi.TotalSpaces, &soi.BookedSpaces)
		if err != nil {
			return nil, fmt.Errorf("error scanning hourly availability slot: %w", err)
		}
		results = append(results, soi)
		if !hasTotalSpacesEntry && soi.TotalSpaces > 0 {
			hasTotalSpacesEntry = true
		}
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error after iterating hourly availability rows: %w", err)
	}

	// Una verificación más robusta para "tipo de vehículo no configurado":
	var configuredSpaces sql.NullInt64
	err = r.DB.QueryRow("SELECT spaces FROM vehicle_spaces WHERE vehicle_type_id = $1", vehicleTypeID).Scan(&configuredSpaces)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []SlotOccupationInfo{}, fmt.Errorf("vehicle type %d not configured in vehicle_spaces", vehicleTypeID)
		}
		return nil, fmt.Errorf("error checking vehicle space configuration: %w", err)
	}

	return results, nil
}

func (r *ReservationRepository) GetPriceForUnit(vehicleTypeID int, reservationTimeID int) (int, error) {
	var price int
	err := r.DB.QueryRow(`SELECT price FROM vehicle_prices WHERE vehicle_type_id = $1 AND reservation_time_id = $2`, vehicleTypeID, reservationTimeID).Scan(&price)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("no price configured for vehicle_type_id %d and reservation_time_id %d", vehicleTypeID, reservationTimeID)
		}
		return 0, err
	}
	return price, nil
}

func (r *ReservationRepository) CreateReservation(res *db.Reservation) error {
	query := `
		INSERT INTO reservations
		(code, user_name, user_email, user_phone, vehicle_type_id, vehicle_plate, vehicle_model, payment_method_id, status, start_time, end_time, created_at, updated_at, stripe_customer_id, stripe_payment_intent_id, stripe_setup_intent_id, stripe_payment_method_id, payment_status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		RETURNING id, created_at, updated_at`
	return r.DB.QueryRow(query,
		res.Code,
		res.UserName,
		res.UserEmail,
		res.UserPhone,
		res.VehicleTypeID,
		res.VehiclePlate,
		res.VehicleModel,
		res.PaymentMethodID,
		res.Status,
		res.StartTime,
		res.EndTime,
		res.CreatedAt,
		res.UpdatedAt,
		res.StripeCustomerID,
		res.StripePaymentIntentID,
		res.StripeSetupIntentID,
		res.StripePaymentMethodID,
		res.PaymentStatus,
	).Scan(&res.ID, &res.CreatedAt, &res.UpdatedAt)
}

func (r *ReservationRepository) GetReservationByCode(code, email string) (*entities.ReservationResponse, error) {
	var res entities.ReservationResponse

	query := `
        SELECT
            r.code, r.user_name, r.user_email, r.user_phone,
            r.vehicle_type_id, vt.name AS vehicle_type_name,
            r.vehicle_plate, r.vehicle_model,
            r.payment_method_id, pm.name AS payment_method_name,
            r.status, r.start_time, r.end_time, r.created_at, r.updated_at
        FROM reservations r
        JOIN vehicle_types vt ON r.vehicle_type_id = vt.id
        JOIN payment_method pm ON r.payment_method_id = pm.id
        WHERE r.code = $1 AND r.user_email = $2
    `

	err := r.DB.QueryRow(query, code, email).Scan(
		&res.Code, &res.UserName, &res.UserEmail, &res.UserPhone,
		&res.VehicleTypeID, &res.VehicleTypeName,
		&res.VehiclePlate, &res.VehicleModel,
		&res.PaymentMethodID, &res.PaymentMethodName,
		&res.Status, &res.StartTime, &res.EndTime, &res.CreatedAt, &res.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("reservation with code '%s' and email '%s' not found: %w", code, email, err)
		}
		return nil, fmt.Errorf("error querying or scanning reservation: %w", err)
	}
	return &res, nil
}

func (r *ReservationRepository) CancelReservation(code string) (string, error) {
	query := `UPDATE reservations SET status = 'canceled', updated_at = time.Now().UTC() WHERE code = $1 RETURNING status`
	var status string
	err := r.DB.QueryRow(query, code).Scan(&status)
	if err != nil {
		log.Printf("Error canceling reservation: %v", err)
		return "", err
	}
	return status, nil
}

func (r *ReservationRepository) GetReservationByCodeOnly(code string) (*db.Reservation, error) {
	var res db.Reservation
	query := `
		SELECT id, code, user_name, user_email, user_phone, vehicle_type_id, vehicle_plate, vehicle_model, payment_method_id, status, start_time, end_time, created_at, updated_at, stripe_customer_id, stripe_payment_intent_id, stripe_setup_intent_id, stripe_payment_method_id, payment_status
		FROM reservations WHERE code = $1`
	err := r.DB.QueryRow(query, code).Scan(
		&res.ID, &res.Code, &res.UserName, &res.UserEmail, &res.UserPhone, &res.VehicleTypeID, &res.VehiclePlate, &res.VehicleModel, &res.PaymentMethodID, &res.Status, &res.StartTime, &res.EndTime, &res.CreatedAt, &res.UpdatedAt, &res.StripeCustomerID, &res.StripePaymentIntentID, &res.StripeSetupIntentID, &res.StripePaymentMethodID, &res.PaymentStatus,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("reservation with code '%s' not found: %w", code, err)
		}
		return nil, fmt.Errorf("error querying reservation: %w", err)
	}
	return &res, nil
}
