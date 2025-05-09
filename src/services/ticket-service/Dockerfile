# Build stage
FROM golang:1.23-alpine AS builder

# Install git (if you use Go modules fetching private/public repos)
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod and sum
COPY ticket-service/go.mod ticket-service/go.sum ./

# Copy common module
COPY common /common

# Download dependencies
RUN go mod download

# Copy the source code
COPY ticket-service/. .

# Build the Go app
RUN go build -o ticket-service ./cmd/main.go

# Final stage: minimal runtime image
FROM alpine:latest

# Install ca-certificates for HTTPS connections (important for DBs & APIs)
RUN apk add --no-cache ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/ticket-service .

# Expose the port your Gin app runs on
EXPOSE 8082

# Start the service
CMD ["./ticket-service"]
