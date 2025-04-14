package db

import (
	"log"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // <-- this line makes sure the "postgres" driver is registered!
)

// NewDB initializes a new database connection using sqlx.
func NewDB() *sqlx.DB {
	connStr := "host=db port=5432 user=postgres password=postgres dbname=ticket_db sslmode=disable"
	db, err := sqlx.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	if err = db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("Successfully connected to ticket_db")
	return db
}
