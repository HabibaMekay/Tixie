package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"reservation-service/internal/api"
	"reservation-service/internal/db/repos"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	brokerPkg "tixie.local/broker"
)

type ReservationService struct {
	reservationDB *sqlx.DB
	purchaseRepo  *repos.PurchaseRepository
	reserveRepo   *repos.ReservationRepository
	ticketClient  *http.Client
	broker        *brokerPkg.Broker
}

// PaymentMessage represents the structure of payment confirmation messages
type PaymentMessage struct {
	TicketID int     `json:"ticket_id"`
	Amount   float64 `json:"amount"`
}

func NewReservationService() *ReservationService {
	// Connect to reservation_db only
	reservationConnStr := "host=reservation-db port=5432 user=postgres password=postgres dbname=reservation_db sslmode=disable"
	reservationDB, err := sqlx.Connect("postgres", reservationConnStr)
	if err != nil {
		log.Fatal(err)
	}

	purchaseRepo := repos.NewPurchaseRepository(reservationDB)
	reserveRepo := repos.NewReservationRepository(reservationDB)
	ticketClient := &http.Client{Timeout: 10 * time.Second}

	// Initialize broker
	broker, err := brokerPkg.NewBroker(os.Getenv("RABBITMQ_URL"), "tixie", "topic")
	if err != nil {
		log.Printf("Warning: Failed to create broker: %v", err)
	}

	return &ReservationService{
		reservationDB: reservationDB,
		purchaseRepo:  purchaseRepo,
		reserveRepo:   reserveRepo,
		ticketClient:  ticketClient,
		broker:        broker,
	}
}

// Start a background job to clean up expired reservations
func (s *ReservationService) startReservationCleanup() {
	// Run cleanup every minute
	ticker := time.NewTicker(1 * time.Minute)

	go func() {
		for range ticker.C {
			log.Println("Running expired reservation cleanup job...")

			// Get all expired reservations
			reservations, err := s.reserveRepo.GetExpiredReservations()
			if err != nil {
				log.Printf("Error getting expired reservations: %v", err)
				continue
			}

			log.Printf("Found %d expired reservations to clean up", len(reservations))

			// Process each expired reservation
			processed := 0
			for _, reservation := range reservations {
				// Call the event service to release the reservation
				resp, err := http.Post(
					fmt.Sprintf("%s/v1/%d/release-reservation", os.Getenv("EVENT_SERVICE_URL"), reservation.EventID),
					"application/json",
					nil,
				)
				if err != nil {
					log.Printf("Error releasing reservation for event %d: %v", reservation.EventID, err)
					continue
				}
				resp.Body.Close()

				// Mark the reservation as expired
				err = s.reserveRepo.ExpireReservation(reservation.ID)
				if err != nil {
					log.Printf("Error marking reservation %d as expired: %v", reservation.ID, err)
					continue
				}
				processed++
			}

			log.Printf("Completed cleanup: processed %d of %d expired reservations", processed, len(reservations))
		}
	}()
}

func (s *ReservationService) startMessageConsumer() {
	if s.broker == nil {
		log.Println("Broker not initialized, skipping message consumer")
		return
	}

	// Declare and bind queue for payment confirmations
	queueName := "reservation_payments"
	err := s.broker.DeclareAndBindQueue(queueName, "payment.confirmed")
	if err != nil {
		log.Printf("Failed to declare and bind queue: %v", err)
		return
	}

	// Start consuming messages
	messages, err := s.broker.Consume(queueName)
	if err != nil {
		log.Printf("Failed to start consuming messages: %v", err)
		return
	}

	go func() {
		for msg := range messages {
			log.Printf("Received payment confirmation message: %s", msg.Body)

			var paymentMsg PaymentMessage
			if err := json.Unmarshal(msg.Body, &paymentMsg); err != nil {
				log.Printf("Error unmarshaling payment message: %v", err)
				continue
			}

			// Get user email and ticket details from the database
			purchase, err := s.purchaseRepo.GetPurchaseByTicketID(paymentMsg.TicketID)
			if err != nil {
				log.Printf("Error fetching purchase details: %v", err)
				continue
			}

			// Get user details to get email
			userResp, err := s.ticketClient.Get(fmt.Sprintf("%s/v1/%d", os.Getenv("USER_SERVICE_URL"), purchase.UserID))
			if err != nil {
				log.Printf("Error fetching user details: %v", err)
				continue
			}
			defer userResp.Body.Close()

			var userDetails struct {
				Email string `json:"email"`
			}
			if err := json.NewDecoder(userResp.Body).Decode(&userDetails); err != nil {
				log.Printf("Error parsing user response: %v", err)
				continue
			}

			// Get ticket details
			ticketResp, err := s.ticketClient.Get(fmt.Sprintf("%s/v1/%d", os.Getenv("TICKET_SERVICE_URL"), paymentMsg.TicketID))
			if err != nil {
				log.Printf("Error fetching ticket details: %v", err)
				continue
			}
			defer ticketResp.Body.Close()

			var ticketDetails struct {
				TicketCode string `json:"ticket_code"`
			}
			if err := json.NewDecoder(ticketResp.Body).Decode(&ticketDetails); err != nil {
				log.Printf("Error parsing ticket response: %v", err)
				continue
			}

			// Publish notification message
			notificationMsg := struct {
				RecipientEmail string `json:"recipient_email"`
				TicketID       string `json:"ticket_id"`
			}{
				RecipientEmail: userDetails.Email,
				TicketID:       ticketDetails.TicketCode,
			}

			if err := s.broker.Publish(notificationMsg, "email"); err != nil {
				log.Printf("Error publishing notification message: %v", err)
				continue
			}

			log.Printf("Successfully processed payment confirmation for ticket %d", paymentMsg.TicketID)
		}
	}()

	log.Println("Message consumer started successfully")
}

func main() {
	service := NewReservationService()

	// Start the message consumer
	service.startMessageConsumer()

	// Start the reservation cleanup job
	service.startReservationCleanup()

	router := gin.Default()

	// Setup routes using the routes package
	api.SetupRoutes(router, service.purchaseRepo, service.reserveRepo)

	// Handle graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("Shutting down server...")
		if service.broker != nil {
			if err := service.broker.Close(); err != nil {
				log.Printf("Error closing broker: %v", err)
			}
		}
		if err := service.reservationDB.Close(); err != nil {
			log.Printf("Error closing database connection: %v", err)
		}
		os.Exit(0)
	}()

	// Start the server
	if err := router.Run(":9081"); err != nil {
		log.Fatal(err)
	}
}
