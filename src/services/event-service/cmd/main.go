package main

import (
    "event-service/internal/api"
    "event-service/internal/db"
    "event-service/internal/db/repos"
    "github.com/gin-gonic/gin"
    "log"
)

func main() {

    dbConn := db.ConnectDB()
    if dbConn == nil {
        log.Fatal("Database connection failed")
    }

    repo := repos.NewEventRepository(dbConn)

    r := gin.Default()
    api.SetupRoutes(r, repo)


    r.Run(":8080")
}

