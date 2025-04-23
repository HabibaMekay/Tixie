package main

import (
	"log"
	"user-service/internal/api"
	"user-service/internal/db"
	"user-service/internal/db/repos"

	"github.com/gin-gonic/gin"
)

func main() {
	conn := db.ConnectDB()
	userRepo := repos.NewUserRepository(conn)

	r := gin.Default()
	api.SetupRoutes(r, userRepo)

	log.Println("User Service running on :8081")
	log.Fatal(r.Run(":8081"))
}
