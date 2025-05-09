package repos

import (
	"database/sql"
	"event-service/internal/db/models"
	"fmt"

	circuitbreaker "tixie.local/common"
)

type EventRepository struct {
	DB      *sql.DB
	breaker *circuitbreaker.CircuitBreaker
}

func NewEventRepository(db *sql.DB) *EventRepository {
	return &EventRepository{
		DB:      db,
		breaker: circuitbreaker.NewCircuitBreaker(circuitbreaker.DefaultSettings("event-repository")),
	}
}

func (r *EventRepository) GetAllEvents() ([]models.Event, error) {
	var events []models.Event
	err := r.breaker.Execute(func() error {
		query := `SELECT id, name, date, venue, total_tickets, vendor_id, price, sold_tickets, tickets_left, tickets_reserved, reservation_timeout FROM events`
		rows, err := r.DB.Query(query)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var e models.Event
			if err := rows.Scan(&e.ID, &e.Name, &e.Date, &e.Venue, &e.TotalTickets, &e.VendorID, &e.Price, &e.SoldTickets, &e.TicketsLeft, &e.TicketsReserved, &e.ReservationTimeout); err != nil {
				return err
			}
			events = append(events, e)
		}
		return nil
	})
	return events, err
}

func (r *EventRepository) CreateEvent(event models.Event) error {
	return r.breaker.Execute(func() error {
		query := `
            INSERT INTO events (name, date, venue, total_tickets, vendor_id, price, tickets_reserved, reservation_timeout)
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
        `
		_, err := r.DB.Exec(query, event.Name, event.Date, event.Venue, event.TotalTickets, event.VendorID, event.Price, 0, event.ReservationTimeout)
		return err
	})
}

func (r *EventRepository) GetEventByID(id int) (models.Event, error) {
	var e models.Event
	err := r.breaker.Execute(func() error {
		query := `SELECT id, name, date, venue, total_tickets, vendor_id, price, sold_tickets, tickets_left, tickets_reserved, reservation_timeout 
				  FROM events WHERE id = $1`
		return r.DB.QueryRow(query, id).Scan(&e.ID, &e.Name, &e.Date, &e.Venue, &e.TotalTickets, &e.VendorID, &e.Price, &e.SoldTickets, &e.TicketsLeft, &e.TicketsReserved, &e.ReservationTimeout)
	})
	return e, err
}

func (r *EventRepository) UpdateTicketsSold(eventID string, ticketsToBuy int) error {
	return r.breaker.Execute(func() error {
		var event struct {
			TotalTickets    int `json:"total_tickets"`
			SoldTickets     int `json:"sold_tickets"`
			TicketsReserved int `json:"tickets_reserved"`
		}

		query := `SELECT total_tickets, sold_tickets, tickets_reserved FROM events WHERE id = $1`
		err := r.DB.QueryRow(query, eventID).Scan(&event.TotalTickets, &event.SoldTickets, &event.TicketsReserved)
		if err != nil {
			return fmt.Errorf("event not found: %v", err)
		}

		if ticketsToBuy <= 0 {
			return fmt.Errorf("tickets to buy must be greater than zero")
		}

		// Check both sold and reserved tickets
		if event.SoldTickets+ticketsToBuy > event.TotalTickets {
			return fmt.Errorf("not enough tickets available")
		}

		newSoldTickets := event.SoldTickets + ticketsToBuy
		ticketsLeft := event.TotalTickets - newSoldTickets - event.TicketsReserved

		query = `UPDATE events SET sold_tickets = $1, tickets_left = $2 WHERE id = $3`
		_, err = r.DB.Exec(query, newSoldTickets, ticketsLeft, eventID)
		if err != nil {
			return fmt.Errorf("failed to update tickets sold: %v", err)
		}

		return nil
	})
}

// ReserveTicket temporarily reserves a ticket for an event
func (r *EventRepository) ReserveTicket(eventID int) error {
	return r.breaker.Execute(func() error {
		tx, err := r.DB.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %v", err)
		}

		// Attempt to rollback by default - will be ignored if commit succeeds
		defer tx.Rollback()

		var event struct {
			TotalTickets    int `json:"total_tickets"`
			SoldTickets     int `json:"sold_tickets"`
			TicketsReserved int `json:"tickets_reserved"`
			TicketsLeft     int `json:"tickets_left"`
		}

		// Get current event state with row lock (FOR UPDATE ensures exclusive access)
		query := `SELECT total_tickets, sold_tickets, tickets_reserved, tickets_left 
				  FROM events 
				  WHERE id = $1 
				  FOR UPDATE`
		err = tx.QueryRow(query, eventID).Scan(
			&event.TotalTickets,
			&event.SoldTickets,
			&event.TicketsReserved,
			&event.TicketsLeft,
		)
		if err != nil {
			return fmt.Errorf("event not found: %v", err)
		}

		// Check if tickets are available
		availableTickets := event.TotalTickets - event.SoldTickets - event.TicketsReserved
		if availableTickets <= 0 {
			return fmt.Errorf("no tickets available")
		}

		// Update reserved count and available tickets
		query = `UPDATE events 
				 SET tickets_reserved = tickets_reserved + 1,
				     tickets_left = tickets_left - 1
				 WHERE id = $1`
		_, err = tx.Exec(query, eventID)
		if err != nil {
			return fmt.Errorf("failed to reserve ticket: %v", err)
		}

		// Commit the transaction
		if err = tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit transaction: %v", err)
		}

		return nil
	})
}

// CompleteReservation converts a reservation into a sold ticket
func (r *EventRepository) CompleteReservation(eventID int) error {
	return r.breaker.Execute(func() error {
		tx, err := r.DB.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %v", err)
		}

		defer tx.Rollback()

		// Get current event state with row lock
		query := `SELECT tickets_reserved FROM events WHERE id = $1 FOR UPDATE`
		var ticketsReserved int
		err = tx.QueryRow(query, eventID).Scan(&ticketsReserved)
		if err != nil {
			return fmt.Errorf("event not found: %v", err)
		}

		if ticketsReserved <= 0 {
			return fmt.Errorf("no reserved tickets to complete")
		}

		// Convert a reserved ticket to a sold ticket
		query = `UPDATE events 
				 SET tickets_reserved = tickets_reserved - 1,
				     sold_tickets = sold_tickets + 1
				 WHERE id = $1`
		_, err = tx.Exec(query, eventID)
		if err != nil {
			return fmt.Errorf("failed to complete reservation: %v", err)
		}

		// Commit the transaction
		if err = tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit transaction: %v", err)
		}

		return nil
	})
}

// ReleaseReservation releases a reserved ticket
func (r *EventRepository) ReleaseReservation(eventID int) error {
	return r.breaker.Execute(func() error {
		tx, err := r.DB.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %v", err)
		}

		defer tx.Rollback()

		// Get current event state with row lock
		query := `SELECT tickets_reserved FROM events WHERE id = $1 FOR UPDATE`
		var ticketsReserved int
		err = tx.QueryRow(query, eventID).Scan(&ticketsReserved)
		if err != nil {
			return fmt.Errorf("event not found: %v", err)
		}

		if ticketsReserved <= 0 {
			return fmt.Errorf("no reserved tickets to release")
		}

		// Release a reserved ticket
		query = `UPDATE events 
				 SET tickets_reserved = tickets_reserved - 1,
				     tickets_left = tickets_left + 1
				 WHERE id = $1`
		_, err = tx.Exec(query, eventID)
		if err != nil {
			return fmt.Errorf("failed to release reservation: %v", err)
		}

		// Commit the transaction
		if err = tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit transaction: %v", err)
		}

		return nil
	})
}
