package models

type Ticket struct {
	TicketID   int    `json:"ticket_id" db:"ticket_id"`
	EventID    int    `json:"event_id" db:"event_id"`
	UserID     int    `json:"user_id" db:"user_id"`
	TicketCode string `json:"ticket_code" db:"ticket_code"`
	Status     string `json:"status" db:"status"`
}
