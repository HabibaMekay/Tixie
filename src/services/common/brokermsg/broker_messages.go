package brokermsg

// Topic name constants for consistent broker usage across services, this was made to make the code more comprehensive and improve maintainability more than functionality. this would have easily been dropped in favor of integer representations which might improve performance a teeny tiny bit.
const (
	// Reservation related topics
	TopicReservationCreated   = "reservation.created"
	TopicReservationCompleted = "reservation.completed"
	TopicReservationExpired   = "reservation.expired"

	// Payment related topics
	TopicPaymentProcessed = "payment.processed"
	TopicPaymentFailed    = "payment.failed"

	// Ticket related topics
	TopicTicketIssued    = "ticket.issued"
	TopicTicketValidated = "ticket.validated"

	// Notification related topics
	TopicNotificationEmail = "notification.email"

	// Event related topics
	TopicEventCreated = "event.created"
)

// Message structures

// ReservationCreatedMessage is published when a new reservation is created
type ReservationCreatedMessage struct {
	ReservationID  int   `json:"reservation_id"`
	EventID        int   `json:"event_id"`
	UserID         int   `json:"user_id"`
	ExpirationTime int64 `json:"expiration_time"` // Unix timestamp
}

// ReservationCompletedMessage is published when a reservation is completed
type ReservationCompletedMessage struct {
	ReservationID int `json:"reservation_id"`
	EventID       int `json:"event_id"`
	UserID        int `json:"user_id"`
	Amount        int `json:"amount"` // Amount in cents
}

// ReservationExpiredMessage is published when a reservation expires
type ReservationExpiredMessage struct {
	ReservationID int `json:"reservation_id"`
	EventID       int `json:"event_id"`
}

// PaymentProcessedMessage is published when a payment is successfully processed
type PaymentProcessedMessage struct {
	ReservationID int    `json:"reservation_id"`
	Amount        int    `json:"amount"`
	PaymentID     string `json:"payment_id"`
}

// PaymentFailedMessage is published when a payment fails
type PaymentFailedMessage struct {
	ReservationID int    `json:"reservation_id"`
	Reason        string `json:"reason"`
}

// TicketIssuedMessage is published when a ticket is successfully issued
type TicketIssuedMessage struct {
	TicketID       int    `json:"ticket_id"`
	TicketCode     string `json:"ticket_code"`
	UserID         int    `json:"user_id"`
	EventID        int    `json:"event_id"`
	RecipientEmail string `json:"recipient_email"`
}

// EmailNotificationMessage is published when an email needs to be sent
type EmailNotificationMessage struct {
	RecipientEmail string                 `json:"recipient_email"`
	Subject        string                 `json:"subject"`
	TemplateID     string                 `json:"template_id"`
	TemplateData   map[string]interface{} `json:"template_data"`
}

// EventCreatedMessage is published when a new event is created
type EventCreatedMessage struct {
	Name               string  `json:"name"`
	Date               string  `json:"date"`
	Venue              string  `json:"venue"`
	TotalTickets       int     `json:"total_tickets"`
	VendorID           int     `json:"vendor_id"`
	Price              float64 `json:"price"`
	ReservationTimeout int     `json:"reservation_timeout"`
}
