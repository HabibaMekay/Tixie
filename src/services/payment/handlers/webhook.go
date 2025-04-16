package handlers

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/stripe/stripe-go/webhook"
)

func StripeWebhook(w http.ResponseWriter, r *http.Request) {
	const MaxBodyBytes = int64(65536)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)

	payload, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusServiceUnavailable)
		return
	}

	sigHeader := r.Header.Get("Stripe-Signature")
	secret := os.Getenv("STRIPE_WEBHOOK_SECRET")

	event, err := webhook.ConstructEvent(payload, sigHeader, secret)
	if err != nil {
		log.Printf("Webhook signature verification failed: %v", err)
		http.Error(w, "Webhook signature verification failed", http.StatusBadRequest)
		return
	}

	if event.Type == "payment_intent.succeeded" {
		log.Printf("PaymentIntent %s succeeded", event.ID)
	}

	w.WriteHeader(http.StatusOK)
}

func SimulateWebhook(w http.ResponseWriter, r *http.Request) {
	log.Println("simulated PaymentIntent succeeded")
	w.WriteHeader(http.StatusOK)
}
