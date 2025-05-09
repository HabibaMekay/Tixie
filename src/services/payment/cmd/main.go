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
)

// PaymentMessage represents the structure of incoming payment messages
type PaymentMessage struct {
	TicketID int `json:"ticket_id"`
	Amount   int `json:"amount"`
}

func startMessageConsumer(rabbitmqURL string) {
	broker, err := brokerPkg.NewBroker(rabbitmqURL, "tixie", "topic")
	if err != nil {
		log.Printf("Failed to create broker: %v", err)
		return
	}
	defer broker.Close()

	queueName := "payment_requests"
	err = broker.DeclareAndBindQueue(queueName, "topay")
	if err != nil {
		log.Printf("Failed to declare and bind queue: %v", err)
		return
	}

	messages, err := broker.Consume(queueName)
	if err != nil {
		log.Printf("Failed to start consuming messages: %v", err)
		return
	}

	log.Println("Payment service consumer started. Waiting for messages...")

	go func() {
		for msg := range messages {
			log.Printf("Received payment message: %s", msg.Body)

			var paymentMsg PaymentMessage
			if err := json.Unmarshal(msg.Body, &paymentMsg); err != nil {
				log.Printf("Error unmarshaling payment message: %v", err)
				continue
			}

			// Create payment intent with Stripe
			params := &stripe.PaymentIntentParams{
				Amount:   stripe.Int64(int64(paymentMsg.Amount)),
				Currency: stripe.String(string(stripe.CurrencyUSD)),
			}

			pi, err := paymentintent.New(params)
			if err != nil {
				log.Printf("Error creating payment intent: %v", err)
				continue
			}

			// Publish payment confirmation message
			confirmationMsg := struct {
				TicketID int    `json:"ticket_id"`
				Amount   int    `json:"amount"`
				Status   string `json:"status"`
			}{
				TicketID: paymentMsg.TicketID,
				Amount:   paymentMsg.Amount,
				Status:   "confirmed",
			}

			if err := broker.Publish(confirmationMsg, "payment.confirmed"); err != nil {
				log.Printf("Error publishing confirmation message: %v", err)
				continue
			}

			log.Printf("Successfully processed payment for ticket %d, payment intent ID: %s", paymentMsg.TicketID, pi.ID)
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
