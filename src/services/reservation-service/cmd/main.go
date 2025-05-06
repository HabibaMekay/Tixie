package main

import (
	"log"
	"net/http"
	"reservation-service/internal/api"
	"reservation-service/internal/db/repos"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type ReservationService struct {
	reservationDB *sqlx.DB
	purchaseRepo  *repos.PurchaseRepository
	ticketClient  *http.Client
}

func NewReservationService() *ReservationService {
	// Connect to reservation_db only
	reservationConnStr := "host=reservation-db port=5432 user=postgres password=postgres dbname=reservation_db sslmode=disable"
	reservationDB, err := sqlx.Connect("postgres", reservationConnStr)
	if err != nil {
		log.Fatal(err)
	}

	purchaseRepo := repos.NewPurchaseRepository(reservationDB)
	ticketClient := &http.Client{Timeout: 10 * time.Second}

	return &ReservationService{
		reservationDB: reservationDB,
		purchaseRepo:  purchaseRepo,
		ticketClient:  ticketClient,
	}
}

func main() {
	service := NewReservationService()
	router := gin.Default()

	// Setup routes using the routes package
	api.SetupRoutes(router, service.purchaseRepo)

	// Start the server
	if err := router.Run(":9081"); err != nil {
		log.Fatal(err)
	}
}
