FROM golang:1.23-alpine AS build

WORKDIR /src/event

# Copy dependency files
COPY event-service/go.mod event-service/go.sum ./
COPY common /src/common

# Download dependencies
RUN go mod download

# Copy the service source
COPY event-service/. .

# Build the binary
RUN go build -o event-service ./cmd

# Final stage
FROM alpine:latest

WORKDIR /app
COPY --from=build /src/event/event-service .

EXPOSE 8080

CMD ["./event-service"]


