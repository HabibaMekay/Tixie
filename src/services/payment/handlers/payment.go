package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

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

func (h *PaymentHandler) CreatePaymentIntent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Amount int64 `json:"amount"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Amount <= 0 {
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
			return nil, fmt.Errorf("failed to create payment intent: %v", err)
		}

		return &paymentResponse{
			ClientSecret:   pi.ClientSecret,
			IdempotencyKey: idempotencyKey,
		}, nil
	})

	if result.Error != nil {
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
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
