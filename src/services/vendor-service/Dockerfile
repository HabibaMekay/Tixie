# Build stage
FROM golang:1.23-alpine AS build

WORKDIR /src/vendor

# Copy dependency files first for efficient caching
COPY vendor-service/go.mod vendor-service/go.sum ./

# Copy common module with correct structure
COPY common /src/common

# Download dependencies
RUN go mod download

# Copy the full service source
COPY vendor-service/. .

# Build the binary
RUN go build -o vendor-service ./cmd/main.go

# Final stage: minimal image
FROM alpine:latest

WORKDIR /app
COPY --from=build /src/vendor/vendor-service .

EXPOSE 9060

CMD ["./vendor-service"]
