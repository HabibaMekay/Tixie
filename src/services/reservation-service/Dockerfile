FROM golang:1.23
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o reservation-service ./cmd/main.go
CMD ["./reservation-service"]