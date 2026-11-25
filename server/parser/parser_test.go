package parser

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	database "server/DB"
	"server/auth"
	"strings"
	"testing"
	"time"

	// Import the MariaDB-compatible driver anonymously
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	dem "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
)

func TestConnect(t *testing.T) {
	_ = godotenv.Load()
	fileName := "Game12Season3.dem"
	// matchId := 11
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbHost := "127.0.0.1"
	dbPort := "3144"
	dbName := os.Getenv("DB_NAME")
	log.Printf("%s %s %s", dbUser, dbPassword, dbName)
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
	file, err := os.Open(auth.GetDemoPath() + fileName)
	p := dem.NewParserWithConfig(file, dem.ParserConfig{
		MsgQueueBufferSize:        0,
		IgnorePacketEntitiesPanic: true,
	})
	if err != nil {
		t.Errorf("Error Setting up %v", err)
		return
	}
	defer file.Close()
	p.RegisterEventHandler(func(e events.RoundStart) {
		log.Printf("Round Started!")
	})
	p.RegisterEventHandler(func(e events.RoundEnd) {
		log.Print("Round Ended!")
	})
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

	// t.Logf("HERE IS OUR STRUCT TEAM 1 %v\n", rt.Teams[0].PlayingPlayers)
	t.Log("DONE!")
	// t.Logf("HERE IS OUR STRUCT Team 2 %v\n", rt.Teams[1].PlayingPlayers)
}
