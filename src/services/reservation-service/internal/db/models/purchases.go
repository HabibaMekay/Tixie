package models

import "time"

type Purchase struct {
	ID           int       `db:"id"`
	TicketID     int       `db:"ticket_id"`
	UserID       int       `db:"user_id"`
	EventID      int       `db:"event_id"`
	PurchaseDate time.Time `db:"purchase_date"`
	Status       string    `db:"status"`
}
