package repository

import (
	"database/sql"
	"errors"
	"estacionamienti/internal/db"
	"estacionamienti/internal/entities"
	"estacionamienti/internal/utils"
	"fmt"
	"github.com/lib/pq"
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

func (r *ReservationRepository) GetHourlyAvailabilityDetails(startTime, endTime time.Time, vehicleTypeID int, vehicleTypeName string) ([]SlotOccupationInfo, error) {
	if !endTime.After(startTime) {
		return nil, fmt.Errorf("end time must be after start time")
	}

	vehicleTypes, err := r.GetVehicleTypes()
	if err != nil {
		return nil, fmt.Errorf("could not fetch vehicle types: %w", err)
	}
	// Compose slice of IDs sharing the pool
	var vtList []struct {
		ID   int
		Name string
	}
	for _, vt := range vehicleTypes {
		vtList = append(vtList, struct {
			ID   int
			Name string
		}{vt.ID, vt.Name})
	}
	idsForPool := utils.VehicleTypeIDsForSpace(vtList, vehicleTypeName)
	if len(idsForPool) == 0 {
		return nil, fmt.Errorf("no vehicle type ids found for pool")
	}

	query := `
		WITH requested_slots AS (
			SELECT
				gs.slot_hour_start,
				gs.slot_hour_start + interval '1 hour' AS slot_hour_end
			FROM generate_series(
				$1::timestamptz, -- startTime
				$2::timestamptz - interval '1 hour', -- endTime
				interval '1 hour'
			) AS gs(slot_hour_start)
		),
		total_spaces_for_type AS (
		  SELECT COALESCE(spaces, 0) AS spaces
		  FROM vehicle_spaces
		  WHERE vehicle_type_id = $3
		)
		SELECT
			rs.slot_hour_start,
			rs.slot_hour_end,
			COALESCE((SELECT spaces FROM total_spaces_for_type), 0) AS total_spaces,
			COUNT(r.id) AS booked_spaces
		FROM requested_slots rs
		LEFT JOIN reservations r
			ON r.vehicle_type_id = ANY($4)
			AND r.status = 'active'
			AND r.start_time < rs.slot_hour_end
			AND r.end_time > rs.slot_hour_start
		GROUP BY rs.slot_hour_start, rs.slot_hour_end
		ORDER BY rs.slot_hour_start;
    `

	// $3 is the mapped vehicle_type_id for vehicle_spaces, $4 is the array of ids for reservations
	mappedVehicleTypeID := utils.MapVehicleTypeIDForSpace(vehicleTypeID, vehicleTypeName)
	rows, err := r.DB.Query(query, startTime, endTime, mappedVehicleTypeID, pq.Array(idsForPool))
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
	err = r.DB.QueryRow("SELECT spaces FROM vehicle_spaces WHERE vehicle_type_id = $1", mappedVehicleTypeID).Scan(&configuredSpaces)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []SlotOccupationInfo{}, fmt.Errorf("vehicle type %d not configured in vehicle_spaces", mappedVehicleTypeID)
		}
		return nil, fmt.Errorf("error checking vehicle space configuration: %w", err)
	}

	return results, nil
}

func (r *ReservationRepository) GetPriceForUnit(vehicleTypeID int, reservationTimeID int) (float32, error) {
	var price float32
	err := r.DB.QueryRow(`SELECT price FROM vehicle_prices WHERE vehicle_type_id = $1 AND reservation_time_id = $2`, vehicleTypeID, reservationTimeID).Scan(&price)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, fmt.Errorf("no price configured for vehicle_type_id %d and reservation_time_id %d", vehicleTypeID, reservationTimeID)
		}
		return 0, err
	}
	return price, nil
}

func (r *ReservationRepository) CreateReservation(res *db.Reservation) error {
	query := `
		INSERT INTO reservations
		(code, user_name, user_email, user_phone, vehicle_type_id, vehicle_plate, vehicle_model, payment_method_id, status, start_time, end_time, created_at, updated_at, stripe_session_id, payment_status, language, total_price)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
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
		res.StripeSessionID,
		res.PaymentStatus,
		res.Language,
		res.TotalPrice,
	).Scan(&res.ID, &res.CreatedAt, &res.UpdatedAt)
}

func (r *ReservationRepository) GetReservationByCode(code, email string) (*entities.ReservationResponse, error) {
	var res entities.ReservationResponse

	query := `
        SELECT
            r.code, r.user_name, r.user_email, r.user_phone,
            r.vehicle_type_id, vt.name AS vehicle_type_name,
            r.vehicle_plate, r.vehicle_model,
            r.payment_method_id, pm.name AS payment_method_name, r.stripe_session_id, r.payment_status,
            r.status, r.start_time, r.end_time, r.created_at, r.updated_at, r.language, r.total_price
        FROM reservations r
        JOIN vehicle_types vt ON r.vehicle_type_id = vt.id
        JOIN payment_method pm ON r.payment_method_id = pm.id
        WHERE r.code = $1 AND r.user_email = $2
    `

	var totalPrice sql.NullFloat64
	err := r.DB.QueryRow(query, code, email).Scan(
		&res.Code, &res.UserName, &res.UserEmail, &res.UserPhone,
		&res.VehicleTypeID, &res.VehicleTypeName,
		&res.VehiclePlate, &res.VehicleModel,
		&res.PaymentMethodID, &res.PaymentMethodName, &res.StripeSessionID, &res.PaymentStatus,
		&res.Status, &res.StartTime, &res.EndTime, &res.CreatedAt, &res.UpdatedAt, &res.Language, &totalPrice,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("reservation with code '%s' and email '%s' not found: %w", code, email, err)
		}
		return nil, fmt.Errorf("error querying or scanning reservation: %w", err)
	}
	if totalPrice.Valid {
		res.TotalPrice = float32(totalPrice.Float64)
	} else {
		res.TotalPrice = 0
	}
	return &res, nil
}

func (r *ReservationRepository) CancelReservation(code string) (string, error) {
	timeUpdated := time.Now().UTC()
	query := `
		UPDATE reservations 
		SET status = 'canceled', updated_at = $2 
		WHERE code = $1 
		RETURNING status`
	var status string
	err := r.DB.QueryRow(query, code, timeUpdated).Scan(&status)
	if err != nil {
		return "", err
	}
	return status, nil
}

func (r *ReservationRepository) GetReservationByCodeOnly(code string) (*db.Reservation, error) {
	var res db.Reservation
	query := `
		SELECT id, code, user_name, user_email, user_phone, vehicle_type_id, vehicle_plate, vehicle_model, payment_method_id, status, start_time, end_time, created_at, updated_at, stripe_session_id, payment_status, language, total_price
		FROM reservations WHERE code = $1`
	var totalPrice sql.NullFloat64
	err := r.DB.QueryRow(query, code).Scan(
		&res.ID, &res.Code, &res.UserName, &res.UserEmail, &res.UserPhone, &res.VehicleTypeID, &res.VehiclePlate, &res.VehicleModel, &res.PaymentMethodID, &res.Status, &res.StartTime, &res.EndTime, &res.CreatedAt, &res.UpdatedAt,
		&res.StripeSessionID, &res.PaymentStatus, &res.Language, &totalPrice,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("reservation with code '%s' not found: %w", code, err)
		}
		return nil, fmt.Errorf("error querying reservation: %w", err)
	}
	if totalPrice.Valid {
		res.TotalPrice = totalPrice
	} else {
		res.TotalPrice = sql.NullFloat64{Float64: 0, Valid: false}
	}
	return &res, nil
}

func (r *ReservationRepository) GetReservationByStripeSessionID(sessionID string) (*db.Reservation, error) {
	var res db.Reservation
	var paymentIntentID sql.NullString
	var totalPrice sql.NullFloat64
	query := `
		SELECT id, code, user_name, user_email, user_phone, vehicle_type_id, vehicle_plate, vehicle_model, payment_method_id, status, start_time, end_time, created_at, 
		       updated_at, stripe_session_id, payment_status, language, stripe_payment_intent_id, total_price
		FROM reservations WHERE stripe_session_id = $1`
	err := r.DB.QueryRow(query, sessionID).Scan(
		&res.ID, &res.Code, &res.UserName, &res.UserEmail, &res.UserPhone, &res.VehicleTypeID, &res.VehiclePlate, &res.VehicleModel, &res.PaymentMethodID, &res.Status, &res.StartTime, &res.EndTime, &res.CreatedAt,
		&res.UpdatedAt, &res.StripeSessionID, &res.PaymentStatus, &res.Language, &paymentIntentID, &totalPrice)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("reservation with sessionID '%s' not found: %w", sessionID, err)
		}
		return nil, fmt.Errorf("error querying reservation: %w", err)
	}
	if paymentIntentID.Valid {
		res.StripePaymentIntentID = paymentIntentID
	} else {
		res.StripePaymentIntentID = sql.NullString{String: "", Valid: false}
	}
	if totalPrice.Valid {
		res.TotalPrice = totalPrice
	} else {
		res.TotalPrice = sql.NullFloat64{Float64: 0, Valid: false}
	}
	return &res, nil
}

func (r *ReservationRepository) UpdateReservationAndPaymentStatus(reservationID int, reservationStatus, paymentStatus string) error {
	query := `
		UPDATE reservations
		SET payment_status = $1, status = $2, updated_at = NOW()
		WHERE id = $3`
	_, err := r.DB.Exec(query, paymentStatus, reservationStatus, reservationID)
	return err
}

func (r *ReservationRepository) UpdateReservationStatusPaymentAndIntent(reservationID int, reservationStatus, paymentStatus, paymentIntentID string) error {
	query := `
		UPDATE reservations
		SET payment_status = $1, status = $2, stripe_payment_intent_id = $3, updated_at = NOW()
		WHERE id = $4`
	_, err := r.DB.Exec(query, paymentStatus, reservationStatus, paymentIntentID, reservationID)
	return err
}
