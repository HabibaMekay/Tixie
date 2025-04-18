FROM golang:1.23

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . .

RUN go build -o user-service ./cmd/main.go

EXPOSE 8081

CMD ["./user-service"]
