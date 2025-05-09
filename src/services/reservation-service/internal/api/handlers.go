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
	brokermsg "tixie.local/common/brokermsg"
	"tixie.local/common/circuitbreaker"
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
	purchaseRepo *repos.PurchaseRepository
	reserveRepo  *repos.ReservationRepository
	httpClient   *http.Client
	broker       *brokerPkg.Broker
	breaker      *circuitbreaker.CircuitBreaker
}

func NewHandler(purchaseRepo *repos.PurchaseRepository, reserveRepo *repos.ReservationRepository) *Handler {
	broker, err := brokerPkg.NewBroker(os.Getenv("RABBITMQ_URL"), "payment", "topic")
	if err != nil {
		log.Printf("Warning: Failed to create broker: %v", err)
	}

	return &Handler{
		purchaseRepo: purchaseRepo,
		reserveRepo:  reserveRepo,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		broker:  broker,
		breaker: circuitbreaker.NewCircuitBreaker(circuitbreaker.DefaultSettings("reservation-service")),
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

	// Step 1: Get event details (including reservation timeout)
	result := h.breaker.Execute(func() (interface{}, error) {
		resp, err := h.httpClient.Get(fmt.Sprintf("%s/v1/%d", os.Getenv("EVENT_SERVICE_URL"), input.EventID))
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("event service returned status %d", resp.StatusCode)
		}

		var event struct {
			ID                 int     `json:"id"`
			Price              float64 `json:"price"`
			TicketsLeft        int     `json:"tickets_left"`
			ReservationTimeout int     `json:"reservation_timeout"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&event); err != nil {
			return nil, err
		}
		return event, nil
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

	eventDetails, ok := result.Data.(struct {
		ID                 int     `json:"id"`
		Price              float64 `json:"price"`
		TicketsLeft        int     `json:"tickets_left"`
		ReservationTimeout int     `json:"reservation_timeout"`
	})
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse event response"})
		return
	}

	// Check if tickets are available
	if eventDetails.TicketsLeft <= 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "No tickets available for this event"})
		return
	}

	// Step 2: Reserve the ticket in the event service
	result = h.breaker.Execute(func() (interface{}, error) {
		reserveReq := struct {
			EventID int `json:"event_id"`
		}{EventID: input.EventID}

		reserveReqBody, err := json.Marshal(reserveReq)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal reservation request: %v", err)
		}

		// Make a POST request to reserve the ticket (increment the reserved count)
		resp, err := h.httpClient.Post(
			fmt.Sprintf("%s/v1/%d/reserve", os.Getenv("EVENT_SERVICE_URL"), input.EventID),
			"application/json",
			bytes.NewBuffer(reserveReqBody),
		)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			// Read the error message
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("event service returned status %d: %s", resp.StatusCode, body)
		}

		// Successfully reserved a ticket in the event service
		return true, nil
	})

	if result.Error != nil {
		if circuitbreaker.IsCircuitBreakerError(result.Error) {
			status, msg := circuitbreaker.HandleCircuitBreakerError(result.Error)
			c.JSON(status, gin.H{"error": msg})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to reserve ticket: %v", result.Error)})
		return
	}

	// Step 3: Get user details to verify user exists
	result = h.breaker.Execute(func() (interface{}, error) {
		resp, err := h.httpClient.Get(fmt.Sprintf("%s/v1/%d", os.Getenv("USER_SERVICE_URL"), input.UserID))
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("user service returned status %d", resp.StatusCode)
		}

		// Just checking if user exists
		return true, nil
	})

	if result.Error != nil {
		// If we can't get user details, we should release the reservation
		h.releaseReservation(input.EventID)

		if circuitbreaker.IsCircuitBreakerError(result.Error) {
			status, msg := circuitbreaker.HandleCircuitBreakerError(result.Error)
			c.JSON(status, gin.H{"error": msg})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to fetch user details: %v", result.Error)})
		return
	}

	// Step 4: Create a reservation record in our database
	result = h.breaker.Execute(func() (interface{}, error) {
		reservation, err := h.reserveRepo.CreateReservation(
			input.EventID,
			input.UserID,
			eventDetails.ReservationTimeout,
		)
		if err != nil {
			return nil, err
		}
		return reservation, nil
	})

	if result.Error != nil {
		// If we can't create the reservation record, release the hold on the ticket
		h.releaseReservation(input.EventID)

		if circuitbreaker.IsCircuitBreakerError(result.Error) {
			status, msg := circuitbreaker.HandleCircuitBreakerError(result.Error)
			c.JSON(status, gin.H{"error": msg})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + result.Error.Error()})
		return
	}

	reservation, ok := result.Data.(*models.Reservation)
	if !ok {
		h.releaseReservation(input.EventID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create reservation"})
		return
	}

	// Schedule automatic expiration of reservation
	go h.scheduleReservationExpiration(reservation.ID, input.EventID, reservation.ExpirationTime)

	// Step 5: Publish reservation created event for analytics and audit
	if h.broker != nil {
		// Use common message structure for the reservation created event
		createdMsg := brokermsg.ReservationCreatedMessage{
			ReservationID:  reservation.ID,
			EventID:        input.EventID,
			UserID:         input.UserID,
			ExpirationTime: reservation.ExpirationTime.Unix(),
		}

		// Publish to reservation.created topic
		err := h.broker.Publish(createdMsg, brokermsg.TopicReservationCreated)
		if err != nil {
			// Just log the error but don't fail the request
			log.Printf("Warning: Failed to publish reservation created message: %v", err)
		}
	}

	// Step 6: Return reservation info to the client
	response := struct {
		ReservationID  int       `json:"reservation_id"`
		EventID        int       `json:"event_id"`
		Price          float64   `json:"price"`
		UserID         int       `json:"user_id"`
		Status         string    `json:"status"`
		ExpirationTime time.Time `json:"expiration_time"`
	}{
		ReservationID:  reservation.ID,
		EventID:        input.EventID,
		Price:          eventDetails.Price,
		UserID:         input.UserID,
		Status:         reservation.Status,
		ExpirationTime: reservation.ExpirationTime,
	}

	c.JSON(http.StatusCreated, response)
}

// CompleteReservation converts a reservation to a purchase
func (h *Handler) CompleteReservation(c *gin.Context) {
	log.Println("CompleteReservation called")
	var input struct {
		ReservationID int `json:"reservation_id" binding:"required,gt=0"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	// Step 1: Get the reservation to verify it exists and is still pending
	result := h.breaker.Execute(func() (interface{}, error) {
		reservation, err := h.reserveRepo.GetReservation(input.ReservationID)
		if err != nil {
			return nil, err
		}
		return reservation, nil
	})

	if result.Error != nil {
		if circuitbreaker.IsCircuitBreakerError(result.Error) {
			status, msg := circuitbreaker.HandleCircuitBreakerError(result.Error)
			c.JSON(status, gin.H{"error": msg})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "Reservation not found"})
		return
	}

	reservation, ok := result.Data.(*models.Reservation)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get reservation"})
		return
	}

	// Check if the reservation is still valid
	if reservation.Status != "pending" {
		c.JSON(http.StatusConflict, gin.H{"error": "Reservation is no longer valid"})
		return
	}

	if reservation.ExpirationTime.Before(time.Now().UTC()) {
		c.JSON(http.StatusConflict, gin.H{"error": "Reservation has expired"})
		return
	}

	// Step 2: Convert reservation to a completed ticket in the event service
	result = h.breaker.Execute(func() (interface{}, error) {
		resp, err := h.httpClient.Post(
			fmt.Sprintf("%s/v1/%d/complete-reservation", os.Getenv("EVENT_SERVICE_URL"), reservation.EventID),
			"application/json",
			nil,
		)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("event service returned status %d: %s", resp.StatusCode, body)
		}
		return true, nil
	})

	if result.Error != nil {
		if circuitbreaker.IsCircuitBreakerError(result.Error) {
			status, msg := circuitbreaker.HandleCircuitBreakerError(result.Error)
			c.JSON(status, gin.H{"error": msg})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to complete reservation: %v", result.Error)})
		return
	}

	// Step 3: Get event details to get the price
	result = h.breaker.Execute(func() (interface{}, error) {
		resp, err := h.httpClient.Get(fmt.Sprintf("%s/v1/%d", os.Getenv("EVENT_SERVICE_URL"), reservation.EventID))
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("event service returned status %d", resp.StatusCode)
		}

		var event struct {
			Price float64 `json:"price"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&event); err != nil {
			return nil, err
		}
		return event, nil
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

	eventDetails, ok := result.Data.(struct {
		Price float64 `json:"price"`
	})
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse event response"})
		return
	}

	// Step 4: Mark the reservation as completed in our database
	err := h.reserveRepo.CompleteReservation(input.ReservationID)
	if err != nil {
		log.Printf("Warning: Failed to mark reservation as completed: %v", err)
		// Continue anyway, as this is not critical for the user experience
	}

	// Step 5: Publish reservation completed message for payment processing
	if h.broker != nil {
		// Create reservation completed message using common message structure
		completionMsg := brokermsg.ReservationCompletedMessage{
			ReservationID: input.ReservationID,
			EventID:       reservation.EventID,
			UserID:        reservation.UserID,
			Amount:        int(eventDetails.Price * 100), // Convert to cents
		}

		err := h.broker.Publish(completionMsg, brokermsg.TopicReservationCompleted)
		if err != nil {
			log.Printf("Warning: Failed to publish reservation completion message: %v", err)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "Reservation completed successfully",
		"reservation_id": input.ReservationID,
		"event_id":       reservation.EventID,
		"user_id":        reservation.UserID,
		"amount":         int(eventDetails.Price * 100), // Return amount in cents
	})
}

// Helper method to release a reservation if something fails
func (h *Handler) releaseReservation(eventID int) {
	// Call the event service to release the reservation
	h.breaker.Execute(func() (interface{}, error) {
		resp, err := h.httpClient.Post(
			fmt.Sprintf("%s/v1/%d/release-reservation", os.Getenv("EVENT_SERVICE_URL"), eventID),
			"application/json",
			nil,
		)
		if err != nil {
			log.Printf("Error releasing reservation: %v", err)
			return nil, err
		}
		defer resp.Body.Close()
		return nil, nil
	})
}

// Method to schedule automatic expiration of reservations
func (h *Handler) scheduleReservationExpiration(reservationID int, eventID int, expirationTime time.Time) {
	// Calculate time until expiration
	duration := expirationTime.Sub(time.Now().UTC())
	if duration < 0 {
		// Already expired
		return
	}

	log.Printf("Scheduling expiration for reservation %d in %.2f seconds", reservationID, duration.Seconds())

	// Create a timer to expire the reservation
	time.AfterFunc(duration, func() {
		// Check if the reservation is still pending
		reservation, err := h.reserveRepo.GetReservation(reservationID)
		if err != nil {
			log.Printf("Error getting reservation for expiration: %v", err)
			return
		}

		// Only expire it if it's still pending
		if reservation.Status == "pending" {
			log.Printf("Expiring reservation %d for event %d", reservationID, eventID)

			// Release the ticket in the event service
			h.releaseReservation(eventID)

			// Mark the reservation as expired
			err = h.reserveRepo.ExpireReservation(reservationID)
			if err != nil {
				log.Printf("Error marking reservation as expired: %v", err)
			}
		}
	})
}

// Cleanup job to find and release any expired reservations
func (h *Handler) CleanupExpiredReservations(c *gin.Context) {
	// Get all expired reservations
	result := h.breaker.Execute(func() (interface{}, error) {
		return h.reserveRepo.GetExpiredReservations()
	})

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get expired reservations"})
		return
	}

	reservations, ok := result.Data.([]*models.Reservation)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse reservations"})
		return
	}

	log.Printf("Found %d expired reservations to clean up", len(reservations))

	// Process each expired reservation
	processed := 0
	for _, reservation := range reservations {
		// Release the ticket in the event service
		h.releaseReservation(reservation.EventID)

		// Mark the reservation as expired
		err := h.reserveRepo.ExpireReservation(reservation.ID)
		if err != nil {
			log.Printf("Error marking reservation %d as expired: %v", reservation.ID, err)
			continue
		}
		processed++
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Cleanup completed",
		"processed": processed,
		"total":     len(reservations),
	})
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
