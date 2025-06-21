package repository

import (
	"database/sql"
	"fmt"
	"github.com/lib/pq"
	"log"
	"time"
)

type JobRepository struct {
	DB *sql.DB
}

func NewJobRepository(db *sql.DB) *JobRepository {
	return &JobRepository{DB: db}
}

// GetActiveReservationIDsPastEndTime busca IDs de reservas activas cuya fecha de fin ya pasó.
func (r *JobRepository) GetActiveReservationIDsPastEndTime() ([]int, error) {
	now := time.Now().UTC()
	query := `SELECT id FROM reservations WHERE status = 'active' AND end_time < $1`
	rows, err := r.DB.Query(query, now)
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
func (r *JobRepository) UpdateReservationStatuses(ids []int, newStatus string) error {
	if len(ids) == 0 {
		return nil // No hay nada que actualizar
	}
	now := time.Now().UTC()
	query := `UPDATE reservations SET status = $1, updated_at = $2 WHERE id = ANY($3)`
	result, err := r.DB.Exec(query, newStatus, now, pq.Array(ids))
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
