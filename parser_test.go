package main

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	// Import the MariaDB-compatible driver anonymously
	_ "github.com/go-sql-driver/mysql"
	"github.com/golang/geo/r3"
	dem "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
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
	defer p.Close()
	defer file.Close()
	var matchid, parsed2d, rounds int
	if err := db.QueryRow("SELECT MATCHID, PARSED_2D, (TEAM_A_FINAL_SCORE+ TEAM_B_FINAL_SCORE) as ROUND_TOTAL FROM MATCHES WHERE DEMO_NAME = ?", fileName).Scan(&matchid, &parsed2d, &rounds); err != nil {
		t.Errorf("Error with DB %v", err)
	}
	if parsed2d == 1 {
		var me MatchEvents

		me.RoundPositions = make(map[int]RoundEvents)

		for r := range rounds + 1 {
			if r == 0 {
				continue
			}
			var RE RoundEvents
			RE.PlayerPositions = make(map[int64][]PlayerPositions)
			RE.PlayerNames = make(map[int64]string)
			query := `
				SELECT p.PLAYERID, p.PLAYERNAME, re.XPOS, re.YPOS, re.TICK 
				from ROUND_EVENTS as re 
				JOIN PLAYERS p on p.PLAYERID = re.PLAYERID 
				WHERE MATCHID = ? AND re.ROUND_NO = ?
				ORDER BY re.TICK ASC
			`
			rows, err := db.Query(query, matchid, r)
			if err != nil {
				t.Errorf("ERror: %v", err)
			}
			for rows.Next() {
				var Name string
				var tick, x, y, z int
				var playerid int64
				rows.Scan(&playerid, &Name, &x, &y, &z, &tick)
				RE.Tick = tick
				position := r3.Vector{X: float64(x), Y: float64(y), Z: float64(z)}
				if _, ok := RE.PlayerNames[playerid]; !ok {
					RE.PlayerNames[playerid] = Name
				}

				if len(RE.PlayerPositions[playerid]) == 0 {
					RE.PlayerPositions[playerid] = []PlayerPositions{
						{Position: position},
					}
				} else {
					RE.PlayerPositions[playerid] = append(RE.PlayerPositions[playerid], PlayerPositions{Position: position})
				}
			}
		}
	}
	// 	log.Print("DONE")
	// 	return c.Status(20).JSON(resp)
}
