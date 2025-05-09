package api

import (
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"

	"user-service/internal/db/models"
	"user-service/internal/db/repos"
)

var logger *log.Logger

func init() {
	if _, err := os.Stat("logs"); os.IsNotExist(err) {
		os.MkdirAll("logs", os.ModePerm)
	}
	logFile, err := os.OpenFile("../logs/user.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logger.Println(`{"job":"user-service","level":"fatal","message":"failed to open log file: ` + err.Error() + `"}`)
		os.Exit(1)
	}

	logger = log.New(logFile, "", 0)
}

type Handler struct {
	repo *repos.UserRepository
}

func NewHandler(repo *repos.UserRepository) *Handler {
	return &Handler{repo: repo}
}

func hashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		logger.Println(`{"job":"user-service","level":"error","message":"Failed to hash password: ` + err.Error() + `"}`)
		return "", err
	}
	return string(hashedPassword), nil
}

func (h *Handler) CreateUser(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		logger.Println(`{"job":"user-service","level":"error","message":"Invalid request body: ` + err.Error() + `"}`)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hashedPassword, err := hashPassword(user.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}
	user.Password = hashedPassword

	err = h.repo.CreateUser(user)
	if err != nil {
		if pgErr, ok := err.(*pq.Error); ok && pgErr.Code == "23505" {
			logger.Println(`{"job":"user-service","level":"error","message":"Duplicate user: ` + err.Error() + `"}`)
			c.JSON(http.StatusConflict, gin.H{"error": "Username or Email already exists"})
			return
		}
		logger.Println(`{"job":"user-service","level":"error","message":"Failed to create user: ` + err.Error() + `"}`)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}
	logger.Println(`{"job":"user-service","level":"info","message":"User created successfully"}`)
	c.Status(http.StatusCreated)
}

func (h *Handler) GetUsers(c *gin.Context) {
	users, err := h.repo.GetAllUsers()
	if err != nil {
		logger.Println(`{"job":"user-service","level":"error","message":"Failed to retrieve users: ` + err.Error() + `"}`)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve users"})
		return
	}
	logger.Println(`{"job":"user-service","level":"info","message":"Users retrieved successfully"}`)
	c.JSON(http.StatusOK, users)
}

func (h *Handler) GetUserByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		logger.Println(`{"job":"user-service","level":"error","message":"Invalid user ID: ` + err.Error() + `"}`)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	user, err := h.repo.GetUserByID(id)
	if err != nil {
		logger.Println(`{"job":"user-service","level":"error","message":"User not found: ` + err.Error() + `"}`)
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	logger.Println(`{"job":"user-service","level":"info","message":"User retrieved successfully"}`)
	c.JSON(http.StatusOK, user)
}

func (h *Handler) UpdateUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		logger.Println(`{"job":"user-service","level":"error","message":"Invalid user ID: ` + err.Error() + `"}`)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var updatedUser models.User
	if err := c.ShouldBindJSON(&updatedUser); err != nil {
		logger.Println(`{"job":"user-service","level":"error","message":"Invalid request body: ` + err.Error() + `"}`)
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

	err = h.repo.UpdateUser(id, updatedUser)
	if err != nil {
		if pgErr, ok := err.(*pq.Error); ok && pgErr.Code == "23505" {
			logger.Println(`{"job":"user-service","level":"error","message":"Duplicate user update: ` + err.Error() + `"}`)
			c.JSON(http.StatusConflict, gin.H{"error": "Username or Email already exists"})
			return
		}
		logger.Println(`{"job":"user-service","level":"error","message":"Failed to update user: ` + err.Error() + `"}`)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}
	logger.Println(`{"job":"user-service","level":"info","message":"User updated successfully"}`)
	c.Status(http.StatusOK)
}

func (h *Handler) DeleteUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		logger.Println(`{"job":"user-service","level":"error","message":"Invalid user ID: ` + err.Error() + `"}`)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	err = h.repo.DeleteUser(id)
	if err != nil {
		logger.Println(`{"job":"user-service","level":"error","message":"Failed to delete user: ` + err.Error() + `"}`)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}
	logger.Println(`{"job":"user-service","level":"info","message":"User deleted successfully"}`)
	c.Status(http.StatusNoContent)
}

func (h *Handler) AuthenticateUser(c *gin.Context) {
	var creds models.Credentials
	if err := c.ShouldBindJSON(&creds); err != nil {
		logger.Println(`{"job":"user-service","level":"error","message":"Invalid login body: ` + err.Error() + `"}`)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	valid, err := h.repo.CheckCredentials(creds.Username, creds.Password)
	if err != nil {
		logger.Println(`{"job":"user-service","level":"error","message":"Error checking credentials: ` + err.Error() + `"}`)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	if !valid {
		logger.Println(`{"job":"user-service","level":"error","message":"Invalid login attempt for user: ` + creds.Username + `"}`)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}
	logger.Println(`{"job":"user-service","level":"info","message":"User logged in successfully: ` + creds.Username + `"}`)
	c.Status(http.StatusOK)
}
