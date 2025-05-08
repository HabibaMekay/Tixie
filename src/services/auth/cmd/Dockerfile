# ---- Build Stage ----
FROM golang:1.23 AS builder

# Disable CGO, set target OS/arch
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64

# Set working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy everything from the build context (auth/)
COPY . .

# Build the binary from cmd/main.go
RUN go build -o main ./cmd/main.go

# ---- Run Stage ----
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/main .

EXPOSE 8080

CMD ["./main"]
