module payment

go 1.23.6

require (
	github.com/google/uuid v1.6.0
	github.com/gorilla/mux v1.8.1
	github.com/joho/godotenv v1.5.1
	github.com/rabbitmq/amqp091-go v1.10.0
	github.com/stretchr/testify v1.8.4
	github.com/stripe/stripe-go v70.15.0+incompatible
	tixie.local/broker v0.0.0
	tixie.local/common v0.0.0
	tixie.local/common/brokermsg v0.0.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/objx v0.5.0 // indirect
	golang.org/x/net v0.17.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace tixie.local/broker => ../broker

replace tixie.local/common/brokermsg => ../common/brokermsg

replace tixie.local/common => ../common
