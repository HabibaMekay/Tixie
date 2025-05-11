package api

import (
	"vendor-service/internal/db/repos"

	brokerPkg "tixie.local/broker"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine, repo *repos.VendorRepository) {
	broker, err := brokerPkg.NewBroker("amqp://guest:guest@rabbitmq:5672/", "vendor-service", "topic")
	if err != nil {
		panic(err)
	}

	handler := NewHandler(repo, broker)

	vendors := r.Group("/v1")
	{
		vendors.POST("/signup", handler.CreateVendor)
		vendors.GET("", handler.GetVendors)
		vendors.GET("/:id", handler.GetVendorByID)
		vendors.PUT("/:id", handler.UpdateVendor)
		vendors.DELETE("/:id", handler.DeleteVendor)
		vendors.POST("/authenticate", handler.AuthenticateVendor)
		vendors.POST("/events", handler.CreateVendorEvent)
	}
}
