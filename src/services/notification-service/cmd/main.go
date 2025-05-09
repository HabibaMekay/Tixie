package main

import (
	"encoding/json"
	"log"
	mailer "notification-service/internal/api"
	"os"
	"os/signal"
	"syscall"

	brokerPkg "tixie.local/broker"
)

// EmailMessage represents the expected format of incoming email messages
type EmailMessage struct {
	RecipientEmail string `json:"recipient_email"`
	TicketID       string `json:"ticket_id"`
}

func main() {
	apiKey := os.Getenv("MAILERSEND_API_KEY")
	templateID := os.Getenv("MAILERSEND_TEMPLATE_ID")
	fromEmail := os.Getenv("MAILERSEND_EMAIL")
	rabbitmqURL := os.Getenv("RABBITMQ_URL")

	if apiKey == "" || templateID == "" || fromEmail == "" {
		log.Fatal("Required environment variables are not set: MAILERSEND_API_KEY, MAILERSEND_TEMPLATE_ID, MAILERSEND_EMAIL")
	}

	mailerService := mailer.NewMailerService(apiKey, "Ticket Notifier", fromEmail, templateID)

	b, err := brokerPkg.NewBroker(rabbitmqURL, "notification", "topic")
	if err != nil {
		log.Fatalf("Failed to create broker: %v", err)
	}
	defer b.Close()

	queueName := "email_notifications"
	err = b.DeclareAndBindQueue(queueName, "email")
	if err != nil {
		log.Fatalf("Failed to declare and bind queue: %v", err)
	}

	messages, err := b.Consume(queueName)
	if err != nil {
		log.Fatalf("Failed to start consuming messages: %v", err)
	}

	go func() {
		for msg := range messages {
			log.Printf("Received message: %s", msg.Body)

			var emailMsg EmailMessage
			err := json.Unmarshal(msg.Body, &emailMsg)
			if err != nil {
				log.Printf("Error unmarshaling message: %v", err)
				continue
			}

			err = mailerService.SendTicketEmail(emailMsg.RecipientEmail, emailMsg.TicketID)
			if err != nil {
				log.Printf("Error sending email: %v", err)
				continue
			}

			log.Printf("Successfully processed message for recipient: %s", emailMsg.RecipientEmail)
		}
	}()

	log.Println("Notification service started. Waiting for messages...")

	// Keeps the application running until a termination signal is sent, which is never :shrug:
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Notification service shutting down...")
}
