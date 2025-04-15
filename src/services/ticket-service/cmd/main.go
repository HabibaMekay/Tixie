package main

import (
	"ticket-service/internal/api"
	"ticket-service/internal/db"
	"ticket-service/internal/db/repos"

	"github.com/gin-gonic/gin"
)

func main() {
	// Create database connection
	dbConn := db.NewDB()

	// Create a ticket repository instance using the connection
	repo := repos.NewTicketRepository(dbConn)

	// Initialize Gin
	r := gin.Default()

	// Set up your API routes
	api.SetupRoutes(r, repo)

	// Run the server on port 8082
	r.Run(":8082")
}
