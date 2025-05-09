package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reservation-service/internal/db/models"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockReservationRepo mocks the reservation repository
type MockReservationRepo struct {
	mock.Mock
}

func (m *MockReservationRepo) CreateReservation(eventID, userID, timeout int) (*models.Reservation, error) {
	args := m.Called(eventID, userID, timeout)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Reservation), args.Error(1)
}

func (m *MockReservationRepo) GetReservation(id int) (*models.Reservation, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Reservation), args.Error(1)
}

func (m *MockReservationRepo) CompleteReservation(id int) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockReservationRepo) ExpireReservation(id int) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockReservationRepo) GetExpiredReservations() ([]*models.Reservation, error) {
	args := m.Called()
	return args.Get(0).([]*models.Reservation), args.Error(1)
}

// MockPurchaseRepo mocks the purchase repository
type MockPurchaseRepo struct {
	mock.Mock
}

func (m *MockPurchaseRepo) GetPurchaseByTicketID(ticketID int) (*models.Purchase, error) {
	args := m.Called(ticketID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Purchase), args.Error(1)
}

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

// MockCircuitBreaker mocks the circuit breaker
type MockCircuitBreaker struct {
	mock.Mock
}

type Result struct {
	Data  interface{}
	Error error
}

func (m *MockCircuitBreaker) Execute(fn func() (interface{}, error)) Result {
	args := m.Called(fn)
	return args.Get(0).(Result)
}

// TestHandler is a testing version of Handler that uses our mock objects
type TestHandler struct {
	reserveRepo  *MockReservationRepo
	purchaseRepo *MockPurchaseRepo
	httpClient   *http.Client
	broker       *MockBroker
	breaker      *MockCircuitBreaker
}

// Forward handler methods to match the real Handler
func (h *TestHandler) ReserveTicket(c *gin.Context)              {}
func (h *TestHandler) CompleteReservation(c *gin.Context)        {}
func (h *TestHandler) CleanupExpiredReservations(c *gin.Context) {}

// Setup test router with mocked dependencies
func setupTestRouter(reserveRepo *MockReservationRepo, purchaseRepo *MockPurchaseRepo, broker *MockBroker, breaker *MockCircuitBreaker) (*gin.Engine, *TestHandler) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	handler := &TestHandler{
		reserveRepo:  reserveRepo,
		purchaseRepo: purchaseRepo,
		httpClient:   &http.Client{},
		broker:       broker,
		breaker:      breaker,
	}

	router.POST("/reserve", handler.ReserveTicket)
	router.POST("/complete", handler.CompleteReservation)

	return router, handler
}

// TestReserveTicket tests the reservation creation flow
func TestReserveTicket(t *testing.T) {
	mockReserveRepo := new(MockReservationRepo)
	mockPurchaseRepo := new(MockPurchaseRepo)
	mockBroker := new(MockBroker)
	mockBreaker := new(MockCircuitBreaker)

	// Setup test data
	eventID := 123
	userID := 456
	reservationID := 789
	expirationTime := time.Now().Add(15 * time.Minute)

	mockReservation := &models.Reservation{
		ID:             reservationID,
		EventID:        eventID,
		UserID:         userID,
		Status:         "pending",
		ExpirationTime: expirationTime,
	}

	// Setup expected behavior
	mockBreaker.On("Execute", mock.AnythingOfType("func() (interface {}, error)")).Return(Result{
		Data: struct {
			ID                 int     `json:"id"`
			Price              float64 `json:"price"`
			TicketsLeft        int     `json:"tickets_left"`
			ReservationTimeout int     `json:"reservation_timeout"`
		}{
			ID:                 eventID,
			Price:              10.0,
			TicketsLeft:        100,
			ReservationTimeout: 15,
		},
		Error: nil,
	}).Once()

	mockBreaker.On("Execute", mock.AnythingOfType("func() (interface {}, error)")).Return(Result{
		Data:  true,
		Error: nil,
	}).Once()

	mockBreaker.On("Execute", mock.AnythingOfType("func() (interface {}, error)")).Return(Result{
		Data:  true,
		Error: nil,
	}).Once()

	mockBreaker.On("Execute", mock.AnythingOfType("func() (interface {}, error)")).Return(Result{
		Data:  mockReservation,
		Error: nil,
	}).Once()

	// Setup router with mocked dependencies
	router, _ := setupTestRouter(mockReserveRepo, mockPurchaseRepo, mockBroker, mockBreaker)

	// Create request
	reqBody := gin.H{
		"event_id": eventID,
		"user_id":  userID,
	}
	reqJSON, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest(http.MethodPost, "/reserve", bytes.NewBuffer(reqJSON))
	req.Header.Set("Content-Type", "application/json")

	// Perform the request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusCreated, w.Code)

	// Verify mock expectations
	mockBreaker.AssertExpectations(t)
}

// TestCompleteReservation tests the reservation completion flow
func TestCompleteReservation(t *testing.T) {
	mockReserveRepo := new(MockReservationRepo)
	mockPurchaseRepo := new(MockPurchaseRepo)
	mockBroker := new(MockBroker)
	mockBreaker := new(MockCircuitBreaker)

	// Setup test data
	reservationID := 789
	eventID := 123
	userID := 456

	mockReservation := &models.Reservation{
		ID:             reservationID,
		EventID:        eventID,
		UserID:         userID,
		Status:         "pending",
		ExpirationTime: time.Now().Add(15 * time.Minute),
	}

	// Setup expected behavior
	mockBreaker.On("Execute", mock.AnythingOfType("func() (interface {}, error)")).Return(Result{
		Data:  mockReservation,
		Error: nil,
	}).Once()

	mockBreaker.On("Execute", mock.AnythingOfType("func() (interface {}, error)")).Return(Result{
		Data:  nil,
		Error: nil,
	}).Once()

	// Setup router with mocked dependencies
	router, _ := setupTestRouter(mockReserveRepo, mockPurchaseRepo, mockBroker, mockBreaker)

	// Create request
	reqBody := gin.H{
		"reservation_id": reservationID,
	}
	reqJSON, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest(http.MethodPost, "/complete", bytes.NewBuffer(reqJSON))
	req.Header.Set("Content-Type", "application/json")

	// Perform the request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify mock expectations
	mockBreaker.AssertExpectations(t)
}

// TestReservationExpiration tests the reservation expiration logic
func TestReservationExpiration(t *testing.T) {
	mockReserveRepo := new(MockReservationRepo)
	mockPurchaseRepo := new(MockPurchaseRepo)
	mockBroker := new(MockBroker)
	mockBreaker := new(MockCircuitBreaker)

	// Setup test data
	reservationID := 789
	eventID := 123

	expiredReservations := []*models.Reservation{
		{
			ID:      reservationID,
			EventID: eventID,
			Status:  "pending",
		},
	}

	// Setup expected behavior
	mockReserveRepo.On("GetExpiredReservations").Return(expiredReservations, nil)
	mockReserveRepo.On("ExpireReservation", reservationID).Return(nil)

	// Create test context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Setup handler directly
	handler := &TestHandler{
		reserveRepo:  mockReserveRepo,
		purchaseRepo: mockPurchaseRepo,
		httpClient:   &http.Client{},
		broker:       mockBroker,
		breaker:      mockBreaker,
	}

	// Call the cleanup method directly
	handler.CleanupExpiredReservations(c)

	// Verify mock expectations
	mockReserveRepo.AssertExpectations(t)
}
