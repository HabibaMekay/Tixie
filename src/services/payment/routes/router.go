package routes

import (
	"payment/handlers"

	"github.com/gorilla/mux"
)

func SetupRouter() *mux.Router {
	r := mux.NewRouter()

	paymentHandler := handlers.NewPaymentHandler()
	webhookHandler := handlers.NewWebhookHandler()

	r.HandleFunc("/create-payment-intent", paymentHandler.CreatePaymentIntent).Methods("POST")
	// r.HandleFunc("/webhook", webhookHandler.StripeWebhook).Methods("POST")
	r.HandleFunc("/simulate-webhook", webhookHandler.SimulateWebhook).Methods("POST")

	return r
}
