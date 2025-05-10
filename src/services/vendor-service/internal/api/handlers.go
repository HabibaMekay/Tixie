package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"vendor-service/config"
	"vendor-service/internal/db/models"
	"vendor-service/internal/db/repos"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
	circuitbreaker "tixie.local/common"
)

type Handler struct {
	repo                *repos.VendorRepository
	eventServiceBreaker *circuitbreaker.Breaker
}

func NewHandler(repo *repos.VendorRepository) *Handler {
	return &Handler{
		repo:                repo,
		eventServiceBreaker: circuitbreaker.NewBreaker("event-service-client"),
	}
}

func hashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

func (h *Handler) CreateVendor(c *gin.Context) {
	var vendor models.Vendor
	if err := c.ShouldBindJSON(&vendor); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hashedPassword, err := hashPassword(vendor.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}
	vendor.Password = hashedPassword

	err = h.repo.CreateVendor(vendor)
	if err != nil {
		if pgErr, ok := err.(*pq.Error); ok && pgErr.Code == "23505" {
			c.JSON(http.StatusConflict, gin.H{"error": "Vendor name or email already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create vendor"})
		return
	}

	c.Status(http.StatusCreated)
}

func (h *Handler) GetVendors(c *gin.Context) {
	vendors, err := h.repo.GetAllVendors()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve vendors"})
		return
	}
	c.JSON(http.StatusOK, vendors)
}

func (h *Handler) GetVendorByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid vendor ID"})
		return
	}

	vendor, err := h.repo.GetVendorByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vendor not found"})
		return
	}

	c.JSON(http.StatusOK, vendor)
}

func (h *Handler) UpdateVendor(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid vendor ID"})
		return
	}

	var updatedVendor models.Vendor
	if err := c.ShouldBindJSON(&updatedVendor); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if updatedVendor.Password != "" {
		hashedPassword, err := hashPassword(updatedVendor.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
			return
		}
		updatedVendor.Password = hashedPassword
	}

	err = h.repo.UpdateVendor(id, updatedVendor)
	if err != nil {
		if pgErr, ok := err.(*pq.Error); ok && pgErr.Code == "23505" {
			c.JSON(http.StatusConflict, gin.H{"error": "Vendor name or email already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update vendor"})
		return
	}

	c.Status(http.StatusOK)
}

func (h *Handler) DeleteVendor(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid vendor ID"})
		return
	}

	err = h.repo.DeleteVendor(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete vendor"})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) AuthenticateVendor(c *gin.Context) {
	var creds models.Credentials
	if err := c.ShouldBindJSON(&creds); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	valid, err := h.repo.CheckCredentials(creds.Username, creds.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	if !valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	c.Status(http.StatusOK)
}

func (h *Handler) CreateVendorEvent(c *gin.Context) {
	vendorName := c.GetHeader("username")
	if vendorName == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing vendor information"})
		return
	}

	vendorID, err := h.repo.GetVendorIDByName(vendorName)
	if err != nil {
		if err.Error() == "vendor not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Vendor not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get vendor ID"})
		return
	}

	var event models.Event
	if err := c.ShouldBindJSON(&event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event input"})
		return
	}
	event.VendorID = vendorID

	jsonData, err := json.Marshal(event)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error marshalling event data"})
		return
	}

	var resp *http.Response
	result := h.eventServiceBreaker.Execute(func() (interface{}, error) {
		var err error
		resp, err = http.Post(config.AppConfig.EventServiceURL+"/v1", "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			return nil, fmt.Errorf("event service returned status: %d", resp.StatusCode)
		}

		return resp, nil
	})

	if result.Error != nil {
		if circuitbreaker.IsCircuitBreakerError(result.Error) {
			status, msg := circuitbreaker.HandleCircuitBreakerError(result.Error)
			log.Printf("Circuit breaker error when calling event service: %v", result.Error)
			c.JSON(status, gin.H{"error": msg})
			return
		}
		log.Printf("Error calling event service: %v", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create event"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Event created successfully"})
}
