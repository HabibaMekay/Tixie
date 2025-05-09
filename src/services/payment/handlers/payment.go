package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"log"
	"os"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/paymentintent"
	circuitbreaker "tixie.local/common"
)

type PaymentHandler struct {
	breaker *circuitbreaker.Breaker
}

func NewPaymentHandler() *PaymentHandler {
	return &PaymentHandler{
		breaker: circuitbreaker.NewBreaker("payment-service"),
	}
}

type paymentResponse struct {
	ClientSecret   string `json:"client_secret"`
	IdempotencyKey string `json:"idempotency_key"`
}

var logger *log.Logger

func init() {
	if _, err := os.Stat("logs"); os.IsNotExist(err) {
		os.MkdirAll("logs", os.ModePerm)
	}
	logFile, err := os.OpenFile("logs/service.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("failed to open log file: %v", err)
	}

	logger = log.New(logFile, "PAYMENT: ", log.LstdFlags|log.Lshortfile)
}

func (h *PaymentHandler) CreatePaymentIntent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Amount int64 `json:"amount"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Printf("Invalid request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Amount <= 0 {
		logger.Printf("Invalid amount: %d", req.Amount)
		http.Error(w, "amount must be greater than 0", http.StatusBadRequest)
		return
	}

	idempotencyKey := uuid.New().String()

	result := h.breaker.Execute(func() (interface{}, error) {
		params := &stripe.PaymentIntentParams{
			Amount:   stripe.Int64(req.Amount),
			Currency: stripe.String(string(stripe.CurrencyUSD)),
		}
		params.SetIdempotencyKey(idempotencyKey)

		pi, err := paymentintent.New(params)
		if err != nil {
			logger.Printf("Failed to create payment intent: %v", err)
			return nil, fmt.Errorf("failed to create payment intent: %v", err)
		}

		return &paymentResponse{
			ClientSecret:   pi.ClientSecret,
			IdempotencyKey: idempotencyKey,
		}, nil
	})

	if result.Error != nil {
		logger.Printf("Error from breaker: %v", result.Error)
		if circuitbreaker.IsCircuitBreakerError(result.Error) {
			status, msg := circuitbreaker.HandleCircuitBreakerError(result.Error)
			http.Error(w, msg, status)
			return
		}
		http.Error(w, result.Error.Error(), http.StatusInternalServerError)
		return
	}

	response, ok := result.Data.(*paymentResponse)
	if !ok {
		logger.Printf("Unexpected response type from breaker: %+v", result.Data)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	logger.Printf("Successfully created payment intent: %+v", response)
}
