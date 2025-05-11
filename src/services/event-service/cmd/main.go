package main

import (
	"encoding/json"
	"event-service/internal/api"
	"event-service/internal/db"
	"event-service/internal/db/models"
	"event-service/internal/db/repos"
	"log"

	"github.com/gin-gonic/gin"
	brokerPkg "tixie.local/broker"
	brokermsg "tixie.local/common/brokermsg"
)

func main() {
	dbConn := db.ConnectDB()
	if dbConn == nil {
		log.Fatal("Database connection failed")
	}

	repo := repos.NewEventRepository(dbConn)
	broker, err := brokerPkg.NewBroker("amqp://guest:guest@rabbitmq:5672/", "event-service", "topic")
	if err != nil {
		log.Fatal("Failed to connect to broker:", err)
	}

	// Set up a consumer for event stuff
	queueName := "event_creation"
	err = broker.DeclareAndBindQueue(queueName, brokermsg.TopicEventCreated)
	if err != nil {
		log.Fatal("Failed to set up queue:", err)
	}

	messages, err := broker.Consume(queueName)
	if err != nil {
		log.Fatal("Failed to start consuming:", err)
	}

	// Start consumer goroutine
	go func() {
		for msg := range messages {
			var eventMsg brokermsg.EventCreatedMessage
			if err := json.Unmarshal(msg.Body, &eventMsg); err != nil {
				log.Printf("Error unmarshaling message: %v", err)
				continue
			}

			event := models.Event{
				Name:               eventMsg.Name,
				Date:               eventMsg.Date,
				Venue:              eventMsg.Venue,
				TotalTickets:       eventMsg.TotalTickets,
				VendorID:           eventMsg.VendorID,
				Price:              eventMsg.Price,
				ReservationTimeout: eventMsg.ReservationTimeout,
			}

			if err := repo.CreateEvent(event); err != nil {
				log.Printf("Error creating event: %v", err)
			}
		}
	}()

	r := gin.Default()
	api.SetupRoutes(r, repo)

	log.Println("Event Service running on :8080")
	r.Run(":8080")
}
