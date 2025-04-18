package db

import (
    "database/sql"
    _ "github.com/lib/pq"
    "log"
)

var DB *sql.DB

func ConnectDB() {
    var err error
    connStr := "host=db-user user=postgres password=password dbname=userdb sslmode=disable"
    DB, err = sql.Open("postgres", connStr)
    if err != nil {
        log.Fatal(err)
    }

    if err = DB.Ping(); err != nil {
        log.Fatal("Cannot connect to DB:", err)
    }

    log.Println("Connected to the DB")
}
