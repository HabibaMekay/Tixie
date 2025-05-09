# Build stage
FROM golang:1.23-alpine AS build

WORKDIR /src/user

# Copy only what's needed for dependency resolution
COPY user-service/go.mod user-service/go.sum ./

# Copy common module with correct structure
COPY common /src/common

# Download dependencies
RUN go mod download

# Copy the rest of the service source code
COPY user-service/. .

# Build the application binary
RUN go build -o user-service ./cmd/main.go

# Final stage: minimal image
FROM alpine:latest

WORKDIR /app
COPY --from=build /src/user/user-service .

EXPOSE 8081

CMD ["./user-service"]
