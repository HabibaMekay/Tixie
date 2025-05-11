package api

import (
	"net/http"
	"strconv"
	"vendor-service/internal/db/models"
	"vendor-service/internal/db/repos"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
	brokerPkg "tixie.local/broker"
	circuitbreaker "tixie.local/common"
	brokermsg "tixie.local/common/brokermsg"
)

type Handler struct {
	repo                *repos.VendorRepository
	eventServiceBreaker *circuitbreaker.Breaker
	broker              *brokerPkg.Broker
}

func NewHandler(repo *repos.VendorRepository, broker *brokerPkg.Broker) *Handler {
	return &Handler{
		repo:                repo,
		eventServiceBreaker: circuitbreaker.NewBreaker("event-service-client"),
		broker:              broker,
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
	username := c.GetHeader("username")
	if username == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing username header"})
		return
	}

	vendorID, err := h.repo.GetVendorIDByName(username)
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

	// Create event message
	eventMsg := brokermsg.EventCreatedMessage{
		Name:               event.Name,
		Date:               event.Date,
		Venue:              event.Location,
		TotalTickets:       event.TotalTickets,
		VendorID:           vendorID,
		Price:              event.Price,
		ReservationTimeout: event.ReservationTimeout,
	}

	// Publish event creation message
	err = h.broker.Publish(eventMsg, brokermsg.TopicEventCreated)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create event"})
		return
	}

	c.Status(http.StatusCreated)
}
