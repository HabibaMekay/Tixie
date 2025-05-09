package handlers

import (
	"encoding/json"
	"net/http"

	"log"
	"os"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/paymentintent"
)

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

func CreatePaymentIntent(w http.ResponseWriter, r *http.Request) {
	logger.Println("Payment processing started")

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
	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(req.Amount),
		Currency: stripe.String(string(stripe.CurrencyUSD)),
	}
	params.SetIdempotencyKey(idempotencyKey)

	pi, err := paymentintent.New(params)
	if err != nil {
		logger.Printf("Failed to create payment intent: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	logger.Printf("Payment intent created successfully: %s", pi.ID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"client_secret":   pi.ClientSecret,
		"idempotency_key": idempotencyKey,
	})
}
