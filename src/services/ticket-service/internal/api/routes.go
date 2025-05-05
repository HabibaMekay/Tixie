package api

import (
	"ticket-service/internal/db/repos"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine, repo *repos.TicketRepository, gatewayBaseURL string) {

	handler := NewHandler(repo, gatewayBaseURL)

	// API routes for tickets
	tickets := r.Group("/v1")
	{

		tickets.GET("/:id", handler.GetTicketByID)

		tickets.GET("", handler.GetTicketsByEventID)

		tickets.POST("", handler.CreateTicket)

		tickets.PUT("/:id/status", handler.UpdateTicketStatus)
	}
}
