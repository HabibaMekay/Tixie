package mailer

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/mailersend/mailersend-go"
)

type MailerService struct {
	Client     *mailersend.Mailersend
	FromEmail  string
	FromName   string
	TemplateID string
}

func NewMailerService(apiKey, fromName, fromEmail, templateID string) *MailerService {
	return &MailerService{
		Client:     mailersend.NewMailersend(apiKey),
		FromEmail:  fromEmail,
		FromName:   fromName,
		TemplateID: templateID,
	}
}

func (m *MailerService) SendTicketEmail(to, ticketID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	from := mailersend.From{
		Name:  "Tixie",
		Email: os.Getenv("MAILERSEND_EMAIL"),
	}

	recipients := []mailersend.Recipient{
		{
			Email: to,
		},
	}

	personalization := []mailersend.Personalization{
		{
			Email: to,
			Data: map[string]interface{}{
				"ticket_id": ticketID,
			},
		},
	}

	message := m.Client.Email.NewMessage()
	message.SetFrom(from)
	message.SetRecipients(recipients)
	message.SetSubject("Your QR Code Ticket")
	message.SetTemplateID(m.TemplateID)
	message.SetPersonalization(personalization)

	res, err := m.Client.Email.Send(ctx, message)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	log.Println("Email sent. Message ID:", res.Header.Get("X-Message-Id"))
	return nil
}
