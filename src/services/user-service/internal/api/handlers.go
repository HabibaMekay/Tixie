package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
	circuitbreaker "tixie.local/common"

	"user-service/internal/db/models"
	"user-service/internal/db/repos"
)

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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hashedPassword, err := hashPassword(user.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}
	user.Password = hashedPassword

	result := h.breaker.Execute(func() (interface{}, error) {
		return nil, h.repo.CreateUser(user)
	})

	if result.Error != nil {
		if circuitbreaker.IsCircuitBreakerError(result.Error) {
			status, msg := circuitbreaker.HandleCircuitBreakerError(result.Error)
			c.JSON(status, gin.H{"error": msg})
			return
		}
		if pgErr, ok := result.Error.(*pq.Error); ok && pgErr.Code == "23505" {
			c.JSON(http.StatusConflict, gin.H{"error": "Username or Email already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	c.Status(http.StatusCreated)
}

func (h *Handler) GetUsers(c *gin.Context) {
	result := h.breaker.Execute(func() (interface{}, error) {
		return h.repo.GetAllUsers()
	})

	if result.Error != nil {
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, users)
}

func (h *Handler) GetUserByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	result := h.breaker.Execute(func() (interface{}, error) {
		return h.repo.GetUserByID(id)
	})

	if result.Error != nil {
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	if user.ID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *Handler) UpdateUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var updatedUser models.User
	if err := c.ShouldBindJSON(&updatedUser); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if updatedUser.Password != "" {
		hashedPassword, err := hashPassword(updatedUser.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
			return
		}
		updatedUser.Password = hashedPassword
	}

	result := h.breaker.Execute(func() (interface{}, error) {
		return nil, h.repo.UpdateUser(id, updatedUser)
	})

	if result.Error != nil {
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

	c.Status(http.StatusOK)
}

func (h *Handler) DeleteUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	result := h.breaker.Execute(func() (interface{}, error) {
		return nil, h.repo.DeleteUser(id)
	})

	if result.Error != nil {
		if circuitbreaker.IsCircuitBreakerError(result.Error) {
			status, msg := circuitbreaker.HandleCircuitBreakerError(result.Error)
			c.JSON(status, gin.H{"error": msg})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) AuthenticateUser(c *gin.Context) {
	var creds models.Credentials
	if err := c.ShouldBindJSON(&creds); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	result := h.breaker.Execute(func() (interface{}, error) {
		return h.repo.CheckCredentials(creds.Username, creds.Password)
	})

	if result.Error != nil {
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	if !valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	c.Status(http.StatusOK)
}
