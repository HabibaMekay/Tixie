# Use the official Golang image as a builder
FROM golang:1.23 AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the Go application
RUN go build -o auth-service main.go

# Use a minimal image for the final container
FROM debian:bookworm-slim

# Set working directory
WORKDIR /app

# Copy the binary from the builder
COPY --from=builder /app/auth-service .

# Copy the .env file if needed


# Expose the port the app runs on
EXPOSE 8080

# Run the Go app
CMD ["./auth-service"]
