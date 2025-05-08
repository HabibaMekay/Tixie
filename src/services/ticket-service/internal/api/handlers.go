package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"ticket-service/internal/db/models"
	"ticket-service/internal/db/repos"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
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
}

// NewHandler creates a new Handler with dependencies.
func NewHandler(repo *repos.TicketRepository) *Handler {
	return &Handler{
		repo: repo,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (h *Handler) GetTicketByID(c *gin.Context) {
	log.Println("GetTicketByID called")
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ticket ID"})
		return
	}

	ticket, err := h.repo.GetTicketByID(ticketID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + err.Error()})
		return
	}
	if ticket == nil {
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

	tickets, err := h.repo.GetTicketsByEventID(eventID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + err.Error()})
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

	createdTicket, err := h.repo.CreateTicket(ticket)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			c.JSON(http.StatusConflict, gin.H{"error": "Ticket code already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + err.Error()})
		return
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

	ticket, err := h.repo.GetTicketByID(ticketID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + err.Error()})
		return
	}
	if ticket == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ticket not found"})
		return
	}

	updatedTicket, err := h.repo.UpdateTicketStatus(ticketID, input.Status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + err.Error()})
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
	log.Printf("Raw ticketCode: %q", ticketCode) // Debug the raw ticketCode

	// Normalize ticketCode: trim whitespace and convert to lowercase
	ticketCode = strings.TrimSpace(strings.ToLower(ticketCode))

	log.Printf("Normalized ticketCode: %s", ticketCode)

	ticket, err := h.repo.GetTicketByCode(ticketCode)
	if err != nil {
		log.Printf("Database error for ticketCode %s: %v", ticketCode, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Tiiiicket not found"})
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
	//url := fmt.Sprintf("%s/v1/%d", os.Getenv("EVENT_SERVICE_1"), eventID)
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
	//url := fmt.Sprintf("%s/v1/%d", os.Getenv("USER_SERVICE_1"), userID)
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

	events, err := h.repo.GetEventsWithTickets()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get events: " + err.Error()})
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
	events, err := h.repo.GetEventsWithTickets()
	if err != nil {
		return fmt.Errorf("failed to get events: %v", err)
	}

	type EventResponse struct {
		EventID     int    `json:"event_id"`
		EventName   string `json:"event_name"`
		TicketCount int    `json:"ticket_count"`
	}

	var enrichedEvents []EventResponse
	for _, event := range events {
		url := fmt.Sprintf("http://event-service-1:8080/v1/%d", event.EventID)
		resp, err := h.httpClient.Get(url)

		eventResponse := EventResponse{
			EventID:     event.EventID,
			EventName:   "",
			TicketCount: event.TicketCount,
		}

		if err == nil && resp.StatusCode == http.StatusOK {
			var eventDetails struct {
				Name string `json:"name"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&eventDetails); err == nil {
				eventResponse.EventName = eventDetails.Name
			}
			resp.Body.Close()
		}

		enrichedEvents = append(enrichedEvents, eventResponse)
	}

	response := gin.H{"events": enrichedEvents}

	client.mu.Lock()
	defer client.mu.Unlock()

	return client.conn.WriteJSON(response)
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
	tickets, err := h.repo.GetTicketsByEventID(eventID)
	if err != nil {
		return fmt.Errorf("failed to get tickets: %v", err)
	}

	client.mu.Lock()
	defer client.mu.Unlock()

	return client.conn.WriteJSON(gin.H{
		"event_id": eventID,
		"tickets":  tickets,
	})
}
