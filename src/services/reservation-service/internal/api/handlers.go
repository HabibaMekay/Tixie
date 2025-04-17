package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"reservation-service/internal/db/models"
	"reservation-service/internal/db/repos"
	"time"

	"github.com/gin-gonic/gin"
)

type ReservationHandler struct {
	purchaseRepo *repos.PurchaseRepository
	ticketClient *http.Client
}

func NewReservationHandler(purchaseRepo *repos.PurchaseRepository, ticketClient *http.Client) *ReservationHandler {
	return &ReservationHandler{
		purchaseRepo: purchaseRepo,
		ticketClient: ticketClient,
	}
}

func (h *ReservationHandler) ReserveTicket(c *gin.Context) {
	var req struct {
		UserID   int `json:"user_id" binding:"required"`
		TicketID int `json:"ticket_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Step 1: Fetch ticket from ticket-service
	var ticket struct {
		ID      int     `json:"id"`
		EventID int     `json:"event_id"`
		Price   float64 `json:"price"`
		Status  string  `json:"status"`
	}
	ticketResp, err := h.ticketClient.Get("http://ticket-service:8080/tickets/" + fmt.Sprint(req.TicketID))
	if err != nil || ticketResp.StatusCode != http.StatusOK {
		status := http.StatusInternalServerError
		if ticketResp != nil {
			status = ticketResp.StatusCode
		}
		c.JSON(status, gin.H{"error": "Failed to fetch ticket"})
		return
	}
	defer ticketResp.Body.Close()
	if err := json.NewDecoder(ticketResp.Body).Decode(&ticket); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode ticket"})
		return
	}
	if ticket.Status != "available" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ticket not available"})
		return
	}

	// Step 2: Verify user exists via ticket-service
	userResp, err := h.ticketClient.Get("http://ticket-service:8080/users/" + fmt.Sprint(req.UserID) + "/exists")
	if err != nil || userResp.StatusCode != http.StatusOK {
		status := http.StatusInternalServerError
		if userResp != nil {
			status = userResp.StatusCode
		}
		c.JSON(status, gin.H{"error": "Failed to check user"})
		return
	}
	defer userResp.Body.Close()
	var userExists struct {
		Exists bool `json:"exists"`
	}
	if err := json.NewDecoder(userResp.Body).Decode(&userExists); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode user check"})
		return
	}
	if !userExists.Exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User not found"})
		return
	}

	// Step 3: Verify event exists via ticket-service
	eventResp, err := h.ticketClient.Get("http://ticket-service:8080/events/" + fmt.Sprint(ticket.EventID) + "/exists")
	if err != nil || eventResp.StatusCode != http.StatusOK {
		status := http.StatusInternalServerError
		if eventResp != nil {
			status = eventResp.StatusCode
		}
		c.JSON(status, gin.H{"error": "Failed to check event"})
		return
	}
	defer eventResp.Body.Close()
	var eventExists struct {
		Exists bool `json:"exists"`
	}
	if err := json.NewDecoder(eventResp.Body).Decode(&eventExists); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode event check"})
		return
	}
	if !eventExists.Exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Event not found"})
		return
	}

	// Step 4: Process payment (placeholder)
	paymentSuccess := h.processPayment(req.UserID, ticket.Price)
	if !paymentSuccess {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Payment failed"})
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
	createdPurchase, err := h.purchaseRepo.CreatePurchase(purchase)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create purchase: " + err.Error()})
		return
	}

	// Step 6: Update ticket status via ticket-service using PATCH
	updateReq := map[string]string{"status": "reserved"}
	updateBody, _ := json.Marshal(updateReq)
	updateURL := "http://ticket-service:8080/tickets/" + fmt.Sprint(req.TicketID) + "/status"
	updateRequest, err := http.NewRequest(http.MethodPatch, updateURL, bytes.NewBuffer(updateBody))
	if err != nil {
		h.purchaseRepo.UpdatePurchaseStatus(createdPurchase.ID, "cancelled")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create PATCH request: " + err.Error()})
		return
	}
	updateRequest.Header.Set("Content-Type", "application/json")
	updateResp, err := h.ticketClient.Do(updateRequest)
	if err != nil || updateResp.StatusCode != http.StatusOK {
		h.purchaseRepo.UpdatePurchaseStatus(createdPurchase.ID, "cancelled")
		status := http.StatusInternalServerError
		if updateResp != nil {
			status = updateResp.StatusCode
		}
		c.JSON(status, gin.H{"error": "Failed to update ticket status"})
		return
	}
	defer updateResp.Body.Close()

	// Step 7: Confirm purchase in reservation_db
	_, err = h.purchaseRepo.UpdatePurchaseStatus(createdPurchase.ID, "confirmed")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to confirm purchase: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, createdPurchase)
}

func (h *ReservationHandler) processPayment(userID int, amount float64) bool {
	// Placeholder for payment processing logic
	return true
}
