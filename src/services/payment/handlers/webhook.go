package handlers

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/stripe/stripe-go/webhook"
	brokerPkg "tixie.local/broker"
	"tixie.local/common"
)

type WebhookHandler struct {
	broker  *brokerPkg.Broker
	breaker *common.Breaker
}

type EmailMessage struct {
	RecipientEmail string `json:"recipient_email"`
	TicketID       string `json:"ticket_id"`
}

func NewWebhookHandler() *WebhookHandler {
	broker, err := brokerPkg.NewBroker(os.Getenv("RABBITMQ_URL"), "notification", "topic")
	if err != nil {
		logger.Printf("Warning: Failed to create broker: %v", err)
	}

	return &WebhookHandler{
		broker:  broker,
		breaker: common.NewBreaker("payment-webhook-service"),
	}
}

func (h *WebhookHandler) StripeWebhook(w http.ResponseWriter, r *http.Request) {
	const MaxBodyBytes = int64(65536)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)

	payload, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusServiceUnavailable)
		return
	}

	sigHeader := r.Header.Get("Stripe-Signature")
	secret := os.Getenv("SECRET_KEY")

	result := h.breaker.Execute(func() (interface{}, error) {
		event, err := webhook.ConstructEvent(payload, sigHeader, secret)
		if err != nil {
			return nil, fmt.Errorf("webhook signature verification failed: %v", err)
		}

		if event.Type == "payment_intent.succeeded" {
			logger.Printf("PaymentIntent %s succeeded", event.ID)
		}

		return event, nil
	})

	if result.Error != nil {
		logger.Printf("Webhook error: %v", result.Error)

		if common.IsCircuitBreakerError(result.Error) {
			status, msg := common.HandleCircuitBreakerError(result.Error)
			http.Error(w, msg, status)
			return
		}

		http.Error(w, result.Error.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *WebhookHandler) SimulateWebhook(w http.ResponseWriter, r *http.Request) {
	logger.Println("Simulated PaymentIntent succeeded")

	if h.broker == nil {
		http.Error(w, "Message broker not available", http.StatusServiceUnavailable)
		return
	}

	result := h.breaker.Execute(func() (interface{}, error) {
		msg := EmailMessage{
			RecipientEmail: "leaguedo@gmail.com",
			TicketID:       "abc-123-ticket",
		}
		if err := h.broker.Publish(msg, "email"); err != nil {
			return nil, fmt.Errorf("failed to publish email notification: %v", err)
		}
		return nil, nil
	})

	if result.Error != nil {
		logger.Printf("Failed to publish notification: %v", result.Error)

		if common.IsCircuitBreakerError(result.Error) {
			status, msg := common.HandleCircuitBreakerError(result.Error)
			http.Error(w, msg, status)
			return
		}

		http.Error(w, result.Error.Error(), http.StatusInternalServerError)
		return
	}

	logger.Println("Published email notification to notification-service successfully.")
	w.WriteHeader(http.StatusOK)
}
