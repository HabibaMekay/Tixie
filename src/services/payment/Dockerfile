# Build stage
FROM golang:1.23-alpine AS build

WORKDIR /src/payment

# Copy only what's needed for dependency resolution
COPY payment/go.mod payment/go.sum ./
COPY broker /src/broker

# Let Go resolve and cache dependencies
RUN go mod download

# Copy the rest of the source code (only payment service code)
COPY payment/. .

# Build the application
RUN go build -o payment-service ./cmd/main.go

# Final stage: minimal image
FROM alpine:latest

WORKDIR /app
COPY --from=build /src/payment/payment-service .

EXPOSE 8088
CMD ["./payment-service"]
