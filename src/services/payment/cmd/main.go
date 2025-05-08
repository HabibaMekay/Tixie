package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"payment/routes"

	"github.com/stripe/stripe-go"
)

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

	// Setup router
	r := routes.SetupRouter()

	// Start server
	port := ":8088"
	fmt.Printf("Payment service running on port %s\n", port)
	log.Fatal(http.ListenAndServe(port, r))
}
