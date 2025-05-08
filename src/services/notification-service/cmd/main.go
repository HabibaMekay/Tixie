package main

import (
	"log"
	"os"

	mailer "notification-service/internal/api"
)

func main() {
	apiKey := os.Getenv("MAILERSEND_API_KEY")
	templateID := os.Getenv("MAILERSEND_TEMPLATE_ID")

	svc := mailer.NewMailerService(apiKey, "Ticket Notifier", "you@yourdomain.com", templateID)

	err := svc.SendTicketEmail("recipient@email.com", "abc123xyz")
	if err != nil {
		log.Fatal("Failed to send:", err)
	}
}
