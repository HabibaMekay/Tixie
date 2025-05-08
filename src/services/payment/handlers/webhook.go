package handlers

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/stripe/stripe-go/webhook"
	brokerPkg "tixie.local/broker"
)

type EmailMessage struct {
	RecipientEmail string `json:"recipient_email"`
	TicketID       string `json:"ticket_id"`
}

func StripeWebhook(w http.ResponseWriter, r *http.Request) {
	const MaxBodyBytes = int64(65536)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)

	payload, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusServiceUnavailable)
		return
	}

	sigHeader := r.Header.Get("Stripe-Signature")
	secret := os.Getenv("SECRET_KEY")

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

	b, err := brokerPkg.NewBroker(os.Getenv("RABBITMQ_URL"), "notification", "topic")
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer b.Close()
	msg := EmailMessage{
		RecipientEmail: "leaguedo@gmail.com",
		TicketID:       "abc-123-ticket",
	}
	err = b.Publish(msg, "email")
	if err != nil {
		log.Fatalf("Failed to publish email notification: %v", err)
	}

	log.Println("Published email notification to notification-service successfully.")
	w.WriteHeader(http.StatusOK)
}
