package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"user-service/internal/db/models"

	"github.com/lib/pq"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Mock UserRepository ---
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) CreateUser(user models.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserRepository) GetAllUsers() ([]models.User, error) {
	args := m.Called()
	return args.Get(0).([]models.User), args.Error(1)
}

func (m *MockUserRepository) GetUserByID(id int) (models.User, error) {
	args := m.Called(id)
	return args.Get(0).(models.User), args.Error(1)
}

func (m *MockUserRepository) UpdateUser(id int, user models.User) error {
	args := m.Called(id, user)
	return args.Error(0)
}

func (m *MockUserRepository) DeleteUser(id int) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockUserRepository) CheckCredentials(username, password string) (bool, error) {
	args := m.Called(username, password)
	return args.Bool(0), args.Error(1)
}

// --- Test CreateUser Success ---
func TestCreateUser_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockUserRepository)
	handler := NewHandler(mockRepo)

	user := models.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	}

	// Hashing happens internally, so we mock with a matcher
	mockRepo.On("CreateUser", mock.MatchedBy(func(u models.User) bool {
		return u.Username == user.Username && u.Email == user.Email
	})).Return(nil)

	body, _ := json.Marshal(user)
	req, _ := http.NewRequest(http.MethodPost, "/v1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req

	handler.CreateUser(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockRepo.AssertExpectations(t)
}

// --- Test CreateUser Conflict ---
func TestCreateUser_Conflict(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockUserRepository)
	handler := NewHandler(mockRepo)

	user := models.User{
		Username: "existinguser",
		Email:    "duplicate@example.com",
		Password: "secret",
	}

	pgErr := &pq.Error{Code: "23505"}
	mockRepo.On("CreateUser", mock.Anything).Return(pgErr)

	body, _ := json.Marshal(user)
	req, _ := http.NewRequest(http.MethodPost, "/v1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req

	handler.CreateUser(c)

	assert.Equal(t, http.StatusConflict, w.Code)
	mockRepo.AssertExpectations(t)
}
