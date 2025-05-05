package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"reservation-service/internal/db/models"
	"reservation-service/internal/db/repos"
	"time"

	"github.com/gin-gonic/gin"
)

// Handler holds dependencies for API handlers.
type Handler struct {
	repo           *repos.PurchaseRepository
	gatewayBaseURL string // e.g., "http://gateway1:8083/api/v1"
	httpClient     *http.Client
}

// NewHandler creates a new Handler with dependencies.
func NewHandler(repo *repos.PurchaseRepository, gatewayBaseURL string) *Handler {
	return &Handler{
		repo:           repo,
		gatewayBaseURL: gatewayBaseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// CreatePurchase creates a new purchase and triggers ticket generation.
func (h *Handler) ReserveTicket(c *gin.Context) {
	log.Println("ReserveTicket called")
	var input struct {
		EventID int `json:"event_id" binding:"required,gt=0"`
		UserID  int `json:"user_id" binding:"required,gt=0"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	// Validate event_id via event service
	if err := h.validateEvent(input.EventID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid event_id: %v", err)})
		return
	}

	// Validate user_id via user service
	if err := h.validateUser(input.UserID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid user_id: %v", err)})
		return
	}

	// Call ticket service to generate a ticket
	ticketReq := struct {
		EventID int `json:"event_id"`
		UserID  int `json:"user_id"`
	}{EventID: input.EventID, UserID: input.UserID}
	ticketReqBody, _ := json.Marshal(ticketReq)
	resp, err := h.httpClient.Post(h.gatewayBaseURL+"/tickets", "application/json", bytes.NewBuffer(ticketReqBody))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to contact ticket service: %v", err)})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Ticket service error (status: %d)", resp.StatusCode)})
		return
	}

	var ticketResp struct {
		TicketID int `json:"ticket_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ticketResp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse ticket response"})
		return
	}

	// Create purchase
	purchase := &models.Purchase{
		TicketID:     ticketResp.TicketID,
		UserID:       input.UserID,
		EventID:      input.EventID,
		PurchaseDate: time.Now().UTC(),
		Status:       "confirmed",
	}

	createdPurchase, err := h.repo.CreatePurchase(purchase)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, createdPurchase)
}

// validateEvent checks if an event exists via the event service.
func (h *Handler) validateEvent(eventID int) error {
	url := fmt.Sprintf("%s/events/%d", h.gatewayBaseURL, eventID)
	resp, err := h.httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("failed to contact event service: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("event not found or service error (status: %d)", resp.StatusCode)
	}
	return nil
}

// validateUser checks if a user exists via the user service.
func (h *Handler) validateUser(userID int) error {
	url := fmt.Sprintf("%s/users/%d", h.gatewayBaseURL, userID)
	resp, err := h.httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("failed to contact user service: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("user not found or service error (status: %d)", resp.StatusCode)
	}
	return nil
}
