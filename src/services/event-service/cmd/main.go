package main

import (
    "log"
    "net/http"
    "event-service/internal/db"
    "event-service/internal/api"
)

func main() {
    db.ConnectDB()
    r := api.SetupRouter()
    
    log.Println("Event Service running on :8080")
    log.Fatal(http.ListenAndServe(":8080", r))
}

