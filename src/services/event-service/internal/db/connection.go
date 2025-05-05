package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func ConnectDB() *sql.DB {
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		os.Getenv("DB_EVENT_HOST"),
		os.Getenv("DB_EVENT_PORT"),
		os.Getenv("DB_EVENT_USER"),
		os.Getenv("DB_EVENT_PASSWORD"),
		os.Getenv("DB_EVENT_NAME"),
		os.Getenv("DB_EVENT_SSLMODE"),
	)

	fmt.Println("Connection string:", connStr)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to eventdb: %v", err)
	}

	if err = db.Ping(); err != nil {
		log.Fatalf("Failed to ping eventdb: %v", err)
	}

	log.Println("Successfully connected to eventdb")
	return db
}
