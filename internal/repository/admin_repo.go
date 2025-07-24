package repository

import (
	"database/sql"
	"errors"
	"estacionamienti/internal/db"
	"estacionamienti/internal/entities"
	"fmt"
	"log"
	"strconv"
)

type AdminRepository struct {
	DB *sql.DB
}

func NewAdminRepository(db *sql.DB) *AdminRepository {
	return &AdminRepository{DB: db}
}

func (r *AdminRepository) ListReservationsWithFilters(code, startTime, endTime, vehicleType, status, limit, offset string) ([]entities.ReservationResponse, error) {
	query := `
	SELECT
		r.code, r.user_name, r.user_email, r.user_phone, r.vehicle_type_id, vt.name AS vehicle_type_name,
		r.vehicle_plate, r.vehicle_model, r.payment_method_id, pm.name AS payment_method_name, COALESCE(r.payment_status, '') AS payment_status,
		r.status, r.start_time, r.end_time, r.created_at, r.updated_at, COALESCE(r.total_price, 0) AS total_price
	FROM reservations r
	JOIN vehicle_types vt ON vt.id = r.vehicle_type_id
	JOIN payment_method pm ON pm.id = r.payment_method_id
	WHERE 1=1`
	args := []interface{}{}
	idx := 1

	if startTime != "" {
		query += " AND DATE(r.start_time) >= $" + strconv.Itoa(idx)
		args = append(args, startTime)
		idx++
	}
	if endTime != "" {
		query += " AND DATE(r.end_time) <= $" + strconv.Itoa(idx)
		args = append(args, endTime)
		idx++
	}
	if code != "" {
		query += " AND r.code LIKE $" + strconv.Itoa(idx)
		args = append(args, "%"+code+"%")
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
			&res.VehiclePlate, &res.VehicleModel, &res.PaymentMethodID, &res.PaymentMethodName, &res.PaymentStatus,
			&res.Status, &res.StartTime, &res.EndTime, &res.CreatedAt, &res.UpdatedAt, &res.TotalPrice,
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
            r.status, r.start_time, r.end_time, r.created_at, r.updated_at, r.language, r.total_price
        FROM reservations r
        JOIN vehicle_types vt ON vt.id = r.vehicle_type_id
        JOIN payment_method pm ON pm.id = r.payment_method_id
        WHERE r.code = $1`

	log.Printf("SQL Query: %s | Args: %v", query, code)
	err := r.DB.QueryRow(query, code).Scan(
		&res.Code, &res.UserName, &res.UserEmail, &res.UserPhone,
		&res.VehicleTypeID, &res.VehicleTypeName,
		&res.VehiclePlate, &res.VehicleModel,
		&res.PaymentMethodID, &res.PaymentMethodName,
		&res.Status, &res.StartTime, &res.EndTime, &res.CreatedAt, &res.UpdatedAt, &res.Language, &res.TotalPrice,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("reservation with code '%s' not found: %w", code, err)
		}
		return nil, fmt.Errorf("error querying or scanning reservation: %w", err)
	}
	return &res, nil
}

func (r *AdminRepository) ListVehicleSpaces() ([]db.VehicleSpaceWithPrices, error) {
	query := `SELECT vt.id, vt.name, vs.spaces FROM vehicle_spaces vs JOIN vehicle_types vt ON vs.vehicle_type_id = vt.id`
	log.Printf("SQL Query: %s | Args: %v", query, nil)
	rows, err := r.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []db.VehicleSpaceWithPrices
	for rows.Next() {
		var vehicleTypeID int
		var vehicleTypeName string
		var spaces int
		err := rows.Scan(&vehicleTypeID, &vehicleTypeName, &spaces)
		if err != nil {
			continue
		}

		// Query prices for this vehicle type
		pricesQuery := `SELECT rt.name, vp.price FROM vehicle_prices vp JOIN reservation_times rt ON vp.reservation_time_id = rt.id WHERE vp.vehicle_type_id = $1`
		log.Printf("SQL Query: %s | Args: %v", pricesQuery, vehicleTypeID)
		priceRows, err := r.DB.Query(pricesQuery, vehicleTypeID)
		if err != nil {
			continue
		}
		prices := make(map[string]int)
		for priceRows.Next() {
			var reservationTime string
			var price int
			if err := priceRows.Scan(&reservationTime, &price); err == nil {
				prices[reservationTime] = price
			}
		}
		priceRows.Close()

		result = append(result, db.VehicleSpaceWithPrices{
			VehicleType: vehicleTypeName,
			Spaces:      spaces,
			Prices:      prices,
		})
	}
	return result, nil
}

func (r *AdminRepository) UpdateVehicleSpaces(vehicleType string, spaces int) error {
	query := `
		UPDATE vehicle_spaces vs
		SET spaces = $1
		FROM vehicle_types vt
		WHERE vs.vehicle_type_id = vt.id AND vt.name = $2
	`
	log.Printf("SQL Query: %s | Args: %v", query, []interface{}{spaces, vehicleType})
	_, err := r.DB.Exec(query, spaces, vehicleType)
	return err
}

func (r *AdminRepository) UpdateVehiclePrice(vehicleType string, timeName string, price int) error {
	// Get vehicle_type_id
	var vehicleTypeID int
	query := `SELECT id FROM vehicle_types WHERE name = $1`
	log.Printf("SQL Query: %s | Args: %v", query, vehicleType)
	err := r.DB.QueryRow(query, vehicleType).Scan(&vehicleTypeID)
	if err != nil {
		return err
	}
	// Get reservation_time_id
	var reservationTimeID int
	query = `SELECT id FROM reservation_times WHERE name = $1`
	log.Printf("SQL Query: %s | Args: %v", query, timeName)
	err = r.DB.QueryRow(query, timeName).Scan(&reservationTimeID)
	if err != nil {
		return err
	}
	// Upsert price
	query = `
		INSERT INTO vehicle_prices (vehicle_type_id, reservation_time_id, price)
		VALUES ($1, $2, $3)
		ON CONFLICT (vehicle_type_id, reservation_time_id)
		DO UPDATE SET price = EXCLUDED.price
	`
	log.Printf("SQL Query: %s | Args: %v", query, []interface{}{vehicleTypeID, reservationTimeID, price})
	_, err = r.DB.Exec(query, vehicleTypeID, reservationTimeID, price)
	return err
}
