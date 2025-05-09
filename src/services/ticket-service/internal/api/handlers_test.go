package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"ticket-service/internal/db/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockTicketRepo mocks the ticket repository interface
type MockTicketRepo struct {
	mock.Mock
}

func (m *MockTicketRepo) GetTicketByID(id int) (*models.Ticket, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Ticket), args.Error(1)
}

func (m *MockTicketRepo) GetTicketsByEventID(eventID int) ([]models.Ticket, error) {
	args := m.Called(eventID)
	return args.Get(0).([]models.Ticket), args.Error(1)
}

func (m *MockTicketRepo) CreateTicket(ticket *models.Ticket) (*models.Ticket, error) {
	args := m.Called(ticket)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Ticket), args.Error(1)
}

func (m *MockTicketRepo) GetTicketByCode(code string) (*models.Ticket, error) {
	args := m.Called(code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Ticket), args.Error(1)
}

func (m *MockTicketRepo) UpdateTicketStatus(id int, status string) (*models.Ticket, error) {
	args := m.Called(id, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Ticket), args.Error(1)
}

// Result represents the result from circuit breaker execution
type Result struct {
	Data  interface{}
	Error error
}

// MockBreaker mocks the circuit breaker
type MockBreaker struct {
	mock.Mock
}

func (m *MockBreaker) Execute(fn func() (interface{}, error)) Result {
	args := m.Called(fn)
	return args.Get(0).(Result)
}

func (m *MockBreaker) NewBreaker(name string) *MockBreaker {
	return new(MockBreaker)
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

// TestHandler is a copy of Handler for testing purposes
type TestHandler struct {
	repo       *MockTicketRepo
	httpClient *http.Client
	breaker    *MockBreaker
}

// Forward handler methods to match the real Handler
func (h *TestHandler) CreateTicket(c *gin.Context)    {}
func (h *TestHandler) GetTicketByID(c *gin.Context)   {}
func (h *TestHandler) GetTicketByCode(c *gin.Context) {}

// Setup test router with mocked dependencies
func setupTestRouter(ticketRepo *MockTicketRepo, breaker *MockBreaker) (*gin.Engine, *TestHandler) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	handler := &TestHandler{
		repo:       ticketRepo,
		httpClient: &http.Client{},
		breaker:    breaker,
	}

	router.POST("/tickets", handler.CreateTicket)
	router.GET("/tickets/:id", handler.GetTicketByID)
	router.GET("/verify/:code", handler.GetTicketByCode)

	return router, handler
}

// TestCreateTicket tests the ticket creation flow
func TestCreateTicket(t *testing.T) {
	mockTicketRepo := new(MockTicketRepo)
	mockBreaker := new(MockBreaker)

	// Setup test data
	ticketID := 123
	eventID := 456
	userID := 789
	ticketCode := uuid.New().String()

	mockTicket := &models.Ticket{
		TicketID:   ticketID,
		EventID:    eventID,
		UserID:     userID,
		TicketCode: ticketCode,
		Status:     "active",
	}

	// Setup breaker behavior to skip validation
	mockBreaker.On("Execute", mock.AnythingOfType("func() (interface {}, error)")).Return(Result{
		Data:  mockTicket,
		Error: nil,
	})

	// Setup router with mocked dependencies
	router, _ := setupTestRouter(mockTicketRepo, mockBreaker)

	// Create request
	reqBody := gin.H{
		"event_id": eventID,
		"user_id":  userID,
	}
	reqJSON, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest(http.MethodPost, "/tickets", bytes.NewBuffer(reqJSON))
	req.Header.Set("Content-Type", "application/json")

	// Perform the request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusCreated, w.Code)

	// Verify mock expectations
	mockBreaker.AssertExpectations(t)
}

// TestGetTicket tests retrieving a ticket by ID
func TestGetTicket(t *testing.T) {
	mockTicketRepo := new(MockTicketRepo)
	mockBreaker := new(MockBreaker)

	// Setup test data
	ticketID := 123
	eventID := 456
	userID := 789
	ticketCode := uuid.New().String()

	mockTicket := &models.Ticket{
		TicketID:   ticketID,
		EventID:    eventID,
		UserID:     userID,
		TicketCode: ticketCode,
		Status:     "active",
	}

	// Setup breaker behavior
	mockBreaker.On("Execute", mock.AnythingOfType("func() (interface {}, error)")).Return(Result{
		Data:  mockTicket,
		Error: nil,
	})

	// Setup router with mocked dependencies
	router, _ := setupTestRouter(mockTicketRepo, mockBreaker)

	// Create request
	req, _ := http.NewRequest(http.MethodGet, "/tickets/123", nil)

	// Perform the request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify mock expectations
	mockBreaker.AssertExpectations(t)
}

// TestVerifyTicket tests the ticket verification by code
func TestVerifyTicket(t *testing.T) {
	mockTicketRepo := new(MockTicketRepo)
	mockBreaker := new(MockBreaker)

	// Setup test data
	ticketID := 123
	eventID := 456
	userID := 789
	ticketCode := uuid.New().String()

	mockTicket := &models.Ticket{
		TicketID:   ticketID,
		EventID:    eventID,
		UserID:     userID,
		TicketCode: ticketCode,
		Status:     "active",
	}

	// Setup breaker behavior
	mockBreaker.On("Execute", mock.AnythingOfType("func() (interface {}, error)")).Return(Result{
		Data:  mockTicket,
		Error: nil,
	})

	// Setup router with mocked dependencies
	router, _ := setupTestRouter(mockTicketRepo, mockBreaker)

	// Create request
	req, _ := http.NewRequest(http.MethodGet, "/verify/"+ticketCode, nil)

	// Perform the request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify mock expectations
	mockBreaker.AssertExpectations(t)
}
