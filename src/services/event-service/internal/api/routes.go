package api

import (
    "github.com/gorilla/mux"
)

func SetupRouter() *mux.Router {
    r := mux.NewRouter()
    r.HandleFunc("/events", GetEvents).Methods("GET")
    r.HandleFunc("/events", CreateEvent).Methods("POST")
    return r
}
