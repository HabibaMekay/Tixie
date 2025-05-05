package repos

import (
    "database/sql"
    "event-service/internal/db/models"
)

type EventRepository struct {
    DB *sql.DB
}

func NewEventRepository(db *sql.DB) *EventRepository {
    return &EventRepository{DB: db}
}

func (r *EventRepository) GetAllEvents() ([]models.Event, error) {
    query := `SELECT id, name, date, venue, total_tickets, vendor_id FROM events`
    rows, err := r.DB.Query(query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var events []models.Event
    for rows.Next() {
        var e models.Event
        if err := rows.Scan(&e.ID, &e.Name, &e.Date, &e.Venue, &e.TotalTickets, &e.VendorID); err != nil {
            return nil, err
        }
        events = append(events, e)
    }

    return events, nil
}

func (r *EventRepository) CreateEvent(event models.Event) error {
    query := `
        INSERT INTO events (name, date, venue, total_tickets, vendor_id)
        VALUES ($1, $2, $3, $4, $5)
    `
    _, err := r.DB.Exec(query, event.Name, event.Date, event.Venue, event.TotalTickets, event.VendorID)
    return err
}

