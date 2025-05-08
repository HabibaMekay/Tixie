package api

import (
    "event-service/internal/db/models"
    "event-service/internal/db/repos"
    "net/http"
    "strconv"
    "github.com/gin-gonic/gin"
)


type EventHandler struct {
    Repo *repos.EventRepository
}


func NewEventHandler(repo *repos.EventRepository) *EventHandler {
    return &EventHandler{Repo: repo}
}


func (h *EventHandler) GetEvents(c *gin.Context) {
    events, err := h.Repo.GetAllEvents()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, events)
}


func (h *EventHandler) CreateEvent(c *gin.Context) {
    var event models.Event
    if err := c.ShouldBindJSON(&event); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
        return
    }
    if event.Price <= 0 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Price must be greater than zero"})
        return
    }

    if err := h.Repo.CreateEvent(event); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create event"})
        return
    }

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

