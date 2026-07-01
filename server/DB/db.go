// internal/database/database.go
package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var DB *sql.DB

func Connect() {
	// Use your environment variables here!

	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	// log.Printf("%s %s", dbUser, dbPassword)
	dbHost := "127.0.0.1"
	dbPort := "3144" // Default MariaDB port
	dbName := os.Getenv("DB_NAME")
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", dbUser, dbPassword, dbHost, dbPort, dbName)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}

	// Important: Configure the pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	DB = db
}
