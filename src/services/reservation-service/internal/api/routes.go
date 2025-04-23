package api

import (
	"net/http"
	"reservation-service/internal/db/repos"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine, purchaseRepo *repos.PurchaseRepository, ticketClient *http.Client) {
	handler := NewReservationHandler(purchaseRepo, ticketClient)
	r.POST("/reserve", handler.ReserveTicket)
	r.GET("/:id", handler.GetTicket)
}
