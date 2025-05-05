package api

import (
    "event-service/internal/db/models"
    "event-service/internal/db/repos"
    "net/http"

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

    if err := h.Repo.CreateEvent(event); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create event"})
        return
    }

    c.JSON(http.StatusCreated, event)
}



