package api

import (
	"reservation-service/internal/db/repos"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine, purchaseRepo *repos.PurchaseRepository, reserveRepo *repos.ReservationRepository) {
	handler := NewHandler(purchaseRepo, reserveRepo)
	res := r.Group("/v1")
	{
		res.POST("", handler.ReserveTicket)
		//res.GET("/:id", handler.GetTicket)
		res.POST("/verify", handler.VerifyTicket)
		res.POST("/complete", handler.CompleteReservation)
		res.GET("/cleanup", handler.CleanupExpiredReservations)
	}
}
