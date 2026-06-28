package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/golang/geo/r2"
	ex "github.com/markus-wa/demoinfocs-golang/v5/examples"
	dem "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
)

func getDemoPath() string {
	return os.Getenv("DEMO_PATH")
}

func recoverParseToEnd(p dem.Parser) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("internal parser panic: %v", r)
		}
	}()
	return p.ParseToEnd()
}

func parse_demo_stats(fileName string, MATCHID int) (BaseDemo, error) {
	file, p, err := setupDemoFile(fileName)
	if err != nil {
		return BaseDemo{}, err
	}
	defer file.Close()
	setup := &DemoSetup{MatchId: MATCHID}
	rounds := 0
	rt := newRoundTracker(setup, &rounds)
	setupMap(p, setup)
	setupTeams(p, setup, rt)
	setupRoundInfo(p, rt)

	setUpKillTracking(p, rt)
	setUpTradeTracking(p, setup, rt)
	setUpClutchTracking(p, setup, rt)
	setUpFlashTracking(p, setup, rt)
	setUpGrenadeDamageTracking(p, setup, rt)
	setUpDamageTracking(p, rt)
	setUpSideTracking(p, rt)
	info, err := file.Stat()
	defer p.Close()
	parseErr := recoverParseToEnd(p)
	if parseErr != nil {
		log.Printf("[parse_demo_stats] Parse error: %v", parseErr)
		log.Printf("[parse_demo_stats] Error contains 'UnexpectedEndOfDemo': %v",
			strings.Contains(parseErr.Error(), "UnexpectedEndOfDemo"))

		// Only return error if it's NOT an EOF error
		if !strings.Contains(parseErr.Error(), "UnexpectedEndOfDemo") {
			log.Printf("[parse_demo_stats] FATAL: Non-EOF error, returning")
			return BaseDemo{}, parseErr
		}

		// If it's just EOF, log it and continue to save the data
		log.Printf("[parse_demo_stats] Caught expected OF, continuing to save collected stats to database")
	}
	log.Printf("[parse_demo_stats] Saving match stats: %s vs %s (%d-%d final)",
		rt.Teams[0].ClanName, rt.Teams[1].ClanName,
		rt.Teams[0].EndScore, rt.Teams[1].EndScore)
	_, err = DB.Exec(`
		UPDATE MATCHES 
			SET
				PARSED_STATS = 1,
				TEAM_A_NAME = ?,TEAM_A_T_SCORE = ?, TEAM_A_CT_SCORE = ?, TEAM_A_FINAL_SCORE = ?,
				TEAM_B_NAME = ?,TEAM_B_T_SCORE = ?, TEAM_B_CT_SCORE = ?, TEAM_B_FINAL_SCORE = ?, MAP = ?
			WHERE 
				MATCHID = ?
	`, rt.Teams[0].ClanName, rt.Teams[0].TScore, rt.Teams[0].CTScore, rt.Teams[0].EndScore,
		rt.Teams[1].ClanName, rt.Teams[1].TScore, rt.Teams[1].CTScore, rt.Teams[1].EndScore, setup.GameMap, MATCHID)

	if err != nil {
		return BaseDemo{}, err
	}
	for i, team := range rt.Teams {
		for _, player := range team.PlayingPlayers {
			_, err := DB.Exec(`INSERT INTO MATCH_STATS 
    (MATCHID, PLAYERID, TEAMNAME,
     TOTAL_KILLS, TOTAL_DEATHS, TOTAL_ASSISTS, TOTAL_DAMAGE,
     HEADSHOTS, ENTRY_KILLS, ENTRY_DEATHS,
     UTILITY_DAMAGE, HE_DAMAGE, FIRE_DAMAGE,
     ONE_KILL_COUNT, TWO_KILL_COUNT, THREE_KILL_COUNT, FOUR_KILL_COUNT, FIVE_KILL_COUNT,
     TRADED_KILLS, TRADED_DEATHS,
     CLUTCHES_WON, CLUTCHES_COUNT,
     FLASH_ASSISTS, TEAM_FLASHES, ENEMIES_FLASHED)
    VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
				MATCHID, player.ID, rt.Teams[i].ClanName,
				player.Stats.Kills, player.Stats.Deaths, player.Stats.Assists, player.Stats.Damage,
				player.Stats.HeadshotKills, player.Stats.EntryKills, player.Stats.EntryDeaths,
				player.Stats.HEDamage+player.Stats.FireDamage, player.Stats.HEDamage, player.Stats.FireDamage,
				player.Stats.OneFragCount, player.Stats.TwoFrags, player.Stats.ThreeFrags, player.Stats.FourFrags, player.Stats.FiveFrags,
				player.Stats.TradeKills, player.Stats.TradedDeaths,
				player.Stats.ClutchesWon, player.Stats.ClutchCount,
				player.Stats.FlashAssists, player.Stats.TeamFlashed, player.Stats.EnemiesFlashed,
			)
			if err != nil {
				return BaseDemo{}, err
			}
		}
	}
	if err != nil {
		return BaseDemo{}, err
	}
	infoSend := BaseDemo{
		FileName:  info.Name(),
		ModDate:   info.ModTime().Local().String(),
		FileSize:  fmt.Sprintf("%.2f", float64(info.Size())/1024.0/1024.0*1.00),
		BaseStats: true,
		Parsed:    false,
		Map:       setup.GameMap,
		TeamStats: *rt.Teams,
	}
	return infoSend, nil
}

func Parse2D(filename string) MatchEvents {

	var matchid int
	var matchmap string
	if err := DB.QueryRow("SELECT MATCHID, MAP FROM MATCHES WHERE DEMO_NAME = ?", filename).Scan(&matchid, &matchmap); err != nil {
		panic(err)
	}
	file, p, err := setupDemoFile(filename)
	if err != nil {
		panic(err)
	}
	defer p.Close()
	defer file.Close()
	setup := &DemoSetup{MatchId: matchid}
	rounds := 0
	rt := newRoundTracker(setup, &rounds)
	setupMap(p, setup)
	setupTeams(p, setup, rt)
	setupRoundInfo(p, rt)
	const size = 500
	var playerBuffer []posEntry
	var grenadeBuffer []GrenadeEntry
	var fireBuffer []FireEntry
	var eventBuffer []EventEntry
	posBatch := make(chan []posEntry, size)
	grenadeBatch := make(chan []GrenadeEntry, size)
	fireBatch := make(chan []FireEntry, size)
	eventBatch := make(chan []EventEntry, size)
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
	var playback MatchEvents
	var RoundPositions map[int]RoundInfo
	var FireParticles map[int][]r2.Point
	RoundPositions = make(map[int]RoundInfo)
	FireParticles = make(map[int][]r2.Point)
	setUpRoundCycle(p, rt, &playerBuffer, &grenadeBuffer, &fireBuffer, &eventBuffer, RoundPositions, FireParticles)
	setUpPositionTracking(p, rt, RoundPositions, &playerBuffer, posBatch)
	setUpFireTracking(p, rt, RoundPositions, FireParticles, &fireBuffer, fireBatch)
	setUpEntityTracking(p, rt, RoundPositions, &grenadeBuffer, grenadeBatch)
	setUpEventTracking(p, rt, RoundPositions, &eventBuffer, eventBatch)
	setUpSideTracking(p, rt)
	log.Printf("STARTING PARSE")

	parseerr := recoverParseToEnd(p)
	if parseerr != nil {
		if !errors.Is(parseerr, dem.ErrUnexpectedEndOfDemo) {
			close(posBatch)
			close(grenadeBatch)
			close(fireBatch)
			close(eventBatch)
			wg.Wait()
			fmt.Errorf("Parse error: %v", parseerr)
			return MatchEvents{} // stop here — don't fall through to flush on closed channels
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
	fmt.Printf("Final Round %d ended. Flushing %d rows to DB...\n", len(RoundPositions), bufferSize)
	_, err = DB.Exec("UPDATE MATCHES SET PARSED_2D = 1 WHERE MATCHID = ?", matchid)

	if err != nil {
		panic(err)
	}
	playback.MapMeta = ex.GetMapMetadata(matchmap)
	playback.RoundPositions = RoundPositions[1]
	playback.Rounds = len(RoundPositions)
	return playback
}
