package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
	circuitbreaker "tixie.local/common"

	"log"
	"os"

	"user-service/internal/db/models"
	"user-service/internal/db/repos"
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

	logger = log.New(logFile, "USER-SERVICE: ", log.LstdFlags|log.Lshortfile)
}

type Handler struct {
	repo    *repos.UserRepository
	breaker *circuitbreaker.Breaker
}

func NewHandler(repo *repos.UserRepository) *Handler {
	return &Handler{
		repo:    repo,
		breaker: circuitbreaker.NewBreaker("user-service"),
	}
}

func hashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

func (h *Handler) CreateUser(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		logger.Printf("error: ", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hashedPassword, err := hashPassword(user.Password)
	if err != nil {
		logger.Printf("error: ", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}
	user.Password = hashedPassword
	result := h.breaker.Execute(func() (interface{}, error) {
		return nil, h.repo.CreateUser(user)
	})

	if result.Error != nil {
		// Log all errors
		logger.Printf("CreateUser error: %v", result.Error)

		if circuitbreaker.IsCircuitBreakerError(result.Error) {
			status, msg := circuitbreaker.HandleCircuitBreakerError(result.Error)
			c.JSON(status, gin.H{"error": msg})
			return
		}

		if pgErr, ok := result.Error.(*pq.Error); ok && pgErr.Code == "23505" {
			c.JSON(http.StatusConflict, gin.H{"error": "Username or Email already exists"})
			return
		}
		logger.Printf("error: ", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}
	logger.Println("User created succesfully")
	c.Status(http.StatusCreated)
}

func (h *Handler) GetUsers(c *gin.Context) {
	result := h.breaker.Execute(func() (interface{}, error) {
		return h.repo.GetAllUsers()
	})

	if result.Error != nil {
		logger.Printf("GetAllUsers error: %v", result.Error)

		if circuitbreaker.IsCircuitBreakerError(result.Error) {
			status, msg := circuitbreaker.HandleCircuitBreakerError(result.Error)
			c.JSON(status, gin.H{"error": msg})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve users"})
		return
	}

	users, ok := result.Data.([]models.User)
	if !ok {
		logger.Printf("Unexpected result type from breaker: %+v", result.Data)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	logger.Println("Users retrieved successfully")
	c.JSON(http.StatusOK, users)
}

func (h *Handler) GetUserByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		logger.Printf("Invalid user ID: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	result := h.breaker.Execute(func() (interface{}, error) {
		return h.repo.GetUserByID(id)
	})

	if result.Error != nil {
		logger.Printf("Error retrieving user by ID %d: %v", id, result.Error)

		if circuitbreaker.IsCircuitBreakerError(result.Error) {
			status, msg := circuitbreaker.HandleCircuitBreakerError(result.Error)
			c.JSON(status, gin.H{"error": msg})
			return
		}

		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	user, ok := result.Data.(models.User)
	if !ok {
		logger.Printf("Unexpected data type in breaker result: %+v", result.Data)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	if user.ID == 0 {
		logger.Printf("User with ID %d not found", id)
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	logger.Printf("User with ID %d found successfully", id)
	c.JSON(http.StatusOK, user)
}

func (h *Handler) UpdateUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		logger.Printf("Invalid user ID: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var updatedUser models.User
	if err := c.ShouldBindJSON(&updatedUser); err != nil {
		logger.Printf("Invalid request body: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if updatedUser.Password != "" {
		hashedPassword, err := hashPassword(updatedUser.Password)
		if err != nil {
			logger.Printf("Password hashing failed: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
			return
		}
		updatedUser.Password = hashedPassword
	}

	result := h.breaker.Execute(func() (interface{}, error) {
		return nil, h.repo.UpdateUser(id, updatedUser)
	})

	if result.Error != nil {
		logger.Printf("UpdateUser error: %v", result.Error)

		if circuitbreaker.IsCircuitBreakerError(result.Error) {
			status, msg := circuitbreaker.HandleCircuitBreakerError(result.Error)
			c.JSON(status, gin.H{"error": msg})
			return
		}

		if pgErr, ok := result.Error.(*pq.Error); ok && pgErr.Code == "23505" {
			c.JSON(http.StatusConflict, gin.H{"error": "Username or Email already exists"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	logger.Printf("User with ID %d updated successfully", id)
	c.Status(http.StatusOK)
}

func (h *Handler) DeleteUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		logger.Printf("Invalid user ID: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	result := h.breaker.Execute(func() (interface{}, error) {
		return nil, h.repo.DeleteUser(id)
	})

	if result.Error != nil {
		logger.Printf("DeleteUser error: %v", result.Error)

		if circuitbreaker.IsCircuitBreakerError(result.Error) {
			status, msg := circuitbreaker.HandleCircuitBreakerError(result.Error)
			c.JSON(status, gin.H{"error": msg})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}

	logger.Printf("User with ID %d deleted successfully", id)
	c.Status(http.StatusNoContent)
}

func (h *Handler) AuthenticateUser(c *gin.Context) {
	var creds models.Credentials
	if err := c.ShouldBindJSON(&creds); err != nil {
		logger.Printf("Invalid request body: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	result := h.breaker.Execute(func() (interface{}, error) {
		return h.repo.CheckCredentials(creds.Username, creds.Password)
	})

	if result.Error != nil {
		logger.Printf("Authentication error: %v", result.Error)

		if circuitbreaker.IsCircuitBreakerError(result.Error) {
			status, msg := circuitbreaker.HandleCircuitBreakerError(result.Error)
			c.JSON(status, gin.H{"error": msg})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	valid, ok := result.Data.(bool)
	if !ok {
		logger.Println("Unexpected return type from credential check")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	if !valid {
		logger.Println("Invalid username or password attempt")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	logger.Println("User logged in successfully")
	c.Status(http.StatusOK)
}
