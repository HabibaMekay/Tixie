FROM golang:1.23

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . .

RUN go build -o event-service ./cmd

EXPOSE 8080

CMD ["./event-service"]


