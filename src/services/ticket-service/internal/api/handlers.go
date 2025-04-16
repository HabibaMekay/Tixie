package api

import (
	"log"
	"net/http"
	"strconv"
	"ticket-service/internal/db/models"
	"ticket-service/internal/db/repos"

	"github.com/gin-gonic/gin"
)

// Handler holds dependencies for API handlers.
type Handler struct {
	repo *repos.TicketRepository
}

// NewHandler creates a new Handler with dependencies.
func NewHandler(repo *repos.TicketRepository) *Handler {
	return &Handler{repo: repo}
}

// GetTicketByID retrieves a ticket by its ID.
func (h *Handler) GetTicketByID(c *gin.Context) {
	log.Println("GetTicketByID called")
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ticket ID"})
		return
	}

	ticket, err := h.repo.GetTicketByID(ticketID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if ticket == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ticket not found"})
		return
	}

	c.JSON(http.StatusOK, ticket)
}

// // GetTicketsByEventID retrieves tickets for a given event_id.
// func (h *Handler) GetTicketsByEventID(c *gin.Context) {
// 	eventID, err := strconv.Atoi(c.Query("event_id"))
// 	if err != nil || eventID <= 0 {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID"})
// 		return
// 	}

// 	tickets, err := h.repo.GetTicketsByEventID(eventID)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 		return
// 	}

// 	c.JSON(http.StatusOK, tickets)
// }

// CreateTicket creates a new ticket.
func (h *Handler) CreateTicket(c *gin.Context) {
	var ticket models.Ticket
	if err := c.ShouldBindJSON(&ticket); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Basic validation for event_id (since there's no events table or event-service)
	if ticket.EventID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID"})
		return
	}

	createdTicket, err := h.repo.CreateTicket(&ticket)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, createdTicket)
}

// UpdateTicketStatus updates the status of a ticket.
func (h *Handler) UpdateTicketStatus(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ticket ID"})
		return
	}

	var input struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Basic validation for status
	if input.Status != "available" && input.Status != "sold" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status"})
		return
	}

	ticket, err := h.repo.GetTicketByID(ticketID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if ticket == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ticket not found"})
		return
	}

	updatedTicket, err := h.repo.UpdateTicketStatus(ticketID, input.Status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, updatedTicket)
}
