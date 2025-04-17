package main

import (
	"log"
	"reservation-service/internal/db/models"
	"reservation-service/internal/db/repos"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type ReservationService struct {
	ticketDB      *sqlx.DB
	reservationDB *sqlx.DB
	purchaseRepo  *repos.PurchaseRepository
}

func NewReservationService() *ReservationService {
	// Connect to ticket_db (for tickets and users)
	ticketConnStr := "host=ticket-db port=5432 user=postgres password=your_password dbname=ticket_db sslmode=disable"
	ticketDB, err := sqlx.Connect("postgres", ticketConnStr)
	if err != nil {
		log.Fatal(err)
	}

	// Connect to reservation_db (for purchases)
	reservationConnStr := "host=reservation-db port=5432 user=postgres password=your_password dbname=reservation_db sslmode=disable"
	reservationDB, err := sqlx.Connect("postgres", reservationConnStr)
	if err != nil {
		log.Fatal(err)
	}

	purchaseRepo := repos.NewPurchaseRepository(reservationDB)

	return &ReservationService{
		ticketDB:      ticketDB,
		reservationDB: reservationDB,
		purchaseRepo:  purchaseRepo,
	}
}

func (s *ReservationService) ReserveTicket(c *gin.Context) {
	var req struct {
		UserID   int `json:"user_id"`
		TicketID int `json:"ticket_id"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	// Step 1: Check ticket availability in ticket_db
	var ticket struct {
		ID      int     `db:"id"`
		EventID int     `db:"event_id"`
		Price   float64 `db:"price"`
		Status  string  `db:"status"`
	}
	err := s.ticketDB.Get(&ticket, "SELECT id, event_id, price, status FROM tickets WHERE id=$1", req.TicketID)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to fetch ticket"})
		return
	}
	if ticket.Status != "available" {
		c.JSON(400, gin.H{"error": "Ticket not available"})
		return
	}

	// Step 2: Verify user exists in ticket_db
	var userExists bool
	err = s.ticketDB.Get(&userExists, "SELECT EXISTS(SELECT 1 FROM ticket WHERE id=$1)", req.UserID) //////////////////////////////
	if err != nil || !userExists {
		c.JSON(400, gin.H{"error": "User not found"})
		return
	}

	// // Step 3: Verify event exists in ticket_db
	// var eventExists bool
	// err = s.ticketDB.Get(&eventExists, "SELECT EXISTS(SELECT 1 FROM events WHERE id=$1)", ticket.EventID)
	// if err != nil || !eventExists {
	//     c.JSON(400, gin.H{"error": "Event not found"})
	//     return
	// }

	// Step 4: Process payment (placeholder)
	paymentSuccess := s.processPayment(req.UserID, ticket.Price)
	if !paymentSuccess {
		c.JSON(500, gin.H{"error": "Payment failed"})
		return
	}

	// Step 5: Create purchase record in reservation_db
	purchase := &models.Purchase{
		TicketID:     req.TicketID,
		UserID:       req.UserID,
		EventID:      ticket.EventID,
		PurchaseDate: time.Now(),
		Status:       "pending",
	}
	createdPurchase, err := s.purchaseRepo.CreatePurchase(purchase)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to create purchase"})
		return
	}

	// Step 6: Update ticket status in ticket_db
	_, err = s.ticketDB.Exec("UPDATE tickets SET status='reserved' WHERE id=$1 AND status='available'", req.TicketID)
	if err != nil {
		// Rollback purchase in reservation_db
		s.purchaseRepo.UpdatePurchaseStatus(createdPurchase.ID, "cancelled")
		c.JSON(500, gin.H{"error": "Failed to update ticket status"})
		return
	}

	// Step 7: Confirm purchase in reservation_db
	_, err = s.purchaseRepo.UpdatePurchaseStatus(createdPurchase.ID, "confirmed")
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to confirm purchase"})
		return
	}

	c.JSON(200, createdPurchase)
}

func (s *ReservationService) processPayment(userID int, amount float64) bool {
	// Placeholder for payment integration
	return true
}

func main() {
	service := NewReservationService()
	router := gin.Default()

	router.POST("/reserve", service.ReserveTicket)

	router.Run(":8081")
}
