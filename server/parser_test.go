package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	// Import the MariaDB-compatible driver anonymously
	_ "github.com/go-sql-driver/mysql"
	"github.com/golang/geo/r2"
	"github.com/joho/godotenv"
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
	_ = godotenv.Load()
	fileName := "Game10Season2.dem"
	matchId := 1
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
	file, p, err := setupDemoFile(fileName)
	defer file.Close()
	defer p.Close()
	if err != nil {
		t.Errorf("Error Setting up %v", err)
		return
	}
	setup := &DemoSetup{MatchId: matchId}
	rounds := 0
	rt := newRoundTracker(setup, &rounds)
	setupMap(p, setup)
	setupTeams(p, setup, rt)
	// setupKillTracking(p, rt)
	setupRoundInfo(p, rt)
	var (
		playerBuffer  []posEntry
		grenadeBuffer []GrenadeEntry
		fireBuffer    []FireEntry
		eventBuffer   []EventEntry
	)
	size := 500
	posBatch := make(chan []posEntry, size)
	grenadeBatch := make(chan []GrenadeEntry, size)
	fireBatch := make(chan []FireEntry, size)
	eventBatch := make(chan []EventEntry, size)
	// var playback MatchEvents
	var RoundPositions map[int]RoundInfo
	var FireParticles map[int][]r2.Point
	FireParticles = make(map[int][]r2.Point)
	RoundPositions = make(map[int]RoundInfo)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		// Process positions and events concurrently (no dependencies)
		openChannels := 2
		for openChannels > 0 {
			select {
			case batch, ok := <-posBatch:
				if !ok {
					openChannels--
				} else {
					flushToDB(DB, batch)
				}
			case batch, ok := <-eventBatch:
				if !ok {
					openChannels--
				} else {
					eventFlush(DB, batch)
				}
			}
		}
		for batch := range grenadeBatch {
			grenadeFlush(DB, batch)
		}
		for batch := range fireBatch {
			fireFlush(DB, batch)
		}
	}()
	setUpRoundCycle(p, rt, &playerBuffer, &grenadeBuffer, &fireBuffer, &eventBuffer, RoundPositions, FireParticles)
	setUpPositionTracking(p, rt, RoundPositions, &playerBuffer, posBatch)
	setUpFireTracking(p, rt, RoundPositions, FireParticles, &fireBuffer, fireBatch)
	setUpEntityTracking(p, rt, RoundPositions, &grenadeBuffer, grenadeBatch)
	setUpEventTracking(p, rt, RoundPositions, &eventBuffer, eventBatch)
	setupSideTracking(p, rt)
	t.Logf("Created round tracker rt %v", rt)
	t.Log("STARTING PARSE")
	parseerr := recoverParseToEnd(p)
	if parseerr != nil {
		if !errors.Is(parseerr, dem.ErrUnexpectedEndOfDemo) {
			close(posBatch)
			close(grenadeBatch)
			close(fireBatch)
			close(eventBatch)
			wg.Wait()
			t.Errorf("Parse error: %v", parseerr)
			return // stop here — don't fall through to flush on closed channels
		}
		log.Printf("[Parse2D] Caught expected EOF, flushing remaining data")
	}
	log.Printf("[Parse2D] Flushing remaining buffers before closing channels")
	bufferSize := len(playerBuffer) + len(grenadeBuffer) + len(fireBuffer) + len(eventBuffer)
	if len(playerBuffer) > 0 {
		batchToSend := make([]posEntry, len(playerBuffer))
		copy(batchToSend, playerBuffer)
		posBatch <- batchToSend
		playerBuffer = playerBuffer[:0]
	}
	if len(grenadeBuffer) > 0 {
		grenadeBatchSend := make([]GrenadeEntry, len(grenadeBuffer))
		copy(grenadeBatchSend, grenadeBuffer)
		grenadeBatch <- grenadeBatchSend
		grenadeBuffer = grenadeBuffer[:0]
	}
	if len(fireBuffer) > 0 {
		fireBatchSend := make([]FireEntry, len(fireBuffer))
		copy(fireBatchSend, fireBuffer)
		fireBatch <- fireBatchSend
		fireBuffer = fireBuffer[:0]
	}
	if len(eventBuffer) > 0 {
		eventBatchSend := make([]EventEntry, len(eventBuffer))
		copy(eventBatchSend, eventBuffer)
		eventBatch <- eventBatchSend
		eventBuffer = eventBuffer[:0]
	}
	close(posBatch)
	close(grenadeBatch)
	close(fireBatch)
	close(eventBatch)
	wg.Wait()
	t.Logf("Final Round %d ended. Flushing %d rows to DB...\n", len(RoundPositions), bufferSize)
	_, err = DB.Exec("UPDATE MATCHES SET PARSED_2D = 1 WHERE MATCHID = ?", matchId)

	if err != nil {
		panic(err)
	}

	// t.Logf("HERE IS OUR STRUCT TEAM 1 %v\n", rt.Teams[0])
	t.Log("DONE!")
	// t.Logf("HERE IS OUR STRUCT Team 2 %v\n", rt.Teams[1])
	// 	return c.Status(20).JSON(resp)
}
