package repository

import (
	"database/sql"
	"estacionamienti/internal/db"
	"estacionamienti/internal/entities"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/lib/pq"
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
		if err == sql.ErrNoRows {
			return []SlotOccupationInfo{}, fmt.Errorf("vehicle type %d not configured in vehicle_spaces", vehicleTypeID)
		}
		return nil, fmt.Errorf("error checking vehicle space configuration: %w", err)
	}
	if !configuredSpaces.Valid || configuredSpaces.Int64 == 0 {
		// Tipo configurado pero con 0 espacios, o no configurado.
		// La query principal manejará esto con COALESCE, pero es bueno saberlo.
	}

	return results, nil
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

	err := r.DB.QueryRow(query, req.StartTime, req.EndTime, req.VehicleTypeID).Scan(&available)
	if err != nil {
		return 1, fmt.Errorf("Error check availability repository", err)
	}
	return available, err
}

func (r *ReservationRepository) CreateReservation(res *db.Reservation) error {
	query := `
		INSERT INTO reservations
		(code, user_name, user_email, user_phone, vehicle_type_id, vehicle_plate, vehicle_model, payment_method_id, status, start_time, end_time, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())
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
	).Scan(&res.ID, &res.CreatedAt, &res.UpdatedAt)
}

func (r *ReservationRepository) GetVehicleTypeIDByName(name string, id *int) error {
	return r.DB.QueryRow(`SELECT id FROM vehicle_types WHERE name = $1`, name).Scan(id)
}

func (r *ReservationRepository) GetReservationByCode(code, email string) (*entities.ReservationResponse, error) {
	var res entities.ReservationResponse

	query := `
        SELECT
            r.id, r.code, r.user_name, r.user_email, r.user_phone,
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
		&res.ID, &res.Code, &res.UserName, &res.UserEmail, &res.UserPhone,
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

// JOBS

// GetActiveReservationIDsPastEndTime busca IDs de reservas activas cuya fecha de fin ya pasó.
func (r *ReservationRepository) GetActiveReservationIDsPastEndTime() ([]int, error) {
	query := `SELECT id FROM reservations WHERE status = 'active' AND end_time < NOW()`
	rows, err := r.DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("error querying active reservations past end time: %w", err)
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("error scanning reservation ID: %w", err)
		}
		ids = append(ids, id)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error after iterating rows: %w", err)
	}
	return ids, nil
}

// UpdateReservationStatusesToFinished actualiza el estado de una lista de reservas a 'Finalizada'.
// También actualiza el campo updated_at.
func (r *ReservationRepository) UpdateReservationStatuses(ids []int, newStatus string) error {
	if len(ids) == 0 {
		return nil // No hay nada que actualizar
	}
	query := `UPDATE reservations SET status = $1, updated_at = NOW() WHERE id = ANY($2)`
	result, err := r.DB.Exec(query, newStatus, pq.Array(ids))
	if err != nil {
		return fmt.Errorf("error updating reservation statuses: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Could not get rows affected: %v", err)
	} else {
		log.Printf("Updated status for %d reservations to '%s'", rowsAffected, newStatus)
	}
	return nil
}

func (r *ReservationRepository) UpdateReservationStripeInfo(reservationID int, stripeCustomerID, stripePaymentIntentID,
	stripeSetupIntentID, stripeUsedPMID, newStatus, newPaymentStatus string) error {
	query := `
		UPDATE reservations
		SET
			stripe_customer_id = $2,
			stripe_payment_intent_id = $3,
			stripe_setup_intent_id = $4,
			stripe_payment_method_id = $5,
			status = $6,
			payment_status = $7,
			updated_at = $8
		WHERE id = $1`

	_, err := r.DB.Exec(query,
		reservationID,
		stripeCustomerID,
		stripePaymentIntentID,
		stripeSetupIntentID,
		stripeUsedPMID,
		newStatus,
		newPaymentStatus,
		time.Now(),
	)
	if err != nil {
		return fmt.Errorf("error actualizando reserva %d con info de Stripe: %w", reservationID, err)
	}
	return nil
}
