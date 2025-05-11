package api

import (
	"ticket-service/internal/db/repos"

	"github.com/gin-gonic/gin"
	brokerPkg "tixie.local/broker"
)

func SetupRoutes(r *gin.Engine, repo *repos.TicketRepository) {

	broker, err := brokerPkg.NewBroker("amqp://guest:guest@rabbitmq:5672/", "notification", "topic")
	if err != nil {
		panic(err)
	}
	handler := NewHandler(repo, broker)

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
