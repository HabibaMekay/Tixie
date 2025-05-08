package repos

import (
    "database/sql"
    "event-service/internal/db/models"
    "fmt"
)

type EventRepository struct {
    DB *sql.DB
}

func NewEventRepository(db *sql.DB) *EventRepository {
    return &EventRepository{DB: db}
}

func (r *EventRepository) GetAllEvents() ([]models.Event, error) {
    query := `SELECT id, name, date, venue, total_tickets, vendor_id, price FROM events`
    rows, err := r.DB.Query(query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var events []models.Event
    for rows.Next() {
        var e models.Event
        if err := rows.Scan(&e.ID, &e.Name, &e.Date, &e.Venue, &e.TotalTickets, &e.VendorID, &e.Price); err != nil {
            return nil, err
        }
        events = append(events, e)
    }

    return events, nil
}

func (r *EventRepository) CreateEvent(event models.Event) error {
    query := `
        INSERT INTO events (name, date, venue, total_tickets, vendor_id, price)
        VALUES ($1, $2, $3, $4, $5, $6)
    `
    _, err := r.DB.Exec(query, event.Name, event.Date, event.Venue, event.TotalTickets, event.VendorID, event.Price)
    return err
}

func (r *EventRepository) GetEventByID(id int) (models.Event, error) {
    var e models.Event
    query := `SELECT id, name, date, venue, total_tickets, vendor_id, price FROM events WHERE id = $1`
    err := r.DB.QueryRow(query, id).Scan(&e.ID, &e.Name, &e.Date, &e.Venue, &e.TotalTickets, &e.VendorID, &e.Price)
    return e, err
}


func (r *EventRepository) UpdateTicketsSold(eventID string, ticketsToBuy int) error {
    var event struct {
        TotalTickets int `json:"total_tickets"`
        SoldTickets  int `json:"sold_tickets"`
    }

    query := `SELECT total_tickets, sold_tickets FROM events WHERE id = $1`
    err := r.DB.QueryRow(query, eventID).Scan(&event.TotalTickets, &event.SoldTickets)
    if err != nil {
        return fmt.Errorf("event not found: %v", err) 
    }

    if ticketsToBuy <= 0 {
        return fmt.Errorf("tickets to buy must be greater than zero")
    }
    if event.SoldTickets+ticketsToBuy > event.TotalTickets {
        return fmt.Errorf("not enough tickets available")
    }

    newSoldTickets := event.SoldTickets + ticketsToBuy
    ticketsLeft := event.TotalTickets - newSoldTickets

    query = `UPDATE events SET sold_tickets = $1, tickets_left = $2 WHERE id = $3`
    _, err = r.DB.Exec(query, newSoldTickets, ticketsLeft, eventID)
    if err != nil {
        return fmt.Errorf("failed to update tickets sold: %v", err)
    }

    return nil
}


