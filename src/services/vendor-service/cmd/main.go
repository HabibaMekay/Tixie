package main

import (
	"log"
	"vendor-service/internal/api"
	"vendor-service/internal/db"
	"vendor-service/internal/db/repos"

	"github.com/gin-gonic/gin"
)

func main() {
	conn := db.ConnectDB()
	vendorRepo := repos.NewVendorRepository(conn)

	r := gin.Default()
	api.SetupRoutes(r, vendorRepo)

	log.Println("Vendor Service running on :9060")
	log.Fatal(r.Run(":9060"))
}
