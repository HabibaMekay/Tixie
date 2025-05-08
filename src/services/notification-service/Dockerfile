# Build stage
FROM golang:1.23-alpine AS build

# Match payment-service structure
WORKDIR /src/notification-service

# Copy mod files relative to context
COPY notification-service/go.mod notification-service/go.sum ./

# Copy the shared broker module for the replace directive to resolve
COPY broker /src/broker

# Copy the rest of the notification service source code
COPY notification-service/. .

# Build the binary
RUN go build -o notification-service ./cmd/main.go

# Final stage: lightweight image
FROM alpine:latest

WORKDIR /app
COPY --from=build /src/notification-service/notification-service .

EXPOSE 8080
CMD ["./notification-service"]
