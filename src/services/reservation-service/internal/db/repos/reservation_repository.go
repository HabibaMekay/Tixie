package repos

import (
	"reservation-service/internal/db/models"
	"time"

	"github.com/jmoiron/sqlx"
)

// ReservationRepository handles database operations for ticket reservations.
type ReservationRepository struct {
	db *sqlx.DB
}

// NewReservationRepository creates a new ReservationRepository.
func NewReservationRepository(db *sqlx.DB) *ReservationRepository {
	return &ReservationRepository{db: db}
}

// CreateReservation creates a new ticket reservation
func (r *ReservationRepository) CreateReservation(eventID, userID int, timeoutSeconds int) (*models.Reservation, error) {
	var reservation models.Reservation

	createdAt := time.Now().UTC()
	expirationTime := createdAt.Add(time.Duration(timeoutSeconds) * time.Second)

	err := r.db.QueryRowx(
		`INSERT INTO reservations (event_id, user_id, status, created_at, expiration_time) 
		 VALUES ($1, $2, $3, $4, $5) 
		 RETURNING *`,
		eventID, userID, "pending", createdAt, expirationTime,
	).StructScan(&reservation)

	if err != nil {
		return nil, err
	}

	return &reservation, nil
}

// GetReservation retrieves a reservation by ID
func (r *ReservationRepository) GetReservation(reservationID int) (*models.Reservation, error) {
	var reservation models.Reservation

	err := r.db.Get(
		&reservation,
		`SELECT * FROM reservations WHERE id = $1`,
		reservationID,
	)

	if err != nil {
		return nil, err
	}

	return &reservation, nil
}

// UpdateReservationStatus updates the status of a reservation
func (r *ReservationRepository) UpdateReservationStatus(reservationID int, status string) error {
	_, err := r.db.Exec(
		`UPDATE reservations SET status = $1 WHERE id = $2`,
		status, reservationID,
	)

	return err
}

// GetExpiredReservations gets all reservations that have expired but still have "pending" status
func (r *ReservationRepository) GetExpiredReservations() ([]*models.Reservation, error) {
	reservations := []*models.Reservation{}

	err := r.db.Select(
		&reservations,
		`SELECT * FROM reservations 
		 WHERE status = 'pending' 
		 AND expiration_time < $1`,
		time.Now().UTC(),
	)

	if err != nil {
		return nil, err
	}

	return reservations, nil
}

// CompleteReservation marks a reservation as completed
func (r *ReservationRepository) CompleteReservation(reservationID int) error {
	_, err := r.db.Exec(
		`UPDATE reservations SET status = 'completed' WHERE id = $1`,
		reservationID,
	)

	return err
}

// ExpireReservation marks a reservation as expired
func (r *ReservationRepository) ExpireReservation(reservationID int) error {
	_, err := r.db.Exec(
		`UPDATE reservations SET status = 'expired' WHERE id = $1`,
		reservationID,
	)

	return err
}
