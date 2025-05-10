package main

import (
	"log"
	"vendor-service/config"
	"vendor-service/internal/api"
	"vendor-service/internal/db"
	"vendor-service/internal/db/repos"

	"github.com/gin-gonic/gin"
)

func main() {
	if err := config.LoadConfig(); err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	conn := db.ConnectDB()
	vendorRepo := repos.NewVendorRepository(conn)

	r := gin.Default()
	api.SetupRoutes(r, vendorRepo)

	log.Println("Vendor Service running on :9060")
	log.Fatal(r.Run(":9060"))
}
