package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/joho/godotenv"
)

//FAILS

func Test_WrongStripeKey(t *testing.T) {
	os.Setenv("STRIPE_SECRET_KEY", "sk_test_invalid")
	paymentHandler := NewPaymentHandler()
	body := []byte(`{"amount": 1000}`)
	req := httptest.NewRequest("POST", "/create-payment-intent", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	paymentHandler.CreatePaymentIntent(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500 for Stripe failure, got %d", rr.Code)
	}
}

func Test_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest("POST", "/create-payment-intent", bytes.NewBuffer([]byte(`invalid`)))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	paymentHandler := NewPaymentHandler()
	paymentHandler.CreatePaymentIntent(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for invalid JSON, got %d", rr.Code)
	}
}

func Test_MissingAmount(t *testing.T) {
	req := httptest.NewRequest("POST", "/create-payment-intent", bytes.NewBuffer([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	paymentHandler := NewPaymentHandler()
	paymentHandler.CreatePaymentIntent(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for missing amount, got %d", rr.Code)
	}
}

func Test_InvalidAmount1(t *testing.T) {
	body := []byte(`{"amount": -1000}`)
	req := httptest.NewRequest("POST", "/create-payment-intent", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	paymentHandler := NewPaymentHandler()
	paymentHandler.CreatePaymentIntent(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for negative amount, got %d", rr.Code)
	}
}

func Test_InvalidAmount2(t *testing.T) {
	body := []byte(`{"amount": 0}`)
	req := httptest.NewRequest("POST", "/create-payment-intent", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	paymentHandler := NewPaymentHandler()
	paymentHandler.CreatePaymentIntent(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for zero amount, got %d", rr.Code)
	}
}

// SUCCESS
func TestSimulateWebhook(t *testing.T) {
	req := httptest.NewRequest("POST", "/simulate-webhook", nil)
	rr := httptest.NewRecorder()

	WebhookHandler := NewWebhookHandler()
	WebhookHandler.SimulateWebhook(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200 from simulate-webhook, got %d", rr.Code)
	}
}

func TestCreatePaymentIntent_SuccessSimulated(t *testing.T) {
	err := godotenv.Load("../.env")
	if err != nil {
		panic("Error loading .env file from parent directory")
	}
	test_key := os.Getenv("SECRET_KEY")
	print("the key is: ", test_key)
	header_input := "Bearer " + test_key
	print("the test header stuff is: ", header_input)
	body := []byte(`{"amount": 1000}`)
	req := httptest.NewRequest("POST", "/create-payment-intent", bytes.NewBuffer(body))
	req.Header.Set("Authorization", header_input)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	paymentHandler := NewPaymentHandler()
	paymentHandler.CreatePaymentIntent(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200 for successful payment intent, got %d", rr.Code)
	}
	var response map[string]string
	err = json.NewDecoder(rr.Body).Decode(&response)
	if err != nil {
		t.Errorf("Response JSON invalid: %v", err)
	}
	if _, ok := response["client_secret"]; !ok {
		t.Errorf("Missing client_secret in response")
	}
	if _, ok := response["idempotency_key"]; !ok {
		t.Errorf("Missing idempotency_key in response")
	}
}
