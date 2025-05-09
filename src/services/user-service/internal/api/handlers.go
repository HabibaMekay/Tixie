package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"

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
	repo *repos.UserRepository
}

func NewHandler(repo *repos.UserRepository) *Handler {
	return &Handler{repo: repo}
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

	err = h.repo.CreateUser(user)
	if err != nil {
		if pgErr, ok := err.(*pq.Error); ok && pgErr.Code == "23505" {
			logger.Printf("error: ", err.Error())
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
	users, err := h.repo.GetAllUsers()
	if err != nil {
		logger.Printf("error: ", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve users"})
		return
	}
	logger.Println("Users get successfully")
	c.JSON(http.StatusOK, users)
}

func (h *Handler) GetUserByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		logger.Printf("error: ", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	user, err := h.repo.GetUserByID(id)
	if err != nil {
		logger.Printf("error: ", err.Error())
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	logger.Printf("User found successfully")
	c.JSON(http.StatusOK, user)
}

func (h *Handler) UpdateUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		logger.Printf("error: ", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var updatedUser models.User
	if err := c.ShouldBindJSON(&updatedUser); err != nil {
		logger.Printf("error: ", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if updatedUser.Password != "" {
		hashedPassword, err := hashPassword(updatedUser.Password)
		if err != nil {
			logger.Printf("error: ", err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
			return
		}
		updatedUser.Password = hashedPassword
	}

	err = h.repo.UpdateUser(id, updatedUser)
	if err != nil {
		if pgErr, ok := err.(*pq.Error); ok && pgErr.Code == "23505" {
			logger.Printf("error: ", err.Error())
			c.JSON(http.StatusConflict, gin.H{"error": "Username or Email already exists"})
			return
		}
		logger.Printf("error: ", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}
	logger.Println("user created succesffully")
	c.Status(http.StatusOK)
}

func (h *Handler) DeleteUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		logger.Printf("error: ", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	err = h.repo.DeleteUser(id)
	if err != nil {
		logger.Printf("error: ", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}
	logger.Println("User deletion successful")

	c.Status(http.StatusNoContent)
}

func (h *Handler) AuthenticateUser(c *gin.Context) {
	var creds models.Credentials
	if err := c.ShouldBindJSON(&creds); err != nil {
		logger.Printf("error: ", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	valid, err := h.repo.CheckCredentials(creds.Username, creds.Password)
	if err != nil {
		logger.Printf("error: ", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	if !valid {
		logger.Printf("error: ", err.Error())
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}
	logger.Println("User logged in successfully")
	c.Status(http.StatusOK)
}
