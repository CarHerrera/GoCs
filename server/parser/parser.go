package parser

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	database "server/DB"
	"server/model"
	"strings"
	"sync"

	"github.com/golang/geo/r2"
	dem "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
)

// lookupRosterTeam returns the active roster team a player already belongs
// to, if any.
func lookupRosterTeam(playerID int64) (string, bool) {
	var rosterTeam string
	err := database.DB.QueryRow(
		"SELECT TEAMNAME FROM TEAM_MEMBERS WHERE PLAYERID = ? AND IS_ACTIVE = 1 LIMIT 1",
		playerID,
	).Scan(&rosterTeam)
	return rosterTeam, err == nil
}

// resolveMatchTeam decides what TEAMNAME to write for a player in this
// match. A player who's already on a roster (TEAM_MEMBERS) always keeps
// that team, regardless of what this particular demo's clan tag says —
// this is what stops a solo/premier match with no clan tag from silently
// reassigning a rostered player to a generic "Team 1"/"Team 2" placeholder.
// A brand-new player on a side with no real clan tag is left unassigned
// (nil TEAMNAME) rather than guessed at.
func resolveMatchTeam(playerID int64, side model.Team) (sql.NullString, bool) {
	if rosterTeam, ok := lookupRosterTeam(playerID); ok {
		return sql.NullString{String: rosterTeam, Valid: true}, false
	}
	if side.IsDefaultName {
		return sql.NullString{}, true
	}
	return sql.NullString{String: side.ClanName, Valid: true}, false
}

// resolveTeamDisplayNames computes MATCHES.TEAM_A_NAME/TEAM_B_NAME the same
// way regardless of which parse path (stats or 2D) runs first: if a side had
// no clan tag but at least one of its players is already on a real roster,
// show that roster name instead of the raw "Team 1"/"Team 2" placeholder.
// rt.Teams[i].ClanName itself is left untouched here; resolveMatchTeam/
// UNASSIGNED_MATCH_PLAYERS still need the raw value.
func resolveTeamDisplayNames(rt *model.RoundTracker) [2]string {
	displayNames := [2]string{rt.Teams[0].ClanName, rt.Teams[1].ClanName}
	for i, team := range rt.Teams {
		if !team.IsDefaultName {
			continue
		}
		for _, player := range team.PlayingPlayers {
			if rosterTeam, ok := lookupRosterTeam(player.ID); ok {
				displayNames[i] = rosterTeam
				break
			}
		}
	}
	return displayNames
}

// seedMatchTeams persists the same roster-aware team identity used by the
// stats path (resolveMatchTeam/resolveTeamDisplayNames) so it's available
// regardless of whether the stats tab or the 2D tab is parsed first. It
// INSERT IGNOREs into MATCH_STATS so it never clobbers a fuller row already
// written by ParseDemoStats, and returns the same team-name -> playerid ->
// playername shape TwoDPlayBack's cached-reload path builds from MATCH_STATS,
// so a fresh (never-cached) 2D response carries real team names too.
func seedMatchTeams(matchID int, rt *model.RoundTracker) map[string]map[int64]string {
	teams := make(map[string]map[int64]string)
	displayNames := resolveTeamDisplayNames(rt)
	database.DB.Exec(`UPDATE MATCHES SET TEAM_A_NAME = ?, TEAM_B_NAME = ? WHERE MATCHID = ?`,
		displayNames[0], displayNames[1], matchID)
	for i, team := range rt.Teams {
		for _, player := range team.PlayingPlayers {
			teamName, unassigned := resolveMatchTeam(player.ID, rt.Teams[i])
			database.DB.Exec(
				`INSERT IGNORE INTO MATCH_STATS (MATCHID, PLAYERID, TEAMNAME, SIDE_INDEX) VALUES (?,?,?,?)`,
				matchID, player.ID, teamName, i,
			)
			if unassigned {
				database.DB.Exec(
					`INSERT IGNORE INTO UNASSIGNED_MATCH_PLAYERS (MATCHID, PLAYERID, SIDE_INDEX, DEFAULT_TEAM_NAME)
					 VALUES (?, ?, ?, ?)`,
					matchID, player.ID, i, rt.Teams[i].ClanName,
				)
				continue
			}
			if _, ok := teams[teamName.String]; !ok {
				teams[teamName.String] = make(map[int64]string)
			}
			teams[teamName.String][player.ID] = player.Name
		}
	}
	return teams
}

func recoverParseToEnd(p dem.Parser) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("internal parser panic: %v", r)
		}
	}()
	return p.ParseToEnd()
}

func ParseDemoStats(fileName string, MATCHID int) (model.BaseDemo, error) {
	file, p, err := setupDemoFile(fileName)
	if err != nil {
		return model.BaseDemo{}, err
	}
	defer file.Close()
	setup := &model.DemoSetup{MatchId: MATCHID}
	rounds := 0
	rt := newRoundTracker(setup, &rounds)
	setupMap(p, setup)
	setupTeams(p, setup, rt)
	setupRoundInfo(p, rt)
	setUpBombPlants(p, setup, rt)
	setUpKillTracking(p, setup, rt)
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

		if !strings.Contains(parseErr.Error(), "UnexpectedEndOfDemo") {
			log.Printf("[parse_demo_stats] FATAL: Non-EOF error, returning")
			return model.BaseDemo{}, parseErr
		}

		log.Printf("[parse_demo_stats] Caught expected OF, continuing to save collected stats to database")
	}
	// MATCHES.TEAM_A_NAME/TEAM_B_NAME are a display label for the match
	// overview, resolved the same way regardless of which parse path runs
	// first (see resolveTeamDisplayNames / seedMatchTeams in Parse2D).
	displayNames := resolveTeamDisplayNames(rt)

	log.Printf("[parse_demo_stats] Saving match stats: %s vs %s (%d-%d final)",
		displayNames[0], displayNames[1],
		rt.Teams[0].EndScore, rt.Teams[1].EndScore)
	_, err = database.DB.Exec(`
		UPDATE MATCHES
			SET
				TEAM_A_NAME = ?,TEAM_A_T_SCORE = ?, TEAM_A_CT_SCORE = ?, TEAM_A_FINAL_SCORE = ?,
				TEAM_B_NAME = ?,TEAM_B_T_SCORE = ?, TEAM_B_CT_SCORE = ?, TEAM_B_FINAL_SCORE = ?, MAP = ?
			WHERE
				MATCHID = ?
	`, displayNames[0], rt.Teams[0].TScore, rt.Teams[0].CTScore, rt.Teams[0].EndScore,
		displayNames[1], rt.Teams[1].TScore, rt.Teams[1].CTScore, rt.Teams[1].EndScore, setup.GameMap, MATCHID)

	if err != nil {
		log.Printf("Error: %v", err)
		return model.BaseDemo{}, err
	}
	for i, team := range rt.Teams {
		for _, player := range team.PlayingPlayers {
			teamName, unassigned := resolveMatchTeam(player.ID, rt.Teams[i])
			// ON DUPLICATE KEY UPDATE: Parse2D may have already seeded this
			// (MATCHID, PLAYERID) row via seedMatchTeams (TEAMNAME/SIDE_INDEX
			// only, stats defaulting to 0) if the 2D tab was viewed before the
			// stats tab. Upsert instead of a plain INSERT so that ordering
			// fills in the full stats on top rather than failing on the
			// primary key.
			_, err := database.DB.Exec(`INSERT INTO MATCH_STATS
    (MATCHID, PLAYERID, TEAMNAME, SIDE_INDEX,
     TOTAL_KILLS, TOTAL_DEATHS, TOTAL_ASSISTS, TOTAL_DAMAGE,
     HEADSHOTS, ENTRY_KILLS, ENTRY_DEATHS,
     UTILITY_DAMAGE, HE_DAMAGE, FIRE_DAMAGE,
     ONE_KILL_COUNT, TWO_KILL_COUNT, THREE_KILL_COUNT, FOUR_KILL_COUNT, FIVE_KILL_COUNT,
     TRADED_KILLS, TRADED_DEATHS,
     CLUTCHES_WON, CLUTCHES_COUNT,
     FLASH_ASSISTS, TEAM_FLASHES, ENEMIES_FLASHED)
    VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
    ON DUPLICATE KEY UPDATE
     TEAMNAME=VALUES(TEAMNAME), SIDE_INDEX=VALUES(SIDE_INDEX),
     TOTAL_KILLS=VALUES(TOTAL_KILLS), TOTAL_DEATHS=VALUES(TOTAL_DEATHS),
     TOTAL_ASSISTS=VALUES(TOTAL_ASSISTS), TOTAL_DAMAGE=VALUES(TOTAL_DAMAGE),
     HEADSHOTS=VALUES(HEADSHOTS), ENTRY_KILLS=VALUES(ENTRY_KILLS), ENTRY_DEATHS=VALUES(ENTRY_DEATHS),
     UTILITY_DAMAGE=VALUES(UTILITY_DAMAGE), HE_DAMAGE=VALUES(HE_DAMAGE), FIRE_DAMAGE=VALUES(FIRE_DAMAGE),
     ONE_KILL_COUNT=VALUES(ONE_KILL_COUNT), TWO_KILL_COUNT=VALUES(TWO_KILL_COUNT),
     THREE_KILL_COUNT=VALUES(THREE_KILL_COUNT), FOUR_KILL_COUNT=VALUES(FOUR_KILL_COUNT),
     FIVE_KILL_COUNT=VALUES(FIVE_KILL_COUNT),
     TRADED_KILLS=VALUES(TRADED_KILLS), TRADED_DEATHS=VALUES(TRADED_DEATHS),
     CLUTCHES_WON=VALUES(CLUTCHES_WON), CLUTCHES_COUNT=VALUES(CLUTCHES_COUNT),
     FLASH_ASSISTS=VALUES(FLASH_ASSISTS), TEAM_FLASHES=VALUES(TEAM_FLASHES), ENEMIES_FLASHED=VALUES(ENEMIES_FLASHED)`,
				MATCHID, player.ID, teamName, i,
				player.Stats.Kills, player.Stats.Deaths, player.Stats.Assists, player.Stats.Damage,
				player.Stats.HeadshotKills, player.Stats.EntryKills, player.Stats.EntryDeaths,
				player.Stats.HEDamage+player.Stats.FireDamage, player.Stats.HEDamage, player.Stats.FireDamage,
				player.Stats.OneFragCount, player.Stats.TwoFrags, player.Stats.ThreeFrags, player.Stats.FourFrags, player.Stats.FiveFrags,
				player.Stats.TradeKills, player.Stats.TradedDeaths,
				player.Stats.ClutchesWon, player.Stats.ClutchCount,
				player.Stats.FlashAssists, player.Stats.TeamFlashed, player.Stats.EnemiesFlashed,
			)
			if err != nil {
				return model.BaseDemo{}, err
			}
			if unassigned {
				if _, err := database.DB.Exec(
					`INSERT IGNORE INTO UNASSIGNED_MATCH_PLAYERS (MATCHID, PLAYERID, SIDE_INDEX, DEFAULT_TEAM_NAME)
					 VALUES (?, ?, ?, ?)`,
					MATCHID, player.ID, i, rt.Teams[i].ClanName,
				); err != nil {
					return model.BaseDemo{}, err
				}
			}
		}
	}
	if _, err := database.DB.Exec(`UPDATE MATCHES SET PARSED_STATS = 1 WHERE MATCHID = ?`, MATCHID); err != nil {
		return model.BaseDemo{}, err
	}
	infoSend := model.BaseDemo{
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

func Parse2D(matchid int) model.MatchEvents {
	var filename string
	var matchmap string
	if err := database.DB.QueryRow("SELECT DEMO_NAME, MAP FROM MATCHES WHERE MATCHID = ?", matchid).Scan(&filename, &matchmap); err != nil {
		panic(err)
	}
	file, p, err := setupDemoFile(filename)
	if err != nil {
		panic(err)
	}
	defer p.Close()
	defer file.Close()
	setup := &model.DemoSetup{MatchId: matchid}
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

		pb := posBatch
		eb := eventBatch
		for pb != nil || eb != nil {
			select {
			case batch, ok := <-pb:
				if !ok {
					pb = nil
				} else {
					flushToDB(database.DB, batch)
				}
			case batch, ok := <-eb:
				if !ok {
					eb = nil
				} else {
					eventFlush(database.DB, batch)
				}
			}
		}
		for batch := range grenadeBatch {
			grenadeFlush(database.DB, batch)
		}
		for batch := range fireBatch {
			fireFlush(database.DB, batch)
		}
	}()
	var playback model.MatchEvents
	var RoundPositions map[int]model.RoundInfo
	var FireParticles map[int][]r2.Point
	RoundPositions = make(map[int]model.RoundInfo)
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
			log.Printf("Parse error: %v", parseerr)
			return model.MatchEvents{}
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
	_, err = database.DB.Exec("UPDATE MATCHES SET PARSED_2D = 1 WHERE MATCHID = ?", matchid)

	if err != nil {
		panic(err)
	}
	mapMeta, err := ResolveMapMetadata(matchmap)
	if err != nil {
		panic(err)
	}
	playback.MapMeta = mapMeta
	playback.RoundPositions = RoundPositions[1]
	playback.Rounds = len(RoundPositions)
	// Persist the same roster-aware team identity the stats path uses, so
	// this first-ever 2D response carries real team names and future cached
	// reloads (TwoDPlayBack's MATCH_STATS join) find rows even if the stats
	// tab is never opened for this match.
	playback.Teams = seedMatchTeams(matchid, rt)
	return playback
}
