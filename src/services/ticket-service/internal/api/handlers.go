package api

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"ticket-service/internal/db/models"
	"ticket-service/internal/db/repos"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

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

	// if err := h.validateEvent(input.EventID); err != nil {
	// 	c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid event_id: %v", err)})
	// 	return
	// }

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

func (h *Handler) validateEvent(eventID int) error {
	url := fmt.Sprintf("%s/v1/%d", os.Getenv("EVENT_SERVICE_URL"), eventID)
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
	url := fmt.Sprintf("%s/v1/%d", os.Getenv("USER_SERVICE_URL"), userID)
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
