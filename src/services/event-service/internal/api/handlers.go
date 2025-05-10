package api

import (
	"event-service/internal/db/models"
	"event-service/internal/db/repos"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"log"
	"os"
)

var logger *log.Logger

func init() {
	if _, err := os.Stat("logs"); os.IsNotExist(err) {
		os.MkdirAll("logs", os.ModePerm)
	}
	logFile, err := os.OpenFile("logs/service.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("failed to open log file: %v", err)
	}

	logger = log.New(logFile, "EVENT-SERVICE: ", log.LstdFlags|log.Lshortfile)
}

type EventHandler struct {
	Repo *repos.EventRepository
}

func NewEventHandler(repo *repos.EventRepository) *EventHandler {
	return &EventHandler{Repo: repo}
}

func (h *EventHandler) GetEvents(c *gin.Context) {
	events, err := h.Repo.GetAllEvents()
	if err != nil {
		logger.Printf("error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, events)
}

func (h *EventHandler) CreateEvent(c *gin.Context) {
	var event models.Event
	if err := c.ShouldBindJSON(&event); err != nil {
		logger.Printf("error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	if event.ReservationTimeout <= 0 {
		event.ReservationTimeout = 600
	}

	if err := h.Repo.CreateEvent(event); err != nil {
		logger.Printf("error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create event"})
		return
	}

	logger.Println("Event successful")
	c.JSON(http.StatusCreated, event)
}

func (h *EventHandler) GetEventByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	event, err := h.Repo.GetEventByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}

	c.JSON(http.StatusOK, event)
}

func (h *EventHandler) UpdateTicketsSold(c *gin.Context) {
	eventID := c.Param("id")
	var input struct {
		TicketsToBuy int `json:"tickets_to_buy"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	if input.TicketsToBuy <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no tickets left"})
		return
	}

	err := h.Repo.UpdateTicketsSold(eventID, input.TicketsToBuy)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Tickets sold updated successfully"})
}

// ReserveTicket temporarily reserves a ticket for an event
func (h *EventHandler) ReserveTicket(c *gin.Context) {
	// Get event ID from URL path
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		logger.Printf("Invalid event ID format: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID"})
		return
	}

	// Call repository to reserve a ticket
	err = h.Repo.ReserveTicket(id)
	if err != nil {
		logger.Printf("Failed to reserve ticket for event %d: %v", id, err)
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	logger.Printf("Successfully reserved ticket for event %d", id)
	c.JSON(http.StatusOK, gin.H{
		"message":  "Ticket reserved successfully",
		"event_id": id,
	})
}

// CompleteReservation confirms a reservation and converts it to a sold ticket
func (h *EventHandler) CompleteReservation(c *gin.Context) {
	// Get event ID from URL path
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		logger.Printf("Invalid event ID format: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID"})
		return
	}

	// Call repository to complete the reservation
	err = h.Repo.CompleteReservation(id)
	if err != nil {
		logger.Printf("Failed to complete reservation for event %d: %v", id, err)
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	logger.Printf("Successfully completed reservation for event %d", id)
	c.JSON(http.StatusOK, gin.H{
		"message":  "Reservation completed successfully",
		"event_id": id,
	})
}

// ReleaseReservation releases a reserved ticket back to the available pool
func (h *EventHandler) ReleaseReservation(c *gin.Context) {
	// Get event ID from URL path
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		logger.Printf("Invalid event ID format: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID"})
		return
	}

	// Call repository to release the reservation
	err = h.Repo.ReleaseReservation(id)
	if err != nil {
		logger.Printf("Failed to release reservation for event %d: %v", id, err)
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	logger.Printf("Successfully released reservation for event %d", id)
	c.JSON(http.StatusOK, gin.H{
		"message":  "Reservation released successfully",
		"event_id": id,
	})
}
