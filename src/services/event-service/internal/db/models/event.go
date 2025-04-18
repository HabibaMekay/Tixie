package models

type Event struct {
    ID           int    `json:"id"`
    Name         string `json:"name"`
    Date         string `json:"date"`
    Venue        string `json:"venue"`
    TotalTickets int    `json:"total_tickets"`
    VendorID     int    `json:"vendor_id"`
}
