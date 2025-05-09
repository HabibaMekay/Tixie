module notification-service

go 1.23.6

toolchain go1.24.0

require (
	github.com/mailersend/mailersend-go v1.6.1
	tixie.local/broker v0.0.0
)

require (
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/rabbitmq/amqp091-go v1.10.0 // indirect
)

replace tixie.local/broker => ../broker
