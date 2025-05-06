package api

import (
	"reservation-service/internal/db/repos"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine, purchaseRepo *repos.PurchaseRepository) {
	handler := NewHandler(purchaseRepo)
	res := r.Group("/v1")
	{
		res.POST("", handler.ReserveTicket)
		//res.GET("/:id", handler.GetTicket)
	}
}
