# Build stage
FROM golang:1.23-alpine AS build

WORKDIR /src/reservation-service

# Copy mod files
COPY reservation-service/go.mod reservation-service/go.sum ./
COPY broker /src/broker
COPY common /src/common

# Let Go resolve and cache dependencies
RUN go mod download

# Copy the rest of the source code (only payment service code)
COPY reservation-service/. .


# Build the binary
RUN go build -o reservation-service ./cmd/main.go

# Final stage
FROM alpine:latest

WORKDIR /app
COPY --from=build /src/reservation-service/reservation-service .

EXPOSE 9081
CMD ["./reservation-service"]