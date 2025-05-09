package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockBroker mocks the message broker
type MockBroker struct {
	mock.Mock
}

func (m *MockBroker) Publish(message interface{}, routingKey string) error {
	args := m.Called(message, routingKey)
	return args.Error(0)
}

func (m *MockBroker) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockBroker) DeclareAndBindQueue(queueName, routingKey string) error {
	args := m.Called(queueName, routingKey)
	return args.Error(0)
}

func (m *MockBroker) Consume(queueName string) (<-chan []byte, error) {
	args := m.Called(queueName)
	return args.Get(0).(<-chan []byte), args.Error(1)
}

// Mock CircuitBreaker for testing
type MockBreaker struct {
	mock.Mock
}

func (m *MockBreaker) Execute(fn func() (interface{}, error)) (Result struct {
	Data  interface{}
	Error error
}) {
	args := m.Called(mock.Anything)
	// Execute the function directly for testing
	data, err := fn()
	Result.Data = data
	Result.Error = err
	return Result
}

// TestSimulateWebhook tests the webhook simulation endpoint
func TestSimulateWebhook(t *testing.T) {
	// Create mocks
	mockBroker := new(MockBroker)
	mockBreaker := new(MockBreaker)

	// Setup test handler
	handler := &WebhookHandler{
		broker:  mockBroker,
		breaker: mockBreaker,
	}

	// Setup expected behavior
	mockBroker.On("Publish", mock.Anything, "email").Return(nil)

	// Create test request
	req, err := http.NewRequest("POST", "/simulate-webhook", nil)
	assert.NoError(t, err)

	// Create recorder for the response
	w := httptest.NewRecorder()

	// Call the handler method
	handler.SimulateWebhook(w, req)

	// Verify expectations
	mockBroker.AssertExpectations(t)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestStripeWebhook tests the Stripe webhook endpoint
func TestStripeWebhook(t *testing.T) {
	// Create mocks
	mockBroker := new(MockBroker)
	mockBreaker := new(MockBreaker)

	// Setup test handler
	handler := &WebhookHandler{
		broker:  mockBroker,
		breaker: mockBreaker,
	}

	// Create invalid test payload
	invalidPayload := strings.NewReader(`{"type": "payment_intent.succeeded", "data": {"object": {"id": "pi_test"}}}`)

	// Create test request
	req, err := http.NewRequest("POST", "/webhook", invalidPayload)
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Stripe-Signature", "invalid_signature")

	// Create recorder for the response
	w := httptest.NewRecorder()

	// Call the handler method
	handler.StripeWebhook(w, req)

	// In a real test, with mocking of the webhook.ConstructEvent we would validate the behavior
	// Here we're expecting a 400 Bad Request because the signature verification will fail
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
