package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reservation-service/internal/db/models"
	"reservation-service/internal/db/repos"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type ReservationHandler struct {
	purchaseRepo  *repos.PurchaseRepository
	ticketClient  *http.Client
	ticketBaseURL string
}

func NewReservationHandler(purchaseRepo *repos.PurchaseRepository, ticketClient *http.Client) *ReservationHandler {
	ticketBaseURL := "http://ticket-service-1:8082" //"http://gateway1:8083/api/v1/tickets"
	fmt.Printf("Initialized ReservationHandler with ticketBaseURL: %s\n", ticketBaseURL)
	return &ReservationHandler{
		purchaseRepo:  purchaseRepo,
		ticketClient:  ticketClient,
		ticketBaseURL: ticketBaseURL,
	}
}

func (h *ReservationHandler) GetTicket(c *gin.Context) {
	// Extract ticket ID from URL parameter
	ticketIDStr := c.Param("id")
	ticketID, err := strconv.Atoi(ticketIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ticket ID"})
		return
	}

	// Fetch ticket from ticket-service
	var ticket struct {
		ID      int     `json:"id"`
		EventID int     `json:"event_id"`
		Price   float64 `json:"price"`
		Status  string  `json:"status"`
	}
	ticketURL := fmt.Sprintf("%s/%d", h.ticketBaseURL, ticketID)
	fmt.Printf("Fetching ticket from URL: %s\n", ticketURL)
	ticketResp, err := h.ticketClient.Get(ticketURL)
	if err != nil || ticketResp.StatusCode != http.StatusOK {
		status := http.StatusInternalServerError
		if ticketResp != nil {
			status = ticketResp.StatusCode
		}
		fmt.Printf("Failed to fetch ticket: URL=%s, err=%v, status=%d\n", ticketURL, err, status)
		c.JSON(status, gin.H{"error": "Failed to fetch ticket"})
		return
	}
	defer ticketResp.Body.Close()

	// Decode the ticket response
	if err := json.NewDecoder(ticketResp.Body).Decode(&ticket); err != nil {
		fmt.Printf("Failed to decode ticket response: err=%v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode ticket"})
		return
	}

	// Return the ticket
	fmt.Printf("Successfully fetched ticket: %+v\n", ticket)
	c.JSON(http.StatusOK, ticket)
}

func (h *ReservationHandler) ReserveTicket(c *gin.Context) {
	var req struct {
		UserID   int `json:"user_id" binding:"required"`
		TicketID int `json:"ticket_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fmt.Printf("Invalid request: err=%v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}
	fmt.Printf("Received reservation request: user_id=%d, ticket_id=%d\n", req.UserID, req.TicketID)

	// Step 1: Fetch ticket from ticket-service
	var ticket struct {
		ID      int     `json:"id"`
		EventID int     `json:"event_id"`
		Price   float64 `json:"price"`
		Status  string  `json:"status"`
	}
	ticketURL := fmt.Sprintf("%s/%d", h.ticketBaseURL, req.TicketID)
	fmt.Printf("Fetching ticket from URL: %s\n", ticketURL)
	ticketResp, err := h.ticketClient.Get(ticketURL)
	if err != nil || ticketResp.StatusCode != http.StatusOK {
		status := http.StatusInternalServerError
		if ticketResp != nil {
			status = ticketResp.StatusCode
		}
		fmt.Printf("Failed to fetch ticket: URL=%s, err=%v, status=%d\n", ticketURL, err, status)
		c.JSON(status, gin.H{"error": "Failed to fetch ticket"})
		return
	}
	defer ticketResp.Body.Close()
	if err := json.NewDecoder(ticketResp.Body).Decode(&ticket); err != nil {
		fmt.Printf("Failed to decode ticket response: err=%v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode ticket"})
		return
	}
	fmt.Printf("Fetched ticket: %+v\n", ticket)
	if ticket.Status != "available" {
		fmt.Printf("Ticket not available: status=%s\n", ticket.Status)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ticket not available"})
		return
	}

	// Step 4: Process payment (placeholder)
	paymentSuccess := h.processPayment(req.UserID, ticket.Price)
	if !paymentSuccess {
		fmt.Println("Payment failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Payment failed"})
		return
	}
	fmt.Println("Payment processed successfully")

	// Step 5: Create purchase record in reservation_db
	purchase := &models.Purchase{
		TicketID:     req.TicketID,
		UserID:       req.UserID,
		EventID:      ticket.EventID,
		PurchaseDate: time.Now(),
		Status:       "pending",
	}
	fmt.Printf("Creating purchase: %+v\n", purchase)
	createdPurchase, err := h.purchaseRepo.CreatePurchase(purchase)
	if err != nil {
		fmt.Printf("Failed to create purchase: err=%v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create purchase: " + err.Error()})
		return
	}
	fmt.Printf("Created purchase: %+v\n", createdPurchase)

	// Step 6: Update ticket status via ticket-service using PUT
	updateReq := map[string]string{"status": "sold"} // Changed from "reserved" to "sold"
	updateBody, _ := json.Marshal(updateReq)
	updateURL := fmt.Sprintf("%s/%d/status", h.ticketBaseURL, req.TicketID)
	fmt.Printf("Sending PUT request to: %s with body: %s\n", updateURL, string(updateBody))
	updateRequest, err := http.NewRequest(http.MethodPut, updateURL, bytes.NewBuffer(updateBody))
	if err != nil {
		fmt.Printf("Failed to create PUT request: err=%v\n", err)
		h.purchaseRepo.UpdatePurchaseStatus(createdPurchase.ID, "cancelled")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create PUT request: " + err.Error()})
		return
	}
	updateRequest.Header.Set("Content-Type", "application/json")
	updateResp, err := h.ticketClient.Do(updateRequest)
	if err != nil || updateResp.StatusCode != http.StatusOK {
		// Read response body for detailed error
		var errorResp map[string]interface{}
		bodyBytes, _ := io.ReadAll(updateResp.Body)
		updateResp.Body.Close()
		if len(bodyBytes) > 0 {
			json.Unmarshal(bodyBytes, &errorResp)
		}
		fmt.Printf("Failed to update ticket status: URL=%s, err=%v, status=%d, response=%v\n", updateURL, err, updateResp.StatusCode, errorResp)
		h.purchaseRepo.UpdatePurchaseStatus(createdPurchase.ID, "cancelled")
		status := http.StatusInternalServerError
		if updateResp != nil {
			status = updateResp.StatusCode
		}
		c.JSON(status, gin.H{"error": "Failed to update ticket status"})
		return
	}
	defer updateResp.Body.Close()
	fmt.Println("Ticket status updated successfully")

	// Step 7: Confirm purchase in reservation_db
	fmt.Printf("Confirming purchase ID %d with status 'confirmed'\n", createdPurchase.ID)
	_, err = h.purchaseRepo.UpdatePurchaseStatus(createdPurchase.ID, "confirmed")
	if err != nil {
		fmt.Printf("Failed to confirm purchase: err=%v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to confirm purchase: " + err.Error()})
		return
	}
	fmt.Printf("Purchase confirmed: ID=%d\n", createdPurchase.ID)

	// Return the confirmed purchase
	c.JSON(http.StatusOK, createdPurchase)
}

func (h *ReservationHandler) processPayment(userID int, amount float64) bool {
	// Placeholder for payment processing logic
	fmt.Printf("Processing payment: user_id=%d, amount=%.2f\n", userID, amount)
	return true
}
