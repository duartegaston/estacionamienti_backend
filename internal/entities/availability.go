package entities

import "time"

type TimeSlotAvailability struct {
	StartTime       time.Time `json:"start_time"`
	EndTime         time.Time `json:"end_time"`
	IsAvailable     bool      `json:"is_available"`
	AvailableSpaces int       `json:"available_spaces"`
}

type AvailabilityResponse struct {
	IsOverallAvailable        bool                   `json:"is_overall_available"`
	RequestedStartTime        time.Time              `json:"requested_start_time"`
	RequestedEndTime          time.Time              `json:"requested_end_time"`
	Message                   string                 `json:"message,omitempty"`
	SlotDetails               []TimeSlotAvailability `json:"slot_details,omitempty"`
	FirstUnavailableSlotStart *time.Time             `json:"first_unavailable_slot_start,omitempty"`
}
