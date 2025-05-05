package api

import (
	"net/http"
	"reservation-service/internal/db/repos"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine, purchaseRepo *repos.PurchaseRepository, ticketClient *http.Client) {
	handler := NewReservationHandler(purchaseRepo, ticketClient)
	reservation := r.Group("/v1")
	{
		reservation.POST("/", handler.ReserveTicket)
		reservation.GET("/:id", handler.GetTicket)
	}
}
