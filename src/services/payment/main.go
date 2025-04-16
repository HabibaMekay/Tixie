package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"payment/routes"

	"github.com/joho/godotenv"
	"github.com/stripe/stripe-go"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	stripe.Key = os.Getenv("SECRET_KEY")

	r := routes.SetupRouter()

	fmt.Println("payment service running on port 8088")
	log.Fatal(http.ListenAndServe(":8088", r))
}
