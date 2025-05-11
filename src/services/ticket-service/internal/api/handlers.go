package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"ticket-service/internal/db/models"
	"ticket-service/internal/db/repos"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	brokerPkg "tixie.local/broker"
	circuitbreaker "tixie.local/common"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type ClientConnection struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

type Handler struct {
	repo       *repos.TicketRepository
	httpClient *http.Client
	breaker    *circuitbreaker.Breaker
	broker     *brokerPkg.Broker
}

func NewHandler(repo *repos.TicketRepository, broker *brokerPkg.Broker) *Handler {
	return &Handler{
		repo: repo,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		breaker: circuitbreaker.NewBreaker("ticket-service"),
		broker:  broker,
	}
}

func (h *Handler) GetTicketByID(c *gin.Context) {
	log.Println("GetTicketByID called")
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ticket ID"})
		return
	}

	result := h.breaker.Execute(func() (interface{}, error) {
		return h.repo.GetTicketByID(ticketID)
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

	ticket, ok := result.Data.(*models.Ticket)
	if !ok || ticket == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ticket not found"})
		return
	}

	c.JSON(http.StatusOK, ticket)
}

func (h *Handler) GetTicketsByEventID(c *gin.Context) {
	log.Println("GetTicketsByEventID called")
	eventID, err := strconv.Atoi(c.Query("event_id"))
	if err != nil || eventID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID"})
		return
	}

	result := h.breaker.Execute(func() (interface{}, error) {
		return h.repo.GetTicketsByEventID(eventID)
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

	tickets, ok := result.Data.([]models.Ticket)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	if len(tickets) == 0 {
		c.JSON(http.StatusOK, gin.H{"message": "No tickets found for event", "tickets": []models.Ticket{}})
		return
	}

	c.JSON(http.StatusOK, tickets)
}

func (h *Handler) CreateTicket(c *gin.Context) {
	log.Println("CreateTicket called")
	var input struct {
		EventID int `json:"event_id" binding:"required,gt=0"`
		UserID  int `json:"user_id" binding:"required,gt=0"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	if err := h.validateEvent(input.EventID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid event_id: %v", err)})
		return
	}

	if err := h.validateUser(input.UserID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid user_id: %v", err)})
		return
	}

	ticketCode := uuid.New().String()

	ticket := &models.Ticket{
		EventID:    input.EventID,
		UserID:     input.UserID,
		TicketCode: ticketCode,
		Status:     "active",
	}

	result := h.breaker.Execute(func() (interface{}, error) {
		return h.repo.CreateTicket(ticket)
	})

	if result.Error != nil {
		if circuitbreaker.IsCircuitBreakerError(result.Error) {
			status, msg := circuitbreaker.HandleCircuitBreakerError(result.Error)
			c.JSON(status, gin.H{"error": msg})
			return
		}
		if strings.Contains(result.Error.Error(), "duplicate key") {
			c.JSON(http.StatusConflict, gin.H{"error": "Ticket code already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + result.Error.Error()})
		return
	}

	createdTicket, ok := result.Data.(*models.Ticket)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	log.Printf("Fetching email for userID: %d", input.UserID)
	userServiceURL := os.Getenv("USER_SERVICE_URL")
	userEmail := ""
	if userServiceURL != "" {
		resp, err := h.httpClient.Get(fmt.Sprintf("%s/v1/email/%d", userServiceURL, input.UserID))
		if err == nil && resp.StatusCode == http.StatusOK {
			var emailResp struct {
				Email string `json:"email"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&emailResp); err == nil {
				userEmail = emailResp.Email
			}
			resp.Body.Close()
		} else {
			log.Printf("Failed to fetch user email for ticket issued message: %v", err)
		}
	}

	emailMsg := struct {
		RecipientEmail string `json:"recipient_email"`
		TicketID       string `json:"ticket_id"`
	}{
		RecipientEmail: userEmail,
		TicketID:       createdTicket.TicketCode,
	}

	if h.broker != nil {
		if err := h.broker.Publish(emailMsg, "email"); err != nil {
			log.Printf("Failed to publish email notification: %v", err)
		}
	}

	c.JSON(http.StatusCreated, createdTicket)
}

func (h *Handler) UpdateTicketStatus(c *gin.Context) {
	log.Println("UpdateTicketStatus called")
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ticket ID"})
		return
	}

	var input struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	if input.Status != "active" && input.Status != "used" && input.Status != "cancelled" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status: must be active, used, or cancelled"})
		return
	}

	// Check if ticket exists
	result := h.breaker.Execute(func() (interface{}, error) {
		return h.repo.GetTicketByID(ticketID)
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

	ticket, ok := result.Data.(*models.Ticket)
	if !ok || ticket == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ticket not found"})
		return
	}

	// Update ticket status
	result = h.breaker.Execute(func() (interface{}, error) {
		return h.repo.UpdateTicketStatus(ticketID, input.Status)
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

	updatedTicket, ok := result.Data.(*models.Ticket)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, updatedTicket)
}

// func (h *Handler) GetTicketByCode(c *gin.Context) {
// 	log.Println("GetTicketByCode called")
// 	ticketCode := c.Param("ticket_code")

// 	ticket, err := h.repo.GetTicketByCode(ticketCode)
// 	if err != nil {
// 		c.JSON(http.StatusNotFound, gin.H{"error": "Ticket not found"})
// 		return
// 	}

//		c.JSON(http.StatusOK, gin.H{
//			"ticket_id": ticket.TicketID,
//			"event_id":  ticket.EventID,
//			"user_id":   ticket.UserID,
//			"status":    ticket.Status,
//		})
//	}
func (h *Handler) GetTicketByCode(c *gin.Context) {
	log.Println("GetTicketByCode called")
	ticketCode := c.Param("ticket_code")
	log.Printf("Raw ticketCode: %q", ticketCode)

	ticketCode = strings.TrimSpace(strings.ToLower(ticketCode))
	log.Printf("Normalized ticketCode: %s", ticketCode)

	result := h.breaker.Execute(func() (interface{}, error) {
		return h.repo.GetTicketByCode(ticketCode)
	})

	if result.Error != nil {
		if circuitbreaker.IsCircuitBreakerError(result.Error) {
			status, msg := circuitbreaker.HandleCircuitBreakerError(result.Error)
			c.JSON(status, gin.H{"error": msg})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "Ticket not found"})
		return
	}

	ticket, ok := result.Data.(*models.Ticket)
	if !ok || ticket == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ticket not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ticket_id": ticket.TicketID,
		"event_id":  ticket.EventID,
		"user_id":   ticket.UserID,
		"status":    ticket.Status,
	})
}

func (h *Handler) validateEvent(eventID int) error {
	url := fmt.Sprintf("http://event-service-1:8080/v1/%d", eventID)
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

func (h *Handler) validateUser(userID int) error {
	url := fmt.Sprintf("http://user-service-1:8081/v1/%d", userID)
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

// GetEventsWithTickets handles the request to get all events that have tickets
func (h *Handler) GetEventsWithTickets(c *gin.Context) {
	log.Println("GetEventsWithTickets called")

	result := h.breaker.Execute(func() (interface{}, error) {
		return h.repo.GetEventsWithTickets()
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

	events, ok := result.Data.([]repos.EventWithTickets)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	if len(events) == 0 {
		c.JSON(http.StatusOK, gin.H{"message": "No events found with tickets", "events": []repos.EventWithTickets{}})
		return
	}

	// For each event, get its name from the event service
	type EventResponse struct {
		EventID     int    `json:"event_id"`
		EventName   string `json:"event_name"`
		TicketCount int    `json:"ticket_count"`
	}

	var enrichedEvents []EventResponse
	for _, event := range events {
		// Call event service to get event name
		url := fmt.Sprintf("http://event-service-1:8080/v1/%d", event.EventID)
		resp, err := h.httpClient.Get(url)

		if err != nil || resp.StatusCode != http.StatusOK {
			// If we can't get the event name, still include the event but with empty name
			enrichedEvents = append(enrichedEvents, EventResponse{
				EventID:     event.EventID,
				EventName:   "",
				TicketCount: event.TicketCount,
			})
			continue
		}
		defer resp.Body.Close()

		// Parse event response
		var eventDetails struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&eventDetails); err != nil {
			enrichedEvents = append(enrichedEvents, EventResponse{
				EventID:     event.EventID,
				EventName:   "",
				TicketCount: event.TicketCount,
			})
			continue
		}

		enrichedEvents = append(enrichedEvents, EventResponse{
			EventID:     event.EventID,
			EventName:   eventDetails.Name,
			TicketCount: event.TicketCount,
		})
	}

	c.JSON(http.StatusOK, gin.H{"events": enrichedEvents})
}

// GetEventsWithTicketsWS handles WebSocket connections for real-time event ticket updates
func (h *Handler) GetEventsWithTicketsWS(c *gin.Context) {
	log.Println("GetEventsWithTicketsWS called")

	// Upgrade HTTP connection to WebSocket
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}
	defer ws.Close()

	client := &ClientConnection{conn: ws, mu: sync.Mutex{}}

	// Send initial data
	if err := h.sendEventData(client); err != nil {
		log.Printf("Failed to send initial data: %v", err)
		return
	}

	// Start ticker for periodic updates
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	// Keep connection alive and send updates
	for {
		select {
		case <-ticker.C:
			if err := h.sendEventData(client); err != nil {
				log.Printf("Failed to send update: %v", err)
				return
			}
		}
	}
}

func (h *Handler) sendEventData(client *ClientConnection) error {
	result := h.breaker.Execute(func() (interface{}, error) {
		return h.repo.GetEventsWithTickets()
	})

	if result.Error != nil {
		return fmt.Errorf("failed to get events: %v", result.Error)
	}

	events, ok := result.Data.([]repos.EventWithTickets)
	if !ok {
		return fmt.Errorf("invalid response type from repository")
	}

	client.mu.Lock()
	defer client.mu.Unlock()

	if err := client.conn.WriteJSON(gin.H{"events": events}); err != nil {
		return fmt.Errorf("failed to write to websocket: %v", err)
	}

	return nil
}

// GetTicketsByEventIDWS handles WebSocket connections for real-time ticket updates for a specific event
func (h *Handler) GetTicketsByEventIDWS(c *gin.Context) {
	log.Println("GetTicketsByEventIDWS called")

	eventID, err := strconv.Atoi(c.Param("event_id"))
	if err != nil || eventID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID"})
		return
	}

	// Upgrade HTTP connection to WebSocket
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}
	defer ws.Close()
	client := &ClientConnection{conn: ws, mu: sync.Mutex{}}
	if err := h.sendTicketData(client, eventID); err != nil {
		log.Printf("Failed to send initial data: %v", err)
		return
	}
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := h.sendTicketData(client, eventID); err != nil {
				log.Printf("Failed to send update: %v", err)
				return
			}
		}
	}
}

func (h *Handler) sendTicketData(client *ClientConnection, eventID int) error {
	result := h.breaker.Execute(func() (interface{}, error) {
		return h.repo.GetTicketsByEventID(eventID)
	})

	if result.Error != nil {
		return fmt.Errorf("failed to get tickets: %v", result.Error)
	}

	tickets, ok := result.Data.([]models.Ticket)
	if !ok {
		return fmt.Errorf("invalid response type from repository")
	}

	client.mu.Lock()
	defer client.mu.Unlock()

	return client.conn.WriteJSON(gin.H{
		"event_id": eventID,
		"tickets":  tickets,
	})
}
