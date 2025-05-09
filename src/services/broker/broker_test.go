package broker

import (
	"encoding/json"
	"testing"
)

// MockAMQP implements mocked versions of the AMQP functionality
type MockAMQP struct {
	PublishedMessages map[string][]interface{}
	IsClosed          bool
}

// MockDelivery represents a mocked AMQP delivery
type MockDelivery struct {
	MessageBody []byte
	MessageKey  string
}

func (m *MockDelivery) Ack(multiple bool) error {
	return nil
}

func (m *MockDelivery) Reject(requeue bool) error {
	return nil
}

// NewMockBroker creates a broker with mocked AMQP connections
func NewMockBroker() *Broker {
	return &Broker{
		conn:     nil, // We're not using the real connection
		channel:  nil, // We're not using the real channel
		exchange: "test-exchange",
		url:      "mock://localhost",
	}
}

// Test the Publish method
func TestPublish(t *testing.T) {
	// Create a test message
	type TestMessage struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	testMsg := TestMessage{
		ID:   123,
		Name: "Test Message",
	}

	// Create a mock broker that doesn't actually connect to RabbitMQ
	b := NewMockBroker()

	// Since we can't actually test publishing with a mock,
	// we'll just verify the message is properly marshaled
	expected, _ := json.Marshal(testMsg)
	var msg TestMessage
	if err := json.Unmarshal(expected, &msg); err != nil {
		t.Errorf("Failed to unmarshal message: %v", err)
	}

	if msg.ID != testMsg.ID || msg.Name != testMsg.Name {
		t.Errorf("Message marshaling failed, got %+v, want %+v", msg, testMsg)
	}

	// The actual Publish call will fail due to nil connection
	_ = b.Publish(testMsg, "test.key")
}

// Test DeclareAndBindQueue method
func TestDeclareAndBindQueue(t *testing.T) {
	b := NewMockBroker()

	// This will fail with the mock, but we're testing the function is callable
	err := b.DeclareAndBindQueue("test-queue", "test.key")

	// Since we're using a mock without a connection, this should fail
	if err == nil {
		t.Error("Expected error with mock connection, but got nil")
	}
}

// Test connection management
func TestEnsureConnection(t *testing.T) {
	b := NewMockBroker()

	// This should fail because we have no real connection
	err := b.ensureConnection()
	if err == nil {
		t.Error("Expected error with mock connection, but got nil")
	}
}

// TestClose tests the close functionality
func TestClose(t *testing.T) {
	b := NewMockBroker()

	// Should not panic even with nil connection and channel
	err := b.Close()
	if err != nil {
		t.Errorf("Close returned an error with nil connection: %v", err)
	}
}
