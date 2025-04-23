package repos

import (
	"reservation-service/internal/db/models"

	"github.com/jmoiron/sqlx"
)

// PurchaseRepository handles database operations for purchases.
type PurchaseRepository struct {
	db *sqlx.DB
}

// NewPurchaseRepository creates a new PurchaseRepository.
func NewPurchaseRepository(db *sqlx.DB) *PurchaseRepository {
	return &PurchaseRepository{db: db}
}

// CreatePurchase creates a new purchase record.
func (r *PurchaseRepository) CreatePurchase(purchase *models.Purchase) (*models.Purchase, error) {
	var createdPurchase models.Purchase
	err := r.db.QueryRowx(
		"INSERT INTO purchases (ticket_id, user_id, event_id, purchase_date, status) VALUES ($1, $2, $3, $4, $5) RETURNING *",
		purchase.TicketID, purchase.UserID, purchase.EventID, purchase.PurchaseDate, purchase.Status,
	).StructScan(&createdPurchase)
	if err != nil {
		return nil, err
	}
	return &createdPurchase, nil
}

// UpdatePurchaseStatus updates the status of a purchase.
func (r *PurchaseRepository) UpdatePurchaseStatus(purchaseID int, status string) (*models.Purchase, error) {
	var updatedPurchase models.Purchase
	err := r.db.QueryRowx(
		"UPDATE purchases SET status=$1 WHERE id=$2 RETURNING *",
		status, purchaseID,
	).StructScan(&updatedPurchase)
	if err != nil {
		return nil, err
	}
	return &updatedPurchase, nil
}
