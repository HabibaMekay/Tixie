package api

import (
	"vendor-service/internal/db/repos"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine, repo *repos.VendorRepository) {
	handler := NewHandler(repo)

	vendors := r.Group("/v1")
	{
		vendors.POST("", handler.CreateVendor)
		vendors.GET("", handler.GetVendors)
		vendors.GET("/:id", handler.GetVendorByID)
		vendors.PUT("/:id", handler.UpdateVendor)
		vendors.DELETE("/:id", handler.DeleteVendor)
		vendors.POST("/authenticate", handler.AuthenticateVendor)
		vendors.POST("/:id/events", handler.CreateVendorEvent)
	}
}
