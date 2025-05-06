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
		os.Getenv("DB_VENDOR_HOST"),
		os.Getenv("DB_PORT_VENDOR"),
		os.Getenv("DB_VENDOR_VENDOR"),
		os.Getenv("DB_PASSWORD_VENDOR"),
		os.Getenv("DB_NAME_VENDOR"),
		os.Getenv("DB_SSLMODE_VENDOR"),
	)
	fmt.Println("Connection string:", connStr)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	if err = db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("Successfully connected to vendordb")
	return db
}
