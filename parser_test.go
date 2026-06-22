package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	// Import the MariaDB-compatible driver anonymously
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
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
	_ = godotenv.Load()
	fileName := "Game5Season2.dem"
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
	// parse_demo_stats(fileName, 2)
	demo := os.Getenv("DEMO_PATH") + fileName
	file, err := os.Open(demo)
	round := 0
	matchStarts := 0
	if err != nil {
		t.Error("Error opening file")
	}
	p := dem.NewParserWithConfig(file, dem.ParserConfig{
		MsgQueueBufferSize:        -1,
		IgnorePacketEntitiesPanic: true,
	})
	p.RegisterNetMessageHandler(func(msg *msg.CSVCMsg_ServerInfo) {
		log.Printf("TIME: %v", msg.TickInterval)
	})
	// p.Logger().SetOutput(io.Discard)
	t.Log("Starting parse")
	p.RegisterEventHandler(func(m events.MatchStartedChanged) {
		t.Log("Match started!")
		round = 1
		matchStarts += 1
	})
	p.RegisterEventHandler(func(k events.Kill) {
		if k.Killer == nil {
			return
		}
		t.Logf("%v got a kill on %v", k.Killer.Name, k.Victim.Name)

	})
	// p.RegisterEventHandler(func(pf events.PlayerFlashed) {
	// 	t.Logf("%v got flashed by %v for %v seconds", pf.Player.Name, pf.Attacker.Name, pf.FlashDuration().Seconds())
	// })
	p.RegisterEventHandler(func(r events.RoundEndOfficial) {
		t.Logf("Round %v ended", round)
		round++
	})
	defer func() {
		if rec := recover(); rec != nil {
			t.Logf("PANIC at round %v: %v", round, rec)
			t.Fail()
		}
	}()
	defer p.Close()
	// var TeamStats [2]Team
	defer file.Close()
	if err := p.ParseToEnd(); err != nil {
		// THIS is the key - log what round we were in when EOF happened
		t.Logf("Parse error at round ~%v: %v, matchrestarts:%v", round, err, matchStarts)
		// Check file size to see if it's suspiciously small
		info, _ := os.Stat(demo)
		t.Logf("File size: %v bytes", info.Size())
	}
	t.Log("Parse completed")
	// 	return c.Status(20).JSON(resp)
}
