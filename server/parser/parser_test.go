package parser

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	database "server/DB"
	"server/model"
	"strings"
	"testing"
	"time"

	// Import the MariaDB-compatible driver anonymously
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

func TestConnect(t *testing.T) {
	_ = godotenv.Load()
	fileName := "Game10Season2.dem"
	matchId := 1
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbHost := "127.0.0.1"
	dbPort := "3144"
	dbName := os.Getenv("DB_NAME")
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", dbUser, dbPassword, dbHost, dbPort, dbName)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	database.DB = db
	file, p, err := setupDemoFile(fileName)
	defer file.Close()
	defer p.Close()
	if err != nil {
		t.Errorf("Error Setting up %v", err)
		return
	}
	setup := &model.DemoSetup{MatchId: matchId}
	rounds := 0
	rt := newRoundTracker(setup, &rounds)
	setupMap(p, setup)
	setupTeams(p, setup, rt)
	setUpKillTracking(p, setup, rt)
	setUpSideTracking(p, rt)
	setUpClutchTracking(p, setup, rt)
	setUpFlashTracking(p, setup, rt)
	setUpGrenadeDamageTracking(p, setup, rt)
	setUpDamageTracking(p, rt)
	defer p.Close()

	parseErr := recoverParseToEnd(p)
	if parseErr != nil {
		log.Printf("[parse_demo_stats] Parse error: %v", parseErr)
		log.Printf("[parse_demo_stats] Error contains 'UnexpectedEndOfDemo': %v",
			strings.Contains(parseErr.Error(), "UnexpectedEndOfDemo"))

		if !strings.Contains(parseErr.Error(), "UnexpectedEndOfDemo") {
			log.Printf("[parse_demo_stats] FATAL: Non-EOF error, returning")
		}

		log.Printf("[parse_demo_stats] Caught expected OF, continuing to save collected stats to database")
	}

	t.Logf("HERE IS OUR STRUCT TEAM 1 %v\n", rt.Teams[0].PlayingPlayers)
	t.Log("DONE!")
	t.Logf("HERE IS OUR STRUCT Team 2 %v\n", rt.Teams[1].PlayingPlayers)
}
