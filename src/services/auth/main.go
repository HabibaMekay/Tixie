package main

import (
	"fmt"
	"net/http"

	"auth-service/config"
	"auth-service/handlers"

	"github.com/gorilla/mux"
)

func main() {
	config.LoadEnv()

	r := mux.NewRouter()
	r.HandleFunc("/login", handlers.Login).Methods("POST")
	r.HandleFunc("/oauth2-login", handlers.OAuth2Login).Methods("GET")
	r.HandleFunc("/callback", handlers.OAuth2Callback).Methods("GET")
	r.HandleFunc("/protected", handlers.Protected).Methods("GET")

	fmt.Println("Server is running on http://localhost:8080")
	http.ListenAndServe(":8080", r)
}
