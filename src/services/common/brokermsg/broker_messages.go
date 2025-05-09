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

	// Audit and analytics
	TopicAuditLog       = "audit.log"
	TopicAnalyticsEvent = "analytics.event"
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
	ReservationID  int    `json:"reservation_id"`
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

// AuditLogMessage is published for audit logging
type AuditLogMessage struct {
	Action     string                 `json:"action"`
	UserID     int                    `json:"user_id"`
	EntityType string                 `json:"entity_type"`
	EntityID   int                    `json:"entity_id"`
	Timestamp  int64                  `json:"timestamp"` // Unix timestamp
	Details    map[string]interface{} `json:"details"`
}

// AnalyticsEventMessage is published for analytics
type AnalyticsEventMessage struct {
	EventType string                 `json:"event_type"`
	Timestamp int64                  `json:"timestamp"` // Unix timestamp
	Metadata  map[string]interface{} `json:"metadata"`
}
