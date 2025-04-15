package repos

import (
	"database/sql"
	"ticket-service/internal/db/models"

	"github.com/jmoiron/sqlx"
)

// TicketRepository handles database operations for tickets.
type TicketRepository struct {
	db *sqlx.DB
}

// NewTicketRepository creates a new TicketRepository.
func NewTicketRepository(db *sqlx.DB) *TicketRepository {
	return &TicketRepository{db: db}
}

// GetTicketByID retrieves a ticket by its ID.
func (r *TicketRepository) GetTicketByID(ticketID int) (*models.Ticket, error) {
	var ticket models.Ticket
	err := r.db.Get(&ticket, "SELECT * FROM ticket WHERE id=$1", ticketID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &ticket, nil
}

// GetTicketsByEventID retrieves tickets for a given event_id.
func (r *TicketRepository) GetTicketsByEventID(eventID int) ([]models.Ticket, error) {
	var tickets []models.Ticket
	err := r.db.Select(&tickets, "SELECT * FROM ticket WHERE event_id=$1 AND status='available'", eventID)
	if err != nil {
		return nil, err
	}
	return tickets, nil
}

// CreateTicket creates a new ticket.
func (r *TicketRepository) CreateTicket(ticket *models.Ticket) (*models.Ticket, error) {
	var createdTicket models.Ticket
	err := r.db.QueryRowx(
		"INSERT INTO ticket (event_id, price, status) VALUES ($1, $2, $3) RETURNING *",
		ticket.EventID, ticket.Price, ticket.Status,
	).StructScan(&createdTicket)
	if err != nil {
		return nil, err
	}
	return &createdTicket, nil
}

// UpdateTicketStatus updates the status of a ticket.
func (r *TicketRepository) UpdateTicketStatus(ticketID int, status string) (*models.Ticket, error) {
	var updatedTicket models.Ticket
	err := r.db.QueryRowx(
		"UPDATE ticket SET status=$1 WHERE id=$2 RETURNING *",
		status, ticketID,
	).StructScan(&updatedTicket)
	if err != nil {
		return nil, err
	}
	return &updatedTicket, nil
}
