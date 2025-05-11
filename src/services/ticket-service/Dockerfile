# Build stage
FROM golang:1.23-alpine AS builder

# Install git (if you use Go modules fetching private/public repos)
RUN apk add --no-cache git

# Set working directory
WORKDIR /src/ticket

# Copy go mod and sum
COPY ticket-service/go.mod ticket-service/go.sum ./

COPY broker /src/broker
COPY common /src/common

# Download dependencies
RUN go mod download

# Copy the source code
COPY ticket-service/. .

# Build the Go app
RUN go build -o ticket-service ./cmd/main.go

# Final stage: minimal runtime image
FROM alpine:latest

RUN apk add --no-cache ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /src/ticket/ticket-service .

# Expose the port your Gin app runs on
EXPOSE 8082

# Start the service
CMD ["./ticket-service"]
