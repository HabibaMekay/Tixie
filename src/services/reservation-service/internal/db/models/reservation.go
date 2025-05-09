package models

import "time"

// Reservation represents a temporary hold on a ticket
type Reservation struct {
	ID             int       `db:"id" json:"id"`
	EventID        int       `db:"event_id" json:"event_id"`
	UserID         int       `db:"user_id" json:"user_id"`
	Status         string    `db:"status" json:"status"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
	ExpirationTime time.Time `db:"expiration_time" json:"expiration_time"`
}
