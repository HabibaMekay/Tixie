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
	circuitbreaker "tixie.local/common"
)

type eventDetails struct {
	Price float64 `json:"price"`
}

type userDetails struct {
	Email string `json:"email"`
}

type ticketResponse struct {
	TicketID   int    `json:"ticket_id"`
	UserID     int    `json:"user_id"`
	TicketCode string `json:"ticket_code"`
}

type paymentResponse struct {
	ClientSecret   string `json:"client_secret"`
	IdempotencyKey string `json:"idempotency_key"`
}

type qrResponse struct {
	Symbol []struct {
		Data  string `json:"data"`
		Error string `json:"error"`
	} `json:"symbol"`
}

type ticketVerificationResponse struct {
	TicketID int    `json:"ticket_id"`
	EventID  int    `json:"event_id"`
	UserID   int    `json:"user_id"`
	Status   string `json:"status"`
}

type Handler struct {
	repo       *repos.PurchaseRepository
	httpClient *http.Client
	broker     *brokerPkg.Broker
	breaker    *circuitbreaker.Breaker
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
		broker:  broker,
		breaker: circuitbreaker.NewBreaker("reservation-service"),
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

	// Fetch event details with circuit breaker
	result := h.breaker.Execute(func() (interface{}, error) {
		resp, err := h.httpClient.Get(fmt.Sprintf("%s/v1/%d", os.Getenv("EVENT_SERVICE_URL"), input.EventID))
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("event service returned status %d", resp.StatusCode)
		}

		var details eventDetails
		if err := json.NewDecoder(resp.Body).Decode(&details); err != nil {
			return nil, err
		}
		return details, nil
	})

	if result.Error != nil {
		if circuitbreaker.IsCircuitBreakerError(result.Error) {
			status, msg := circuitbreaker.HandleCircuitBreakerError(result.Error)
			c.JSON(status, gin.H{"error": msg})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to fetch event details: %v", result.Error)})
		return
	}

	eventDetails, ok := result.Data.(eventDetails)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse event response"})
		return
	}

	// Fetch user details with circuit breaker
	result = h.breaker.Execute(func() (interface{}, error) {
		resp, err := h.httpClient.Get(fmt.Sprintf("%s/v1/%d", os.Getenv("USER_SERVICE_URL"), input.UserID))
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("user service returned status %d", resp.StatusCode)
		}

		var details userDetails
		if err := json.NewDecoder(resp.Body).Decode(&details); err != nil {
			return nil, err
		}
		return details, nil
	})

	if result.Error != nil {
		if circuitbreaker.IsCircuitBreakerError(result.Error) {
			status, msg := circuitbreaker.HandleCircuitBreakerError(result.Error)
			c.JSON(status, gin.H{"error": msg})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to fetch user details: %v", result.Error)})
		return
	}

	userDetails, ok := result.Data.(userDetails)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse user response"})
		return
	}

	// Create ticket with circuit breaker
	result = h.breaker.Execute(func() (interface{}, error) {
		ticketReq := struct {
			EventID int `json:"event_id"`
			UserID  int `json:"user_id"`
		}{EventID: input.EventID, UserID: input.UserID}

		ticketReqBody, err := json.Marshal(ticketReq)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal ticket request: %v", err)
		}

		resp, err := h.httpClient.Post(os.Getenv("TICKET_SERVICE_URL")+"/v1", "application/json", bytes.NewBuffer(ticketReqBody))
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			return nil, fmt.Errorf("ticket service returned status %d", resp.StatusCode)
		}

		var ticket ticketResponse
		if err := json.NewDecoder(resp.Body).Decode(&ticket); err != nil {
			return nil, err
		}
		return ticket, nil
	})

	if result.Error != nil {
		if circuitbreaker.IsCircuitBreakerError(result.Error) {
			status, msg := circuitbreaker.HandleCircuitBreakerError(result.Error)
			c.JSON(status, gin.H{"error": msg})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create ticket: %v", result.Error)})
		return
	}

	ticketResp, ok := result.Data.(ticketResponse)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse ticket response"})
		return
	}

	// Create purchase with circuit breaker
	purchase := &models.Purchase{
		TicketID:     ticketResp.TicketID,
		UserID:       input.UserID,
		EventID:      input.EventID,
		PurchaseDate: time.Now().UTC(),
		Status:       "confirmed",
	}

	result = h.breaker.Execute(func() (interface{}, error) {
		return h.repo.CreatePurchase(purchase)
	})

	if result.Error != nil {
		if circuitbreaker.IsCircuitBreakerError(result.Error) {
			status, msg := circuitbreaker.HandleCircuitBreakerError(result.Error)
			c.JSON(status, gin.H{"error": msg})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + result.Error.Error()})
		return
	}

	createdPurchase, ok := result.Data.(*models.Purchase)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create purchase"})
		return
	}

	// Publish messages with circuit breaker
	if h.broker != nil {
		result = h.breaker.Execute(func() (interface{}, error) {
			paymentMsg := struct {
				TicketID int `json:"ticket_id"`
				Amount   int `json:"amount"`
			}{
				TicketID: ticketResp.TicketID,
				Amount:   int(eventDetails.Price * 100), // Convert to cents
			}
			if err := h.broker.Publish(paymentMsg, "topay"); err != nil {
				return nil, err
			}

			notificationMsg := struct {
				RecipientEmail string `json:"recipient_email"`
				TicketID       string `json:"ticket_id"`
			}{
				RecipientEmail: userDetails.Email,
				TicketID:       ticketResp.TicketCode,
			}
			return nil, h.broker.Publish(notificationMsg, "email")
		})

		if result.Error != nil {
			log.Printf("Warning: Failed to publish messages: %v", result.Error)
		}
	}

	c.JSON(http.StatusCreated, createdPurchase)
}

func (h *Handler) handlePayment(amount int) (bool, error) {
	log.Println("Initiating payment process")

	if amount <= 0 {
		return false, fmt.Errorf("invalid amount: must be greater than zero")
	}

	result := h.breaker.Execute(func() (interface{}, error) {
		payload := map[string]interface{}{
			"amount": amount,
		}
		body, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %v", err)
		}

		req, err := http.NewRequest("POST", "http://payment:8088/create-payment-intent", bytes.NewBuffer(body))
		if err != nil {
			return nil, fmt.Errorf("failed to create payment request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := h.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("payment service error: %v", err)
		}
		defer resp.Body.Close()

		body, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			var errorMsg map[string]interface{}
			if err := json.Unmarshal(body, &errorMsg); err != nil {
				return nil, fmt.Errorf("payment service returned status %d", resp.StatusCode)
			}
			return nil, fmt.Errorf("payment service returned status %d: %v", resp.StatusCode, errorMsg)
		}

		var paymentResp paymentResponse
		if err := json.Unmarshal(body, &paymentResp); err != nil {
			return nil, fmt.Errorf("failed to parse payment response: %v", err)
		}
		if paymentResp.ClientSecret == "" || paymentResp.IdempotencyKey == "" {
			return nil, fmt.Errorf("missing fields in payment response")
		}

		return paymentResp, nil
	})

	if result.Error != nil {
		if circuitbreaker.IsCircuitBreakerError(result.Error) {
			return false, result.Error
		}
		return false, fmt.Errorf("payment failed: %v", result.Error)
	}

	paymentResp, ok := result.Data.(paymentResponse)
	if !ok {
		return false, fmt.Errorf("invalid payment response type")
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

		result := h.breaker.Execute(func() (interface{}, error) {
			// Send QR code image to goqr.me
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			part, err := writer.CreateFormFile("file", "qr.png")
			if err != nil {
				return nil, fmt.Errorf("failed to create form file: %v", err)
			}
			if _, err := io.Copy(part, file); err != nil {
				return nil, fmt.Errorf("failed to copy file: %v", err)
			}
			writer.Close()

			req, err := http.NewRequest("POST", "https://api.qrserver.com/v1/read-qr-code/", body)
			if err != nil {
				return nil, fmt.Errorf("failed to create QR request: %v", err)
			}
			req.Header.Set("Content-Type", writer.FormDataContentType())

			resp, err := h.httpClient.Do(req)
			if err != nil {
				return nil, fmt.Errorf("QR service error: %v", err)
			}
			defer resp.Body.Close()

			var qrResp []qrResponse
			if err := json.NewDecoder(resp.Body).Decode(&qrResp); err != nil {
				return nil, fmt.Errorf("failed to parse QR response: %v", err)
			}
			if len(qrResp) == 0 || len(qrResp[0].Symbol) == 0 {
				return nil, fmt.Errorf("invalid QR code")
			}
			if qrResp[0].Symbol[0].Error != "" {
				return nil, fmt.Errorf("QR code error: %v", qrResp[0].Symbol[0].Error)
			}
			return qrResp[0].Symbol[0].Data, nil
		})

		if result.Error != nil {
			if circuitbreaker.IsCircuitBreakerError(result.Error) {
				status, msg := circuitbreaker.HandleCircuitBreakerError(result.Error)
				c.JSON(status, gin.H{"error": msg})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
			return
		}

		var ok bool
		qrData, ok = result.Data.(string)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid QR response type"})
			return
		}
	} else if url := c.Request.FormValue("url"); url != "" {
		result := h.breaker.Execute(func() (interface{}, error) {
			// Send QR code URL to goqr.me
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			if err := writer.WriteField("file", "@url:"+url); err != nil {
				return nil, fmt.Errorf("failed to write URL field: %v", err)
			}
			writer.Close()

			req, err := http.NewRequest("POST", "https://api.qrserver.com/v1/read-qr-code/", body)
			if err != nil {
				return nil, fmt.Errorf("failed to create QR request: %v", err)
			}
			req.Header.Set("Content-Type", writer.FormDataContentType())

			resp, err := h.httpClient.Do(req)
			if err != nil {
				return nil, fmt.Errorf("QR service error: %v", err)
			}
			defer resp.Body.Close()

			var qrResp []qrResponse
			if err := json.NewDecoder(resp.Body).Decode(&qrResp); err != nil {
				return nil, fmt.Errorf("failed to parse QR response: %v", err)
			}
			if len(qrResp) == 0 || len(qrResp[0].Symbol) == 0 {
				return nil, fmt.Errorf("invalid QR code")
			}
			if qrResp[0].Symbol[0].Error != "" {
				return nil, fmt.Errorf("QR code error: %v", qrResp[0].Symbol[0].Error)
			}
			return qrResp[0].Symbol[0].Data, nil
		})

		if result.Error != nil {
			if circuitbreaker.IsCircuitBreakerError(result.Error) {
				status, msg := circuitbreaker.HandleCircuitBreakerError(result.Error)
				c.JSON(status, gin.H{"error": msg})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
			return
		}

		var ok bool
		qrData, ok = result.Data.(string)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid QR response type"})
			return
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file or URL provided"})
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

	// Query ticket service to verify ticket with circuit breaker
	var result circuitbreaker.Result
	result = h.breaker.Execute(func() (interface{}, error) {
		url := fmt.Sprintf("%s/v1/verify/%s", os.Getenv("TICKET_SERVICE_URL"), ticketCode)
		resp, err := h.httpClient.Get(url)
		if err != nil {
			return nil, fmt.Errorf("ticket service error: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			if resp.StatusCode == http.StatusNotFound {
				return nil, fmt.Errorf("ticket not found or inactive")
			}
			return nil, fmt.Errorf("ticket service returned status %d", resp.StatusCode)
		}

		var ticket ticketVerificationResponse
		if err := json.NewDecoder(resp.Body).Decode(&ticket); err != nil {
			return nil, fmt.Errorf("failed to parse ticket response: %v", err)
		}
		return ticket, nil
	})

	if result.Error != nil {
		if circuitbreaker.IsCircuitBreakerError(result.Error) {
			status, msg := circuitbreaker.HandleCircuitBreakerError(result.Error)
			c.JSON(status, gin.H{"error": msg})
			return
		}
		if strings.Contains(result.Error.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"valid": false, "error": result.Error.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	ticket, ok := result.Data.(ticketVerificationResponse)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid ticket response type"})
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
