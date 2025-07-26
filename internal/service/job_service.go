package service

import (
	"estacionamienti/internal/repository"
	"fmt"
	"log"
	"time"
)

type JobService struct {
	Repo *repository.JobRepository
}

func NewJobService(repo *repository.JobRepository) *JobService {
	return &JobService{Repo: repo}
}

// UpdateFinishedReservations busca reservas activas que han finalizado y actualiza su estado a "finished".
func (s *JobService) UpdateFinishedReservations() error {
	log.Println("Cron Job: Checking for reservations to mark as 'finished'...")

	reservationIDs, err := s.Repo.GetActiveReservationIDsPastEndTime()
	if err != nil {
		return fmt.Errorf("cron job: failed to get active reservations past end time: %w", err)
	}

	if len(reservationIDs) == 0 {
		log.Println("Cron Job: No active reservations found past their end time.")
		return nil
	}

	log.Printf("Cron Job: Found %d reservations to mark as 'finished'. IDs: %v", len(reservationIDs), reservationIDs)

	err = s.Repo.UpdateReservationStatuses(reservationIDs, "finished")
	if err != nil {
		return fmt.Errorf("cron job: failed to update reservation statuses: %w", err)
	}

	log.Printf("Cron Job: Successfully updated %d reservations to 'finished'.", len(reservationIDs))
	return nil
}

// DeleteOldPendingReservations deletes all reservations with status 'pending' created before the given time.
func (s *JobService) DeleteOldPendingReservations(before time.Time) (int64, error) {
	return s.Repo.DeletePendingReservationsOlderThan(before)
}
