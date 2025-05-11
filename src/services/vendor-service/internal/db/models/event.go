package models

type Event struct {
	VendorID           int     `json:"vendor_id"`
	Name               string  `json:"name"`
	Date               string  `json:"date"`
	Location           string  `json:"location"`
	TotalTickets       int     `json:"total_tickets"`
	Price              float64 `json:"price"`
	ReservationTimeout int     `json:"reservation_timeout"`
}
