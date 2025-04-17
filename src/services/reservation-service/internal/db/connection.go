package db

import (
	"fmt"
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // postgres driver
)

// NewDB initializes a new database connection using sqlx.
func NewDB() *sqlx.DB {
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		os.Getenv("DB_HOST_RESERVATION"),
		os.Getenv("DB_PORT_RESERVATION"),
		os.Getenv("DB_USER_RESERVATION"),
		os.Getenv("DB_PASSWORD_RESERVATION"),
		os.Getenv("DB_NAME_RESERVATION"),
		os.Getenv("DB_SSLMODE_RESERVATION"),
	)
	db, err := sqlx.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	if err = db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("Successfully connected to reservation_db")
	return db
}
