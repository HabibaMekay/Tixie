FROM golang:1.23

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . .

RUN go build -o vendor-service ./cmd/main.go

EXPOSE 9060

CMD ["./vendor-service"]
