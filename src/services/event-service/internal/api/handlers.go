package api

import (
	"event-service/internal/db/models"
	"event-service/internal/db/repos"
	"net/http"

	"log"
	"os"

	"github.com/gin-gonic/gin"
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
		logger.Printf("error:", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, events)
}

func (h *EventHandler) CreateEvent(c *gin.Context) {
	var event models.Event
	if err := c.ShouldBindJSON(&event); err != nil {
		logger.Printf("error:", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	if err := h.Repo.CreateEvent(event); err != nil {
		logger.Printf("error:", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create event"})
		return
	}

	logger.Println("Event successful")
	c.JSON(http.StatusCreated, event)
}
