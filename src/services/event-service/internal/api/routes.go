package api

import (
	"event-service/internal/db/repos"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine, repo *repos.EventRepository) {
	handler := NewEventHandler(repo)

	events := r.Group("/v1")
	{
		events.GET("", handler.GetEvents)
		events.POST("", handler.CreateEvent)
		events.GET("/:id", handler.GetEventByID)
		events.PATCH("/:id/tickets", handler.UpdateTicketsSold) 
	}
}
