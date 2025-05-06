package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"reservation-service/internal/db/models"
	"reservation-service/internal/db/repos"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	repo       *repos.PurchaseRepository
	httpClient *http.Client
}

func NewHandler(repo *repos.PurchaseRepository) *Handler {
	return &Handler{
		repo: repo,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

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

	//#########################################################
	//#########################################################

	// paymentSuccess, paymentErr := h.handlePayment(1000) //$$$$$$$$$ here is hardcoded until it's fixed in event
	// if !paymentSuccess {
	// 	c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Payment error: %v", paymentErr)})
	// 	return
	// }

	//##########################################################
	//##########################################################

	paymentSuccess, paymentErr := h.handlePayment(1000) // Hardcoded until event price is fetched
	if !paymentSuccess {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Payment error: %v", paymentErr)})
		return
	}

	ticketReq := struct {
		EventID int `json:"event_id"`
		UserID  int `json:"user_id"`
	}{EventID: input.EventID, UserID: input.UserID}
	ticketReqBody, err := json.Marshal(ticketReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to marshal ticket request: %v", err)})
		return
	}
	resp, err := h.httpClient.Post(os.Getenv("TICKET_SERVICE_URL")+"/v1", "application/json", bytes.NewBuffer(ticketReqBody))
	if err != nil || resp.StatusCode != http.StatusCreated {
		status := http.StatusInternalServerError
		if resp != nil {
			defer resp.Body.Close()
			status = resp.StatusCode
		}
		c.JSON(status, gin.H{"error": fmt.Sprintf("Ticket service error: %v", err)})
		return
	}

	var ticketResp struct {
		TicketID int `json:"ticket_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ticketResp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse ticket response"})
		return
	}

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

func (h *Handler) handlePayment(amount int) (bool, error) {
	log.Println("Initiating payment process")

	if amount <= 0 {
		return false, fmt.Errorf("invalid amount: must be greater than zero")
	}

	payload := map[string]interface{}{
		"amount": amount,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return false, fmt.Errorf("failed to marshal payload: %v", err)
	}

	req, err := http.NewRequest("POST", os.Getenv("PAYMENT_SERVICE_URL")+"/create-payment-intent", bytes.NewBuffer(body))
	if err != nil {
		return false, fmt.Errorf("failed to create payment request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("payment service error: %v", err)
	}
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errorMsg map[string]interface{}
		if err := json.Unmarshal(body, &errorMsg); err != nil {
			return false, fmt.Errorf("payment service returned status %d", resp.StatusCode)
		}
		return false, fmt.Errorf("payment service returned status %d: %v", resp.StatusCode, errorMsg)
	}

	var paymentResp struct {
		ClientSecret   string `json:"client_secret"`
		IdempotencyKey string `json:"idempotency_key"`
	}
	if err := json.Unmarshal(body, &paymentResp); err != nil {
		return false, fmt.Errorf("failed to parse payment response: %v", err)
	}
	if paymentResp.ClientSecret == "" || paymentResp.IdempotencyKey == "" {
		return false, fmt.Errorf("missing fields in payment response")
	}

	log.Printf("Payment successful: client_secret=%s, idempotency_key=%s\n", paymentResp.ClientSecret, paymentResp.IdempotencyKey)
	return true, nil
}
