package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"reservation-service/internal/db/models"
	"reservation-service/internal/db/repos"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	brokerPkg "tixie.local/broker"
)

type Handler struct {
	repo       *repos.PurchaseRepository
	httpClient *http.Client
	broker     *brokerPkg.Broker
}

func NewHandler(repo *repos.PurchaseRepository) *Handler {
	broker, err := brokerPkg.NewBroker(os.Getenv("RABBITMQ_URL"), "payment", "topic")
	if err != nil {
		log.Printf("Warning: Failed to create broker: %v", err)
	}

	return &Handler{
		repo: repo,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		broker: broker,
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

	// Fetch event details to get price so we can log it as if we're actually logging it
	eventResp, err := h.httpClient.Get(fmt.Sprintf("%s/v1/%d", os.Getenv("EVENT_SERVICE_URL"), input.EventID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to fetch event details: %v", err)})
		return
	}
	defer eventResp.Body.Close()

	var eventDetails struct {
		Price float64 `json:"price"`
	}
	if err := json.NewDecoder(eventResp.Body).Decode(&eventDetails); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse event response"})
		return
	}

	// Fetching user details to get email to send the QR code to at the end
	userResp, err := h.httpClient.Get(fmt.Sprintf("%s/v1/%d", os.Getenv("USER_SERVICE_URL"), input.UserID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to fetch user details: %v", err)})
		return
	}
	defer userResp.Body.Close()

	var userDetails struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(userResp.Body).Decode(&userDetails); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse user response"})
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
		TicketID   int    `json:"ticket_id"`
		UserID     int    `json:"user_id"`
		TicketCode string `json:"ticket_code"`
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

	// Publish payment message
	if h.broker != nil {
		paymentMsg := struct {
			TicketID int `json:"ticket_id"`
			Amount   int `json:"amount"`
		}{
			TicketID: ticketResp.TicketID,
			Amount:   1000, // This should be fetched from the event service, for now this is fine
		}
		if err := h.broker.Publish(paymentMsg, "topay"); err != nil {
			log.Printf("Warning: Failed to publish payment message: %v", err)
		} else {
			log.Printf("Published payment message for ticket %d", ticketResp.TicketID)
		}

		notificationMsg := struct {
			RecipientEmail string `json:"recipient_email"`
			TicketID       string `json:"ticket_id"`
		}{
			RecipientEmail: "user@example.com",
			TicketID:       ticketResp.TicketCode,
		}
		if err := h.broker.Publish(notificationMsg, "email"); err != nil {
			log.Printf("Warning: Failed to publish notification message: %v", err)
		} else {
			log.Printf("Published notification message for ticket %s", ticketResp.TicketCode)
		}
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
	//os.Getenv("PAYMENT_SERVICE_URL")+
	req, err := http.NewRequest("POST", "http://payment:8088/create-payment-intent", bytes.NewBuffer(body))
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

func (h *Handler) VerifyTicket(c *gin.Context) {
	log.Println("VerifyTicket called")

	// Parse multipart form
	if err := c.Request.ParseMultipartForm(10 << 20); err != nil { // 10MB max
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse form: " + err.Error()})
		return
	}

	var qrData string
	// Check for file or URL
	file, _, err := c.Request.FormFile("file")
	if err == nil {
		defer file.Close()
		// Send QR code image to goqr.me
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("file", "qr.png")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create form file: " + err.Error()})
			return
		}
		if _, err := io.Copy(part, file); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to copy file: " + err.Error()})
			return
		}
		writer.Close()

		req, err := http.NewRequest("POST", "https://api.qrserver.com/v1/read-qr-code/", body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create QR request: " + err.Error()})
			return
		}
		req.Header.Set("Content-Type", writer.FormDataContentType())

		resp, err := h.httpClient.Do(req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "QR service error: " + err.Error()})
			return
		}
		defer resp.Body.Close()

		var qrResp []struct {
			Symbol []struct {
				Data  string `json:"data"`
				Error string `json:"error"`
			} `json:"symbol"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&qrResp); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse QR response: " + err.Error()})
			return
		}
		if len(qrResp) == 0 || len(qrResp[0].Symbol) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid QR code"})
			return
		}
		if qrResp[0].Symbol[0].Error != "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "QR code error: " + qrResp[0].Symbol[0].Error})
			return
		}
		qrData = qrResp[0].Symbol[0].Data
	} else if url := c.Request.FormValue("url"); url != "" {
		// Send QR code URL to goqr.me
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		if err := writer.WriteField("file", "@url:"+url); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to write URL field: " + err.Error()})
			return
		}
		writer.Close()

		req, err := http.NewRequest("POST", "https://api.qrserver.com/v1/read-qr-code/", body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create QR request: " + err.Error()})
			return
		}
		req.Header.Set("Content-Type", writer.FormDataContentType())

		resp, err := h.httpClient.Do(req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "QR service error: " + err.Error()})
			return
		}
		defer resp.Body.Close()

		var qrResp []struct {
			Symbol []struct {
				Data  string `json:"data"`
				Error string `json:"error"`
			} `json:"symbol"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&qrResp); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse QR response: " + err.Error()})
			return
		}
		if len(qrResp) == 0 || len(qrResp[0].Symbol) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid QR code"})
			return
		}
		if qrResp[0].Symbol[0].Error != "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "QR code error: " + qrResp[0].Symbol[0].Error})
			return
		}
		qrData = qrResp[0].Symbol[0].Data
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing file or URL"})
		return
	}

	// Extract ticket_code (assume format "ticket_code:<UUID>")
	ticketCode := qrData
	if strings.HasPrefix(qrData, "ticket_code:") {
		ticketCode = strings.TrimPrefix(qrData, "ticket_code:")
	}
	if !isValidUUID(ticketCode) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ticket code format"})
		return
	}

	// Query ticket service to verify ticket
	url := fmt.Sprintf("%s/v1/verify/%s", os.Getenv("TICKET_SERVICE_URL"), ticketCode)
	resp, err := h.httpClient.Get(url)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ticket service error: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			c.JSON(http.StatusNotFound, gin.H{"valid": false, "error": "Ticket not found or inactive"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Ticket service returned status %d", resp.StatusCode)})
		return
	}

	var ticket struct {
		TicketID int    `json:"ticket_id"`
		EventID  int    `json:"event_id"`
		UserID   int    `json:"user_id"`
		Status   string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ticket); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse ticket response: " + err.Error()})
		return
	}

	if ticket.Status != "active" {
		c.JSON(http.StatusNotFound, gin.H{"valid": false, "error": "Ticket is not active"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":     true,
		"ticket_id": ticket.TicketID,
		"event_id":  ticket.EventID,
		"user_id":   ticket.UserID,
	})
}

// isValidUUID checks if the string is a valid UUID
func isValidUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
		} else if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}
