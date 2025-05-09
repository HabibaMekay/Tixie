package api

import (
	"ticket-service/internal/db/repos"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine, repo *repos.TicketRepository) {

	handler := NewHandler(repo)

	// API routes for tickets
	tickets := r.Group("/v1")
	{
		tickets.GET("/ws/events-with-tickets", handler.GetEventsWithTicketsWS)

		tickets.GET("/ws/tickets/:event_id", handler.GetTicketsByEventIDWS)

		tickets.GET("/events-with-tickets", handler.GetEventsWithTickets)

		tickets.GET("/:id", handler.GetTicketByID)

		tickets.GET("", handler.GetTicketsByEventID)

		tickets.POST("", handler.CreateTicket)

		tickets.PUT("/:id/status", handler.UpdateTicketStatus)

		tickets.GET("/verify/:ticket_code", handler.GetTicketByCode)
	}
}
