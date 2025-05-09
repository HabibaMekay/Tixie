package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"payment/routes"
	"syscall"

	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/paymentintent"
	brokerPkg "tixie.local/broker"
	brokermsg "tixie.local/common/brokermsg"
)

// PaymentMessage represents the structure of incoming payment messages
type PaymentMessage struct {
	TicketID int `json:"ticket_id"`
	Amount   int `json:"amount"`
}

func startMessageConsumer(rabbitmqURL string) {
	// Create broker instance
	broker, err := brokerPkg.NewBroker(rabbitmqURL, "payment-service", "topic")
	if err != nil {
		log.Printf("Failed to create broker: %v", err)
		return
	}
	defer broker.Close()

	// Set up queue for reservation completed events
	queueName := "payment_reservation_completed"
	err = broker.DeclareAndBindQueue(queueName, brokermsg.TopicReservationCompleted)
	if err != nil {
		log.Printf("Failed to declare and bind queue: %v", err)
		return
	}

	// Start consuming messages
	messages, err := broker.Consume(queueName)
	if err != nil {
		log.Printf("Failed to start consuming messages: %v", err)
		return
	}

	log.Println("Payment service consumer started. Listening for reservation completion events...")

	// Process messages
	go func() {
		for msg := range messages {
			log.Printf("Received reservation completion message: %s", msg.Body)

			// Unmarshal the message
			var reservationMsg brokermsg.ReservationCompletedMessage
			if err := json.Unmarshal(msg.Body, &reservationMsg); err != nil {
				log.Printf("Error unmarshaling reservation message: %v", err)
				continue
			}

			// Create payment intent with Stripe
			params := &stripe.PaymentIntentParams{
				Amount:   stripe.Int64(int64(reservationMsg.Amount)),
				Currency: stripe.String(string(stripe.CurrencyUSD)),
				Metadata: map[string]string{
					"reservation_id": fmt.Sprintf("%d", reservationMsg.ReservationID),
					"event_id":       fmt.Sprintf("%d", reservationMsg.EventID),
					"user_id":        fmt.Sprintf("%d", reservationMsg.UserID),
				},
			}

			pi, err := paymentintent.New(params)
			if err != nil {
				log.Printf("Error creating payment intent: %v", err)

				// Publish payment failed message
				failedMsg := brokermsg.PaymentFailedMessage{
					ReservationID: reservationMsg.ReservationID,
					Reason:        fmt.Sprintf("Failed to create payment: %v", err),
				}

				if pubErr := broker.Publish(failedMsg, brokermsg.TopicPaymentFailed); pubErr != nil {
					log.Printf("Error publishing payment failure message: %v", pubErr)
				}
				continue
			}

			// Publish payment processed message
			processedMsg := brokermsg.PaymentProcessedMessage{
				ReservationID: reservationMsg.ReservationID,
				Amount:        reservationMsg.Amount,
				PaymentID:     pi.ID,
			}

			if err := broker.Publish(processedMsg, brokermsg.TopicPaymentProcessed); err != nil {
				log.Printf("Error publishing payment processed message: %v", err)
				continue
			}

			log.Printf("Successfully processed payment for reservation %d, payment intent ID: %s",
				reservationMsg.ReservationID, pi.ID)
		}
	}()
}

func main() {
	// Initialize Stripe
	stripeKey := os.Getenv("SECRET_KEY")
	if stripeKey == "" {
		log.Fatal("SECRET_KEY environment variable is required")
	}
	stripe.Key = stripeKey

	// Initialize RabbitMQ URL
	rabbitmqURL := os.Getenv("RABBITMQ_URL")
	if rabbitmqURL == "" {
		rabbitmqURL = "amqp://guest:guest@rabbitmq:5672/"
		log.Printf("RABBITMQ_URL not set, using default: %s", rabbitmqURL)
	}
	os.Setenv("RABBITMQ_URL", rabbitmqURL)

	// Start message consumer
	startMessageConsumer(rabbitmqURL)

	// Setup router
	r := routes.SetupRouter()

	// Handle graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("Shutting down server...")
		os.Exit(0)
	}()

	// Start server
	port := ":8088"
	fmt.Printf("Payment service running on port %s\n", port)
	log.Fatal(http.ListenAndServe(port, r))
}
