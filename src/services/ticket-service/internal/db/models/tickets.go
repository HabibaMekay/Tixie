package models

import "time"

type Ticket struct {
	ID        int       `db:"id" json:"id"`                 // Maps to id (SERIAL PK)
	EventID   int       `db:"event_id" json:"event_id"`     // Maps to event_id (INT FK)
	Price     float64   `db:"price" json:"price"`           // Maps to price (DECIMAL(10,2))
	Status    string    `db:"status" json:"status"`         // Maps to status (VARCHAR(20))
	CreatedAt time.Time `db:"created_at" json:"created_at"` // Maps to created_at (TIMESTAMP)
}
