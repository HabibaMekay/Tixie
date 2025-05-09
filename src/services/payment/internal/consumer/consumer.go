package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/paymentintent"
	brokerPkg "tixie.local/broker"
	circuitbreaker "tixie.local/common"
)

type PaymentMessage struct {
	TicketID int   `json:"ticket_id"`
	Amount   int64 `json:"amount"`
}

type PaymentConsumer struct {
	broker        *brokerPkg.Broker
	breaker       *circuitbreaker.Breaker
	numWorkers    int
	prefetchCount int
	processWg     sync.WaitGroup
	ctx           context.Context
	cancel        context.CancelFunc
}

type ConsumerConfig struct {
	NumWorkers    int
	PrefetchCount int
}

func DefaultConsumerConfig() ConsumerConfig {
	return ConsumerConfig{
		NumWorkers:    5,
		PrefetchCount: 10,
	}
}

func NewPaymentConsumer(rabbitmqURL string, config ConsumerConfig) (*PaymentConsumer, error) {
	broker, err := brokerPkg.NewBroker(rabbitmqURL, "payment", "topic")
	if err != nil {
		return nil, fmt.Errorf("failed to create broker: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &PaymentConsumer{
		broker:        broker,
		breaker:       circuitbreaker.NewBreaker("payment-consumer-service"),
		numWorkers:    config.NumWorkers,
		prefetchCount: config.PrefetchCount,
		ctx:           ctx,
		cancel:        cancel,
	}, nil
}

func (c *PaymentConsumer) processMessage(msg amqp.Delivery) {
	defer c.processWg.Done()

	log.Printf("Processing payment message: %s", msg.Body)

	var paymentMsg PaymentMessage
	err := json.Unmarshal(msg.Body, &paymentMsg)
	if err != nil {
		log.Printf("Error unmarshaling message: %v", err)
		msg.Reject(false) // Don't requeue malformed messages
		return
	}

	// Add timeout context for the Stripe API call
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	result := c.breaker.ExecuteContext(ctx, func() (interface{}, error) {
		params := &stripe.PaymentIntentParams{
			Amount:   stripe.Int64(paymentMsg.Amount),
			Currency: stripe.String(string(stripe.CurrencyUSD)),
		}

		pi, err := paymentintent.New(params)
		if err != nil {
			return nil, fmt.Errorf("failed to create payment intent: %v", err)
		}

		log.Printf("Created payment intent for ticket %d: %s", paymentMsg.TicketID, pi.ID)
		return pi, nil
	})

	if result.Error != nil {
		if circuitbreaker.IsCircuitBreakerError(result.Error) {
			log.Printf("Circuit breaker error: %v", result.Error)
			// Requeue the message when circuit breaker is triggered for retry mechanisms to work
			msg.Reject(true)
		} else {
			log.Printf("Error processing payment for ticket %d: %v", paymentMsg.TicketID, result.Error)
			msg.Reject(false)
		}
		return
	}

	log.Printf("Successfully processed payment message for ticket: %d", paymentMsg.TicketID)
	msg.Ack(false) // Acknowledge successful processing
}

func (c *PaymentConsumer) startWorker(messages <-chan amqp.Delivery, workerID int) {
	log.Printf("Starting worker %d", workerID)
	for {
		select {
		case msg, ok := <-messages:
			if !ok {
				log.Printf("Worker %d channel closed", workerID)
				return
			}
			c.processWg.Add(1)
			c.processMessage(msg)
		case <-c.ctx.Done():
			log.Printf("Worker %d shutting down", workerID)
			return
		}
	}
}

func (c *PaymentConsumer) Start() error {
	queueName := "payment_processing"
	err := c.broker.DeclareAndBindQueue(queueName, "topay")
	if err != nil {
		return fmt.Errorf("failed to declare and bind queue: %v", err)
	}

	// Set QoS/prefetch
	err = c.broker.SetQoS(c.prefetchCount, 0, false)
	if err != nil {
		return fmt.Errorf("failed to set QoS: %v", err)
	}

	messages, err := c.broker.Consume(queueName)
	if err != nil {
		return fmt.Errorf("failed to start consuming messages: %v", err)
	}

	// Start worker pool
	for i := 0; i < c.numWorkers; i++ {
		go c.startWorker(messages, i+1)
	}

	log.Printf("Payment consumer started with %d workers. Waiting for messages...", c.numWorkers)

	// Keep the consumer running until a termination signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Payment consumer shutting down...")

	// Initiate graceful shutdown
	c.cancel() // Signal all workers to stop

	// Wait for all in-progress messages to complete with a timeout
	done := make(chan struct{})
	go func() {
		c.processWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("All workers completed gracefully")
	case <-time.After(30 * time.Second):
		log.Println("Shutdown timed out waiting for workers")
	}

	return c.broker.Close()
}

func (c *PaymentConsumer) Close() error {
	c.cancel()
	return c.broker.Close()
}
