package main

import (
    "log"
    "user-service/internal/db"
    "user-service/internal/api"
    "net/http"
)

func main() {
    db.ConnectDB()
    r := api.SetupRoutes()
    
    log.Println("User Service running on :8071")
    log.Fatal(http.ListenAndServe(":8071", r))
}
