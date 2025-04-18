package api

import (
	"github.com/gorilla/mux"
)

func SetupRoutes() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/", CreateUser).Methods("POST")
	r.HandleFunc("/", GetUsers).Methods("GET")
	r.HandleFunc("/{id}", GetUserByID).Methods("GET")
	r.HandleFunc("/{id}", UpdateUser).Methods("PUT")
	r.HandleFunc("/{id}", DeleteUser).Methods("DELETE")
	r.HandleFunc("/authenticate", AuthenticateUser).Methods("POST")

	return r
}
