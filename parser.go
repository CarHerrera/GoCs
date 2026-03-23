package main

import (
	"database/sql"
	"fmt"
	"log"

	// Import the MariaDB-compatible driver anonymously
	_ "github.com/go-sql-driver/mysql"
)

func main() {
	// Define connection parameters
	dbUser := "carlos"
	dbPassword := "herrera"
	dbHost := "127.0.0.1"
	dbPort := "3144" // Default MariaDB port
	dbName := "demos"

	// Format the Data Source Name (DSN)
	// The general format is: "user:password@tcp(host:port)/dbname"
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", dbUser, dbPassword, dbHost, dbPort, dbName)

	// Open a database handle (a connection pool is managed internally)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	defer db.Close() // Ensure the database connection is closed when the main function exits

	// Test the connection to the database
	if err := db.Ping(); err != nil {
		log.Fatalf("Error connecting to the database: %v", err)
	}

	fmt.Println("Successfully connected to MariaDB!")

	// Now you can perform database operations (CRUD)
	// Example: Query data, insert rows, etc.
}
