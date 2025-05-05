package api

import (
	"reservation-service/internal/db/repos"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine, purchaseRepo *repos.PurchaseRepository, gatewayBaseURL string) {
	handler := NewHandler(purchaseRepo, gatewayBaseURL)
	r.POST("/reserve", handler.ReserveTicket)
	//r.GET("/:id", handler.GetTicket)
}
