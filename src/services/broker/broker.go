package broker

import (
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Broker struct {
	conn     *amqp.Connection
	channel  *amqp.Channel
	exchange string
	url      string
}

func NewBroker(rabbitMQURL, exchange string, exchangeType string) (*Broker, error) {
	conn, err := amqp.Dial(rabbitMQURL)
	if err != nil {
		log.Printf("Failed to connect to RabbitMQ: %v", err)
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		log.Printf("Failed to open channel: %v", err)
		conn.Close()
		return nil, err
	}

	if exchange != "" {
		err = ch.ExchangeDeclare(
			exchange,
			exchangeType,
			true,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			log.Printf("Failed to declare exchange: %v", err)
			ch.Close()
			conn.Close()
			return nil, err
		}
	}

	return &Broker{
		conn:     conn,
		channel:  ch,
		exchange: exchange,
		url:      rabbitMQURL,
	}, nil
}

func (b *Broker) ensureConnection() error {
	if b.conn == nil || b.conn.IsClosed() {
		conn, err := amqp.Dial(b.url)
		if err != nil {
			log.Printf("Failed to reconnect to RabbitMQ: %v", err)
			return err
		}
		b.conn = conn

		b.channel, err = conn.Channel()
		if err != nil {
			log.Printf("Failed to open channel on reconnect: %v", err)
			conn.Close()
			return err
		}
	}
	return nil
}

func (b *Broker) Publish(message interface{}, key string) error {
	if err := b.ensureConnection(); err != nil {
		return err
	}

	body, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal message: %v", err)
		return err
	}

	err = b.channel.Publish(
		b.exchange,
		key,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
	if err != nil {
		log.Printf("Failed to publish message: %v", err)
		return err
	}

	log.Println("Published message:", string(body))
	return nil
}
func (b *Broker) DeclareAndBindQueue(queueName, routingKey string) error {
	if err := b.ensureConnection(); err != nil {
		return err
	}

	_, err := b.channel.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	return b.channel.QueueBind(
		queueName,
		routingKey,
		b.exchange,
		false,
		nil,
	)
}
func (b *Broker) Consume(queueName string) (<-chan amqp.Delivery, error) {
	if err := b.ensureConnection(); err != nil {
		return nil, err
	}

	msgs, err := b.channel.Consume(
		queueName,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Printf("Failed to start consuming: %v", err)
		return nil, err
	}

	return msgs, nil
}

func (b *Broker) Close() error {
	if b.channel != nil {
		if err := b.channel.Close(); err != nil {
			log.Printf("Failed to close channel: %v", err)
			return err
		}
	}
	if b.conn != nil {
		if err := b.conn.Close(); err != nil {
			log.Printf("Failed to close connection: %v", err)
			return err
		}
	}
	return nil
}

// SetQoS sets the prefetch count for the channel
func (b *Broker) SetQoS(prefetchCount int, prefetchSize int, global bool) error {
	if err := b.ensureConnection(); err != nil {
		return err
	}
	return b.channel.Qos(prefetchCount, prefetchSize, global)
}
