package api

import (
    "encoding/json"
    "event-service/internal/db/models"
    "event-service/internal/db/repos"
    "net/http"
)

func GetEvents(w http.ResponseWriter, r *http.Request) {
    events, err := repos.GetAllEvents()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(events)
}

func CreateEvent(w http.ResponseWriter, r *http.Request) {
    var event models.Event
    if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
        http.Error(w, "Invalid input", http.StatusBadRequest)
        return
    }

    err := repos.CreateEvent(event)
    if err != nil {
        http.Error(w, "Failed to create event", http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(event)
}


