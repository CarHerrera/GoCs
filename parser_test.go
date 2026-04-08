package main

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	// Import the MariaDB-compatible driver anonymously
	_ "github.com/go-sql-driver/mysql"
	ex "github.com/markus-wa/demoinfocs-golang/v5/examples"
	dem "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/msg"
)

type SQLMatch struct {
	MATCHID      int
	DEMO_NAME    string
	SAVED_DATE   string
	PARSED_STATS int
	PARSED_2D    int
}

func TestConnect(t *testing.T) {
	// Define connection parameters
	dbUser := ""
	dbPassword := ""
	dbHost := "127.0.0.1"
	dbPort := "3144" // Default MariaDB port
	dbName := "demos"

	// Format the Data Source Name (DSN)
	// The general format is: "user:password@tcp(host:port)/dbname"
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", dbUser, dbPassword, dbHost, dbPort, dbName)

	// Open a database handle (a connection pool is managed internally)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Errorf("Error opening database: %v", err)
	}
	defer db.Close() // Ensure the database connection is closed when the main function exits
	db.SetMaxOpenConns(10)
	// Test the connection to the database
	if err := db.Ping(); err != nil {
		t.Errorf("Error connecting to the database: %v", err)
	}
	path := "/home/carlos/NAS/CS2DEMOS/"
	fileName := "furia-vs-vitality-m1-mirage.dem"
	demo := path + fileName
	file, err := os.Open(demo)
	if err != nil {
		t.Error("Error opening file")
	}
	p := dem.NewParser(file)
	var (
		mapMetadata ex.Map
		// mapRadarImg image.Image
	)
	defer p.Close()
	defer file.Close()
	p.RegisterNetMessageHandler(func(msg *msg.CSVCMsg_ServerInfo) {
		// Get metadata for the map that the game was played on for coordinate translations
		mapMetadata = ex.GetMapMetadata(msg.GetMapName())

		// Load map overview image
		// mapRadarImg = ex.GetMapRadar(msg.GetMapName())
	})

	p.RegisterEventHandler(func(e events.Kill) {
		x, y := mapMetadata.TranslateScale(e.Killer.Position().X, e.Killer.Position().Y)
		t.Logf("Player:%s killed %s from (%v,%v)", e.Killer.Name, e.Victim.Name, x, y)
	})

	p.ParseToEnd()
	// 	log.Print("DONE")
	// 	return c.Status(20).JSON(resp)
}
