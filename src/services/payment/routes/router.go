package routes

import (
	"payment/handlers"

	"github.com/gorilla/mux"
)

func SetupRouter() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/create-payment-intent", handlers.CreatePaymentIntent).Methods("POST")
	r.HandleFunc("/webhook", handlers.StripeWebhook).Methods("POST")
	r.HandleFunc("/simulate-webhook", handlers.SimulateWebhook).Methods("POST")
	return r
}
