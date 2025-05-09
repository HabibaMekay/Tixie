package main

import (
	"fmt"
	"log"

	"auth-service/config"
	"auth-service/internal/api"

	// "auth-service/internal/db/repos"
	// "auth-service/internal/db/models"

	"github.com/gin-gonic/gin"
)

func main() {
	config.LoadEnv()
	// db.Connect()
	// init.LoadInitialSQL() // if needed to load purchase.sql

	r := gin.Default()
	api.RegisterRoutes(r)

	fmt.Println("Auth service running on http://localhost:8080")
	log.Fatal(r.Run(":8080"))
}
