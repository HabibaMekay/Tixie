# Use Go 1.23 base image
FROM golang:1.23

# Set the current working directory in the container
WORKDIR /app

# Copy go.mod and go.sum files to the container
COPY go.mod go.sum ./

# Download the dependencies
RUN go mod download

# Copy the entire project files into the container (including the .env file)
COPY . .

# Build the Go application
RUN go build -o payment-service .

# Expose the port that your Go application will run on
EXPOSE 8088

# Command to run the application
CMD ["./payment-service"]
