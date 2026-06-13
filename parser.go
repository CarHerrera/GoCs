package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"slices"
	"sync"

	"github.com/golang/geo/r2"
	"github.com/golang/geo/r3"
	ex "github.com/markus-wa/demoinfocs-golang/v5/examples"
	dem "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/msg"
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
	demo := getDemoPath() + fileName
	file, err := os.Open(demo)
	if err != nil {
		return BaseDemo{}, err
	}
	defer file.Close()
	info, err := file.Stat()
	var TeamStats [2]Team
	lrth := false
	catch := true
	p := dem.NewParser(file)
	defer p.Close()
	var GameMap string
	live := false
	p.RegisterNetMessageHandler(func(msg *msg.CSVCMsg_ServerInfo) {
		GameMap = *msg.MapName
	})
	p.RegisterEventHandler(func(e events.MatchStartedChanged) error {
		GS := p.GameState()
		ctside := GS.TeamCounterTerrorists()
		tside := GS.TeamTerrorists()
		var teamname string
		// start = true
		if GS.GamePhase() == common.GamePhaseStartGamePhase {
			live = true
			for _, player := range tside.Members() {
				team1Name := tside.ClanName()
				if team1Name == "" {
					team1Name = "Team 1"
				}
				if !TeamStats[0].inited {
					TeamStats[0] = Team{
						ID:             tside.ID(),
						EndScore:       -1,
						CTScore:        0,
						TScore:         0,
						ClanName:       team1Name,
						PlayingPlayers: make(map[string]Player),
						inited:         true,
					}
					if err := DB.QueryRow("SELECT TEAMNAME FROM TEAMS WHERE TEAMNAME = ?", team1Name).Scan(&teamname); err != nil {
						if err == sql.ErrNoRows {
							_, err := DB.Exec("INSERT INTO TEAMS (TEAMNAME) VALUES (?)", team1Name)
							if err != nil {
								return err
							}
						} else {
							return err
						}
					}
				}
				TeamStats[0].PlayingPlayers[player.Name] = Player{
					Name: player.Name,
					ID:   int64(player.SteamID64),
					Stats: PlayerStats{
						Kills:   0,
						Assists: 0,
						Deaths:  0,
					},
				}
			}
			for _, player := range ctside.Members() {
				team1Name := ctside.ClanName()
				if team1Name == "" {
					team1Name = "Team 2"
				}
				if !TeamStats[1].inited {
					TeamStats[1] = Team{
						ID:             ctside.ID(),
						EndScore:       -1,
						CTScore:        0,
						TScore:         0,
						ClanName:       team1Name,
						PlayingPlayers: make(map[string]Player),
						inited:         true,
					}
					if err := DB.QueryRow("SELECT TEAMNAME FROM TEAMS WHERE TEAMNAME = ?", team1Name).Scan(&teamname); err != nil {
						if err == sql.ErrNoRows {
							_, err := DB.Exec("INSERT INTO TEAMS (TEAMNAME) VALUES (?)", team1Name)
							if err != nil {
								return err
							}
						} else {
							return err
						}
					}
				}
				TeamStats[1].PlayingPlayers[player.Name] = Player{
					Name: player.Name,
					ID:   int64(player.SteamID64),
					Stats: PlayerStats{
						Kills:   0,
						Assists: 0,
						Deaths:  0,
					},
				}

			}
		}
		return nil
	})
	// Included the following 3 to help debug why trackers weren't working.
	p.RegisterEventHandler(func(h events.TeamSideSwitch) {
		if !live {
			return
		}
		lrth = false
		//
		temp := TeamStats[0].ID
		TeamStats[0].ID = TeamStats[1].ID
		TeamStats[1].ID = temp
		catch = true
	})
	p.RegisterEventHandler(func(lr events.AnnouncementLastRoundHalf) {
		//
		lrth = true
	})
	p.RegisterEventHandler(func(r events.RoundEnd) {
		//
		if lrth {
			catch = false
		}
	})
	p.RegisterEventHandler(func(kill events.Kill) {
		if !live {
			return
		}
		killer := kill.Killer
		asssiter := kill.Assister
		victim := kill.Victim
		if killer != nil && killer.Name != victim.Name {
			team := killer.TeamState
			if team.ID() == TeamStats[0].ID {
				p, _ := TeamStats[0].PlayingPlayers[killer.Name]
				p.Stats.Kills++
				TeamStats[0].PlayingPlayers[killer.Name] = p
			} else {
				p, _ := TeamStats[1].PlayingPlayers[killer.Name]
				p.Stats.Kills++
				TeamStats[1].PlayingPlayers[killer.Name] = p
			}
		}
		if asssiter != nil {
			team := asssiter.TeamState
			if team.ID() == TeamStats[0].ID {
				p, _ := TeamStats[0].PlayingPlayers[asssiter.Name]
				p.Stats.Assists++
				TeamStats[0].PlayingPlayers[asssiter.Name] = p
			} else {
				p, _ := TeamStats[1].PlayingPlayers[asssiter.Name]
				p.Stats.Assists++
				TeamStats[1].PlayingPlayers[asssiter.Name] = p
			}
		}
		if victim != nil {
			team := victim.TeamState
			if team.ID() == TeamStats[0].ID {
				p, _ := TeamStats[0].PlayingPlayers[victim.Name]
				p.Stats.Deaths++
				TeamStats[0].PlayingPlayers[victim.Name] = p
			} else {
				p, _ := TeamStats[1].PlayingPlayers[victim.Name]
				p.Stats.Deaths++
				TeamStats[1].PlayingPlayers[victim.Name] = p
			}
		}
	})
	p.RegisterEventHandler(func(score events.ScoreUpdated) {
		if !live {
			return
		}
		team1 := score.TeamState
		// Check to make sure it isn't null
		if TeamStats[0].inited && catch {
			// team1 (non opp) will always have the score incremented
			//

			if TeamStats[0].ID == team1.ID() {
				TeamStats[0].EndScore = score.NewScore
				if team1.Team() == common.TeamCounterTerrorists {
					TeamStats[0].CTScore += 1
				} else {
					TeamStats[0].TScore += 1
				}
			} else {
				TeamStats[1].EndScore = score.NewScore
				if team1.Team() == common.TeamCounterTerrorists {
					TeamStats[1].CTScore++
				} else {
					TeamStats[1].TScore++
				}
			}
			//
			//
		}
	})
	if err := recoverParseToEnd(p); err != nil {
		return BaseDemo{}, err
	}
	_, err = DB.Exec(`
		UPDATE MATCHES 
			SET
				PARSED_STATS = 1,
				TEAM_A_NAME = ?,TEAM_A_T_SCORE = ?, TEAM_A_CT_SCORE = ?, TEAM_A_FINAL_SCORE = ?,
				TEAM_B_NAME = ?,TEAM_B_T_SCORE = ?, TEAM_B_CT_SCORE = ?, TEAM_B_FINAL_SCORE = ?, MAP = ?
			WHERE 
				MATCHID = ?
	`, TeamStats[0].ClanName, TeamStats[0].TScore, TeamStats[0].CTScore, TeamStats[0].EndScore,
		TeamStats[1].ClanName, TeamStats[1].TScore, TeamStats[1].CTScore, TeamStats[1].EndScore, GameMap, MATCHID)

	if err != nil {
		return BaseDemo{}, err
	}
	for i, team := range TeamStats {
		for _, player := range team.PlayingPlayers {
			var steamid int
			if err := DB.QueryRow("SELECT PLAYERID FROM PLAYERS WHERE PLAYERID = ?", player.ID).Scan(&steamid); err != nil {
				if err == sql.ErrNoRows {
					_, err := DB.Exec("INSERT INTO PLAYERS (PLAYERID,PLAYERNAME,TEAMNAME) VALUES (?,?,?)", player.ID, player.Name, TeamStats[i].ClanName)
					if err != nil {
						return BaseDemo{}, err
					}
				} else {
					return BaseDemo{}, err
				}
			}

			_, err := DB.Exec("INSERT INTO MATCH_STATS (MATCHID,PLAYERID,TOTAL_KILLS,TOTAL_DEATHS,TOTAL_ASSISTS) VALUES (?,?,?,?,?)", MATCHID, player.ID, player.Stats.Kills, player.Stats.Deaths, player.Stats.Assists)
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
		Map:       GameMap,
		TeamStats: TeamStats,
	}
	return infoSend, nil
}

func Parse2D(filename string) MatchEvents {
	demo := getDemoPath() + filename
	file, err := os.Open(demo)
	if err != nil {
		panic(err)
	}
	p := dem.NewParserWithConfig(file, dem.ParserConfig{
		MsgQueueBufferSize:        0,
		IgnorePacketEntitiesPanic: true,
	})
	defer p.Close()
	defer file.Close()
	var matchid int
	var matchmap string
	if err := DB.QueryRow("SELECT MATCHID, MAP FROM MATCHES WHERE DEMO_NAME = ?", filename).Scan(&matchid, &matchmap); err != nil {
		panic(err)
	}
	const batchSize = 5000
	const size = 500
	var playerBuffer []posEntry
	var grenadeBuffer []GrenadeEntry
	var fireBuffer []FireEntry
	var eventBuffer []EventEntry
	posBatch := make(chan []posEntry, size)
	grenadeBatch := make(chan []GrenadeEntry, size)
	fireBatch := make(chan []FireEntry, size)
	eventBatch := make(chan []EventEntry, size)
	insertedRounds := make(map[int]bool)
	insertedParticipants := make(map[string]bool)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		// Process grenades first (they have no dependencies)
		for batch := range grenadeBatch {
			grenadeFlush(DB, batch)
		}
		// Then process fire (depends on grenades being in DB)
		for batch := range fireBatch {
			fireFlush(DB, batch)
		}
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
	}()
	var playback MatchEvents
	var RoundPositions map[int]RoundInfo
	var FireParticles map[int][]r2.Point
	live := false
	rounds := 0
	log.Printf("STARTING PARSE")
	p.RegisterEventHandler(func(e events.MatchStartedChanged) {
		if rounds > 1 {
			return
		}
		live = true
		rounds = 1
		RoundPositions = make(map[int]RoundInfo)
		FireParticles = make(map[int][]r2.Point)
		playerBuffer = playerBuffer[:0]
		grenadeBuffer = grenadeBuffer[:0]
		fireBuffer = fireBuffer[:0]
		eventBuffer = eventBuffer[:0]
		// playback = MatchEvents{Teams: make(map[string]map[int64]string)}
		playback.Teams = make(map[string]map[int64]string)
		gs := p.GameState()
		if !insertedRounds[rounds] {
			_, err := DB.Exec("INSERT IGNORE INTO ROUNDS (MATCHID, ROUND_NO) VALUES (?,?)", matchid, rounds)
			if err != nil {
				panic(err)
			}
			insertedRounds[rounds] = true
		}
		for _, player := range gs.Participants().Playing() {
			name := player.TeamState.ClanName()
			if _, ok := playback.Teams[name]; !ok {
				playback.Teams[name] = make(map[int64]string)
			}
			playback.Teams[name][int64(player.SteamID64)] = player.Name
			key := fmt.Sprintf("%d_%d_%d", matchid, rounds, player.SteamID64)
			if !insertedParticipants[key] {
				_, err = DB.Exec("INSERT IGNORE INTO ROUND_PARTICIPANTS (MATCHID, ROUND_NO, PLAYERID, SIDE) VALUES (?,?,?,?)", matchid, rounds, player.SteamID64, int(player.GetTeam()))
				if err != nil {
					panic(err)
				}
				insertedParticipants[key] = true
			}
		}
		RoundPositions[rounds] = RoundInfo{
			PlayerPositions: make(map[int]map[int64]PlayerState),
			PlayerNames:     make(map[int64]PlayerInfo),
			GrenadeEvents:   make(map[int]map[int]GrenadeState),
			FirePositions:   make(map[int]map[int]FireState),
			RoundTimeline:   make(map[int]RoundEvent),
		}
	})
	// p.RegisterEventHandler(func(events.M))
	p.RegisterEventHandler(func(events.FrameDone) {
		if live {
			gs := p.GameState()
			tick := gs.IngameTick()
			flames := gs.Infernos()
			if tick%4 != 0 {
				return
			}
			for _, player := range gs.Participants().Playing() {
				if player.IsAlive() {
					sqlEntry, playerState := get_pos_entry(player, tick, rounds, matchid, isMoving)
					playerBuffer = append(playerBuffer, sqlEntry)
					// Check to see if player is added
					if _, ok := RoundPositions[rounds].PlayerNames[int64(player.SteamID64)]; !ok {
						RoundPositions[rounds].PlayerNames[int64(player.SteamID64)] = PlayerInfo{Name: player.Name, Side: int(player.GetTeam())}
					}
					// log.Printf("%v", player.Weapons())
					if len(RoundPositions[rounds].PlayerPositions[tick]) == 0 {
						RoundPositions[rounds].PlayerPositions[tick] = make(map[int64]PlayerState)

					}
					RoundPositions[rounds].PlayerPositions[tick][int64(player.SteamID64)] = playerState
				}

			}
			if len(flames) != 0 {
				for key, inf := range flames {
					lastState := FireParticles[key]
					if slices.Equal(lastState, inf.Fires().Active().ConvexHull2D()) {
						return
					} else {
						if len(RoundPositions[rounds].FirePositions[tick]) == 0 {
							RoundPositions[rounds].FirePositions[tick] = make(map[int]FireState)
						}
						if len(inf.Fires().Active().ConvexHull2D()) == 0 {
							RoundPositions[rounds].FirePositions[tick][key] = FireState{
								Vertices: inf.Fires().ConvexHull2D(), Status: "ENDING",
							}
							for i, fire := range inf.Fires().ConvexHull2D() {
								fireBuffer = append(fireBuffer, FireEntry{
									matchid, rounds, tick, key, i, fire.X, fire.Y, "ENDING",
								})
							}
							FireParticles[key] = inf.Fires().Active().ConvexHull2D()
						} else {
							RoundPositions[rounds].FirePositions[tick][key] = FireState{
								Vertices: inf.Fires().Active().ConvexHull2D(), Status: "SPREADING",
							}
							for i, fire := range inf.Fires().Active().ConvexHull2D() {
								fireBuffer = append(fireBuffer, FireEntry{
									matchid, rounds, tick, key, i, fire.X, fire.Y, "SPREADING",
								})
							}
							FireParticles[key] = inf.Fires().Active().ConvexHull2D()
						}

					}
				}

			}
		}

	})
	p.RegisterEventHandler(func(k events.Kill) {
		var (
			id1 = int64(0)
			id2 = int64(0)
		)

		if k.Killer != nil {
			id1 = int64(k.Killer.SteamID64)
		}
		if k.Victim != nil {
			id2 = int64(k.Victim.SteamID64)
		}
		b_e := base_event{
			tick: p.GameState().IngameTick(), roundNo: rounds, matchid: matchid, steamid1: id1, steamid2: id2,
		}
		ee, re := get_event_entry(b_e, PlayerKilled)
		RoundPositions[rounds].RoundTimeline[p.GameState().IngameTick()] = re
		eventBuffer = append(eventBuffer, ee)
	})
	p.RegisterEventHandler(func(e events.RoundStart) {
		if !live {
			return
		}
		// Creating connecting tables
		RoundPositions[rounds] = RoundInfo{
			PlayerPositions: make(map[int]map[int64]PlayerState),
			PlayerNames:     make(map[int64]PlayerInfo),
			GrenadeEvents:   make(map[int]map[int]GrenadeState),
			FirePositions:   make(map[int]map[int]FireState),
			RoundTimeline:   make(map[int]RoundEvent),
		}
		gs := p.GameState()
		if !insertedRounds[rounds] {
			_, err := DB.Exec("INSERT IGNORE INTO ROUNDS (MATCHID, ROUND_NO) VALUES (?,?)", matchid, rounds)
			if err != nil {
				panic(err)
			}
			insertedRounds[rounds] = true
		}
		for _, players := range gs.Participants().Playing() {
			key := fmt.Sprintf("%d_%d_%d", matchid, rounds, players.SteamID64)
			if !insertedParticipants[key] {
				_, err = DB.Exec("INSERT IGNORE INTO ROUND_PARTICIPANTS (MATCHID, ROUND_NO, PLAYERID, SIDE) VALUES (?,?,?,?)", matchid, rounds, players.SteamID64, int(players.GetTeam()))
				if err != nil {
					panic(err)
				}
				insertedParticipants[key] = true
			}
		}
	})
	p.RegisterEventHandler(func(events.RoundFreezetimeEnd) {
		if live {
			tick := p.GameState().IngameTick()
			be := base_event{
				tick: tick, roundNo: rounds, matchid: matchid, steamid1: 0, steamid2: 0,
			}
			ee, re := get_event_entry(be, FreezeTimeEnd)
			RoundPositions[rounds].RoundTimeline[tick] = re
			eventBuffer = append(eventBuffer, ee)
		}
	})
	p.RegisterEventHandler(func(e events.RoundEndOfficial) {
		if !live {
			return
		}
		bufferSize := len(playerBuffer) + len(grenadeBuffer) + len(fireBuffer) + len(eventBuffer)
		fmt.Printf("Round %d ended. Flushing %d rows to DB...\n", rounds, bufferSize)
		rounds += 1
		// IMPORTANT: Clear the playerBuffer for the next round
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
	})

	p.RegisterEventHandler(func(be events.BombPlantBegin) {
		if !live {
			return
		}
		player := be.Player
		tick := p.GameState().IngameTick()
		sqlEntry, playerState := get_pos_entry(player, tick, rounds, matchid, beginPlanting)
		playerBuffer = append(playerBuffer, sqlEntry)
		if len(RoundPositions[rounds].PlayerPositions[tick]) == 0 {
			RoundPositions[rounds].PlayerPositions[tick] = make(map[int64]PlayerState)
		}
		RoundPositions[rounds].PlayerPositions[tick][int64(player.SteamID64)] = playerState
	})
	p.RegisterEventHandler(func(pe events.BombPlantAborted) {
		if !live {
			return
		}
		player := pe.Player
		tick := p.GameState().IngameTick()
		sqlEntry, playerState := get_pos_entry(player, tick, rounds, matchid, beginPlanting)
		playerBuffer = append(playerBuffer, sqlEntry)
		if len(RoundPositions[rounds].PlayerPositions[tick]) == 0 {
			RoundPositions[rounds].PlayerPositions[tick] = make(map[int64]PlayerState)
		}
		RoundPositions[rounds].PlayerPositions[tick][int64(player.SteamID64)] = playerState
	})
	p.RegisterEventHandler(func(be events.BombPlanted) {
		if !live {
			return
		}
		player := be.Player
		tick := p.GameState().IngameTick()
		sqlEntry, playerState := get_pos_entry(player, tick, rounds, matchid, beginPlanting)
		playerBuffer = append(playerBuffer, sqlEntry)
		// ADD BOMB TO GRENADE ENTRY. IS ON FLOOR
		matchInfo := base_grenade{
			tick: tick, roundNo: rounds, matchid: matchid,
			grenid: -1, gren_type: int(common.EqBomb), player: *be.Player,
			pos: player.Position(),
		}
		sqlEntry1, grenadeState := get_grenade_entry(matchInfo, "PLANTED")
		grenadeBuffer = append(grenadeBuffer, sqlEntry1)
		if len(RoundPositions[rounds].GrenadeEvents[tick]) == 0 {
			RoundPositions[rounds].GrenadeEvents[tick] = make(map[int]GrenadeState)
		}
		RoundPositions[rounds].GrenadeEvents[tick][-1] = grenadeState
		if len(RoundPositions[rounds].PlayerPositions[tick]) == 0 {
			RoundPositions[rounds].PlayerPositions[tick] = make(map[int64]PlayerState)
		}
		RoundPositions[rounds].PlayerPositions[tick][int64(player.SteamID64)] = playerState
		b_e := base_event{
			tick: tick, roundNo: rounds, matchid: matchid, steamid1: int64(be.Player.SteamID64), steamid2: 0,
		}
		ee, re := get_event_entry(b_e, BombPlanted)
		RoundPositions[rounds].RoundTimeline[tick] = re
		eventBuffer = append(eventBuffer, ee)
	})
	p.RegisterEventHandler(func(be events.BombDefuseStart) {
		if !live {
			return
		}
		player := be.Player
		tick := p.GameState().IngameTick()
		sqlEntry, playerState := get_pos_entry(player, tick, rounds, matchid, beginPlanting)
		playerBuffer = append(playerBuffer, sqlEntry)
		if len(RoundPositions[rounds].PlayerPositions[tick]) == 0 {
			RoundPositions[rounds].PlayerPositions[tick] = make(map[int64]PlayerState)
		}
		RoundPositions[rounds].PlayerPositions[tick][int64(player.SteamID64)] = playerState
	})
	p.RegisterEventHandler(func(pe events.BombDefuseAborted) {
		if !live {
			return
		}
		player := pe.Player
		tick := p.GameState().IngameTick()
		sqlEntry, playerState := get_pos_entry(player, tick, rounds, matchid, beginPlanting)
		playerBuffer = append(playerBuffer, sqlEntry)
		// log.Printf("%v", player.Weapons())
		if len(RoundPositions[rounds].PlayerPositions[tick]) == 0 {
			RoundPositions[rounds].PlayerPositions[tick] = make(map[int64]PlayerState)
		}
		RoundPositions[rounds].PlayerPositions[tick][int64(player.SteamID64)] = playerState
	})
	p.RegisterEventHandler(func(be events.BombDefused) {
		if !live {
			return
		}
		player := be.Player
		tick := p.GameState().IngameTick()
		sqlEntry, playerState := get_pos_entry(player, tick, rounds, matchid, beginPlanting)
		playerBuffer = append(playerBuffer, sqlEntry)
		// ADD BOMB TO GRENADE ENTRY.
		// BOMB IS ON FLOOR
		matchInfo := base_grenade{
			tick: tick, roundNo: rounds, matchid: matchid,
			grenid: -1, gren_type: int(common.EqBomb), player: *be.Player,
			pos: player.Position(),
		}

		sqlEntry1, grenadeState := get_grenade_entry(matchInfo, "DEFUSED")
		grenadeBuffer = append(grenadeBuffer, sqlEntry1)
		if len(RoundPositions[rounds].GrenadeEvents[tick]) == 0 {
			RoundPositions[rounds].GrenadeEvents[tick] = make(map[int]GrenadeState)
		}
		RoundPositions[rounds].GrenadeEvents[tick][-1] = grenadeState
		if len(RoundPositions[rounds].PlayerPositions[tick]) == 0 {
			RoundPositions[rounds].PlayerPositions[tick] = make(map[int64]PlayerState)
		}
		RoundPositions[rounds].PlayerPositions[tick][int64(player.SteamID64)] = playerState
		b_e := base_event{
			tick: tick, roundNo: rounds, matchid: matchid, steamid1: int64(be.Player.SteamID64), steamid2: 0,
		}
		ee, re := get_event_entry(b_e, BombDefused)
		RoundPositions[rounds].RoundTimeline[tick] = re
		eventBuffer = append(eventBuffer, ee)
	})
	p.RegisterEventHandler(func(be events.BombDropped) {
		player := be.Player
		tick := p.GameState().IngameTick()
		matchInfo := base_grenade{
			tick: tick, roundNo: rounds, matchid: matchid,
			grenid: -1, gren_type: int(common.EqBomb), player: *be.Player,
			pos: player.Position(),
		}
		sqlEntry, grenadeState := get_grenade_entry(matchInfo, "DROPPED")
		grenadeBuffer = append(grenadeBuffer, sqlEntry)
		if len(RoundPositions[rounds].GrenadeEvents[tick]) == 0 {
			RoundPositions[rounds].GrenadeEvents[tick] = make(map[int]GrenadeState)
		}
		RoundPositions[rounds].GrenadeEvents[tick][-1] = grenadeState
	})
	p.RegisterEventHandler(func(bp events.BombPickup) {
		player := bp.Player
		tick := p.GameState().IngameTick()
		matchInfo := base_grenade{
			tick: tick, roundNo: rounds, matchid: matchid,
			grenid: -1, gren_type: int(common.EqBomb), player: *bp.Player,
			pos: player.Position(),
		}
		sqlEntry, grenadeState := get_grenade_entry(matchInfo, "GRABBED")
		grenadeBuffer = append(grenadeBuffer, sqlEntry)
		if len(RoundPositions[rounds].GrenadeEvents[tick]) == 0 {
			RoundPositions[rounds].GrenadeEvents[tick] = make(map[int]GrenadeState)
		}
		RoundPositions[rounds].GrenadeEvents[tick][-1] = grenadeState
	})
	p.RegisterEventHandler(func(g events.GrenadeProjectileThrow) {
		if live != true {
			return
		}
		tick := p.GameState().IngameTick()
		id := g.Projectile.Entity.ID()
		b_e := base_event{
			tick: tick, roundNo: rounds, matchid: matchid, steamid1: int64(g.Projectile.Thrower.SteamID64), steamid2: 0,
		}
		var re RoundEvent
		var ee EventEntry
		switch g.Projectile.WeaponInstance.Type {
		case common.EqSmoke:
			ee, re = get_event_entry(b_e, SmokeThrow)
		case common.EqHE:
			ee, re = get_event_entry(b_e, HeThrow)
		case common.EqFlash:
			ee, re = get_event_entry(b_e, FlashThrow)
		case common.EqIncendiary:
		case common.EqMolotov:
			ee, re = get_event_entry(b_e, FireThrow)
		case common.EqDecoy:
			ee, re = get_event_entry(b_e, DecoyThrow)
		}
		RoundPositions[rounds].RoundTimeline[tick] = re
		eventBuffer = append(eventBuffer, ee)
		matchInfo := base_grenade{
			tick: tick, roundNo: rounds, matchid: matchid,
			grenid: id, gren_type: int(g.Projectile.WeaponInstance.Type), player: *g.Projectile.Thrower,
			pos: g.Projectile.Position(),
		}
		sqlEntry, grenadeState := get_grenade_entry(matchInfo, "FLYING")
		grenadeBuffer = append(grenadeBuffer, sqlEntry)
		if len(RoundPositions[rounds].GrenadeEvents[tick]) == 0 {
			RoundPositions[rounds].GrenadeEvents[tick] = make(map[int]GrenadeState)
		}
		RoundPositions[rounds].GrenadeEvents[tick][id] = grenadeState
		g.Projectile.Entity.OnPositionUpdate(func(pos r3.Vector) {
			upTick := p.GameState().IngameTick()
			if upTick%4 == 0 {
				matchInfo := base_grenade{
					tick: upTick, roundNo: rounds, matchid: matchid,
					grenid: id, gren_type: int(g.Projectile.WeaponInstance.Type), player: *g.Projectile.Thrower,
					pos: pos,
				}
				sqlEntry, grenadeState := get_grenade_entry(matchInfo, "FLYING")
				grenadeBuffer = append(grenadeBuffer, sqlEntry)
				if len(RoundPositions[rounds].PlayerPositions[upTick]) == 0 {
					RoundPositions[rounds].GrenadeEvents[upTick] = make(map[int]GrenadeState)
				}
				RoundPositions[rounds].GrenadeEvents[upTick][id] = grenadeState
			}

		})
	})
	p.RegisterEventHandler(func(g events.GrenadeProjectileDestroy) {
		if live != true {
			return
		}
		if int(g.Projectile.WeaponInstance.Type) == int(common.EqHE) || int(g.Projectile.WeaponInstance.Type) == int(common.EqFlash) || int(g.Projectile.WeaponInstance.Type) == int(common.EqSmoke) {
			return
		}
		tick := p.GameState().IngameTick()
		id := g.Projectile.Entity.ID()
		if len(RoundPositions[rounds].GrenadeEvents[tick]) == 0 {
			RoundPositions[rounds].GrenadeEvents[tick] = make(map[int]GrenadeState)
		}
		matchInfo := base_grenade{
			tick: tick, roundNo: rounds, matchid: matchid,
			grenid: id, gren_type: int(g.Projectile.WeaponInstance.Type), player: *g.Projectile.Thrower,
			pos: g.Projectile.Position(),
		}
		sqlEntry, grenadeState := get_grenade_entry(matchInfo, "LANDED")
		RoundPositions[rounds].GrenadeEvents[tick][id] = grenadeState
		grenadeBuffer = append(grenadeBuffer, sqlEntry)
	})
	p.RegisterEventHandler(func(g events.SmokeStart) {
		if live != true {
			return
		}
		tick := p.GameState().IngameTick()
		id := g.GrenadeEntityID
		position := r3.Vector{X: g.Position.X, Y: g.Position.Y, Z: g.Position.Z}
		matchInfo := base_grenade{
			tick: tick, roundNo: rounds, matchid: matchid,
			grenid: id, gren_type: int(g.GrenadeType), player: *g.Thrower,
			pos: position,
		}
		sqlEntry, grenadeState := get_grenade_entry(matchInfo, "BLOOMED")
		grenadeBuffer = append(grenadeBuffer, sqlEntry)
		if len(RoundPositions[rounds].PlayerPositions[tick]) == 0 {
			RoundPositions[rounds].GrenadeEvents[tick] = make(map[int]GrenadeState)

		}
		RoundPositions[rounds].GrenadeEvents[tick][id] = grenadeState
	})
	p.RegisterEventHandler(func(g events.SmokeExpired) {
		if live != true {
			return
		}
		tick := p.GameState().IngameTick()
		id := g.GrenadeEntityID
		position := r3.Vector{X: g.Position.X, Y: g.Position.Y, Z: g.Position.Z}
		matchInfo := base_grenade{
			tick: tick, roundNo: rounds, matchid: matchid,
			grenid: id, gren_type: int(g.GrenadeType), player: *g.Thrower,
			pos: position,
		}
		sqlEntry, grenadeState := get_grenade_entry(matchInfo, "EXPIRED")
		grenadeBuffer = append(grenadeBuffer, sqlEntry)
		if len(RoundPositions[rounds].PlayerPositions[tick]) == 0 {
			RoundPositions[rounds].GrenadeEvents[tick] = make(map[int]GrenadeState)

		}
		RoundPositions[rounds].GrenadeEvents[tick][id] = grenadeState
	})

	p.RegisterEventHandler(func(g events.HeExplode) {
		if live != true {
			return
		}
		tick := p.GameState().IngameTick()
		id := g.GrenadeEntityID
		position := r3.Vector{X: g.Position.X, Y: g.Position.Y, Z: g.Position.Z}
		matchInfo := base_grenade{
			tick: tick, roundNo: rounds, matchid: matchid,
			grenid: id, gren_type: int(g.GrenadeType), player: *g.Thrower,
			pos: position,
		}
		sqlEntry, grenadeState := get_grenade_entry(matchInfo, "EXPIRED")
		grenadeBuffer = append(grenadeBuffer, sqlEntry)
		if len(RoundPositions[rounds].PlayerPositions[tick]) == 0 {
			RoundPositions[rounds].GrenadeEvents[tick] = make(map[int]GrenadeState)

		}
		RoundPositions[rounds].GrenadeEvents[tick][id] = grenadeState
	})

	p.RegisterEventHandler(func(g events.FlashExplode) {
		if live != true {
			return
		}
		tick := p.GameState().IngameTick()
		id := g.GrenadeEntityID
		position := r3.Vector{X: g.Position.X, Y: g.Position.Y, Z: g.Position.Z}
		matchInfo := base_grenade{
			tick: tick, roundNo: rounds, matchid: matchid,
			grenid: id, gren_type: int(g.GrenadeType), player: *g.Thrower,
			pos: position,
		}
		sqlEntry, grenadeState := get_grenade_entry(matchInfo, "EXPIRED")
		grenadeBuffer = append(grenadeBuffer, sqlEntry)
		if len(RoundPositions[rounds].PlayerPositions[tick]) == 0 {
			RoundPositions[rounds].GrenadeEvents[tick] = make(map[int]GrenadeState)

		}
		RoundPositions[rounds].GrenadeEvents[tick][id] = grenadeState
	})
	p.RegisterEventHandler(func(g events.InfernoStart) {
		if live == false {
			return
		}
		gs := p.GameState()
		tick := gs.IngameTick()
		id := g.Inferno.Entity.ID()

		// FireParticles[id] = g.Inferno.Fires().ConvexHull3D().Vertices
		if len(RoundPositions[rounds].FirePositions[tick]) == 0 {
			RoundPositions[rounds].FirePositions[tick] = make(map[int]FireState)
		}
		RoundPositions[rounds].FirePositions[tick][id] = FireState{
			Vertices: g.Inferno.Fires().Active().ConvexHull2D(), Status: "STARTING",
		}
		for i, fire := range g.Inferno.Fires().Active().ConvexHull2D() {
			fireBuffer = append(fireBuffer, FireEntry{
				matchid, rounds, tick, id, i, fire.X, fire.Y, "STARTING",
			})
		}

	})
	p.RegisterEventHandler(func(g events.InfernoExpired) {
		if live == false {
			return
		}
		gs := p.GameState()
		tick := gs.IngameTick()
		id := g.Inferno.Entity.ID()
		if len(RoundPositions[rounds].FirePositions[tick]) == 0 {
			RoundPositions[rounds].FirePositions[tick] = make(map[int]FireState)
		}
		RoundPositions[rounds].FirePositions[tick][id] = FireState{
			Vertices: g.Inferno.Fires().ConvexHull2D(), Status: "ENDING",
		}
		for i, fire := range g.Inferno.Fires().ConvexHull2D() {
			fireBuffer = append(fireBuffer, FireEntry{
				matchid, rounds, tick, id, i, fire.X, fire.Y, "ENDING",
			})
		}
	})
	err = p.ParseToEnd()
	if err != nil {
		panic(err)
	}
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

func get_pos_entry(player *common.Player, tick int, round int, matchid int, action PlayerAction) (posEntry, PlayerState) {
	pos := player.Position()
	inv := player.Weapons()
	hasBomb := false
	var (
		primary   = common.EqUnknown
		secondary = common.EqUnknown
		smoke     = common.EqUnknown
		hegren    = common.EqUnknown
		fire      = common.EqUnknown
		flash1    = common.EqUnknown
		flash2    = common.EqUnknown
		decoy     = common.EqUnknown
	)
	for _, wep := range inv {
		if wep.Type == common.EqBomb {
			hasBomb = true
		}
		switch wep.Class() {
		case common.EqClassRifle, common.EqClassHeavy, common.EqClassSMG:
			// Note: Cleaned up your duplicate EqClassRifle check here too!
			primary = wep.Type
		case common.EqClassPistols:
			secondary = wep.Type
		case common.EqClassGrenade:
			switch wep.Type {
			case common.EqSmoke:
				smoke = wep.Type
			case common.EqHE:
				hegren = wep.Type
			case common.EqFlash:
				if flash1 == common.EqUnknown {
					flash1 = wep.Type
				} else {
					flash2 = wep.Type
				}
			case common.EqIncendiary:
			case common.EqMolotov:
				fire = wep.Type
			case common.EqDecoy:
				decoy = wep.Type
			}

		}
	}

	var activeWep int
	if player.ActiveWeapon() == nil {
		activeWep = 0
	} else {
		activeWep = int(player.ActiveWeapon().Type)
	}
	pE := posEntry{
		matchid, round, tick, int(player.GetTeam()), player.SteamID64, player.Health(), player.Kills(),
		player.Assists(), player.Deaths(), player.Armor(), player.Money(), int(primary), int(secondary),
		int(smoke), int(hegren), int(flash1), int(flash2), int(fire), int(decoy), hasBomb, pos.X, pos.Y, pos.Z,
		player.FlashDurationTimeRemaining().Seconds(), activeWep, action,
	}
	pS := PlayerState{
		Position: player.Position(), Active_Weapon: activeWep, HP: player.Health(),
		Kills: player.Kills(), Assists: player.Assists(), Deaths: player.Deaths(),
		Primary: int(primary), Secondary: int(secondary), SmokeSlot: int(smoke), HESlot: int(hegren),
		Flashslot1: int(flash1), FlashSlot2: int(flash2), DecoySlot: int(decoy), FireSlot: int(fire),
		Armor: player.Armor(), Money: player.Money(), Action: action, HasBomb: hasBomb,
		BlindDuration: player.FlashDurationTimeRemaining().Seconds(),
	}
	return pE, pS
}
func get_grenade_entry(base base_grenade, grenState string) (GrenadeEntry, GrenadeState) {
	player := base.player
	grenid := base.grenid
	gren_type := base.gren_type
	position := base.pos
	ge := GrenadeEntry{
		base.matchid, base.roundNo, base.tick, grenid, player.SteamID64, position.X, position.Y, position.Z, gren_type, grenState,
	}
	gs := GrenadeState{
		Position: position, Grenade: gren_type, ThrownByName: player.Name, ThrownByid: int64(player.SteamID64), Status: grenState,
	}
	return ge, gs
}
func get_event_entry(base base_event, event_type TrackedEvents) (EventEntry, RoundEvent) {
	ee := EventEntry{
		tick: base.tick, roundNo: base.roundNo, matchid: base.matchid,
		event: int(event_type), steamid1: int64(base.steamid1), steamid2: int64(base.steamid2),
	}
	re := RoundEvent{
		Event: event_type, Player1: int64(base.steamid1), Player2: int64(base.steamid2),
	}
	return ee, re
}
func flushToDB(db *sql.DB, entries []posEntry) {
	tx, err := db.Begin()
	if err != nil {
		panic(err)
	}

	Players := make(map[uint64]int)

	for _, e := range entries {
		_, err = tx.Exec("INSERT IGNORE INTO ROUNDS (MATCHID, ROUND_NO) VALUES (?,?)", e.matchID, e.roundNo)
		if err != nil {
			panic(err)
		}
		_, ok := Players[e.steamID]
		if !ok {
			Players[e.steamID] = e.side
			_, err = DB.Exec("INSERT IGNORE INTO ROUND_PARTICIPANTS (MATCHID, ROUND_NO, PLAYERID, SIDE) VALUES (?,?,?,?)", e.matchID, e.roundNo, e.steamID, e.side)
			if err != nil {
				panic(err)
			}
		} else {
			continue
		}
	}

	stmt, err := tx.Prepare("INSERT INTO PLAYER_EVENTS" +
		"(MATCHID, ROUND_NO, PLAYERID, HP, ACTIVE_WEAPON, HAS_BOMB, KILLS, ASSISTS, DEATHS, ARMOR, DINERO, P_ACTION," +
		"PRIMARY_SLOT,SECONDARY_SLOT,SMOKE_SLOT,FIRE_SLOT,HE_SLOT,DECOY_SLOT,FLASH_SLOT1,FLASH_SLOT2,FLASHED_DURATION,XPOS,YPOS,ZPOS,TICK) VALUES" +
		"(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)" +
		"ON DUPLICATE KEY UPDATE ACTIVE_WEAPON=(ACTIVE_WEAPON), XPOS=VALUES(XPOS), YPOS=VALUES(YPOS), ZPOS=VALUES(ZPOS)")
	if err != nil {
		panic(err)
	}
	defer stmt.Close()

	for _, e := range entries {
		// fmt.Printf("MID:%v, ROUND_NO:%v, STEAM:%v, WEAPON:%v, X:%v, Y:%v, Z:%v, TICK:%v\n", e.matchID, e.roundNo, e.steamID, e.weapon, e.x, e.y, e.z, e.tick)
		if _, err := stmt.Exec(e.matchID, e.roundNo, e.steamID, e.hp, e.weapon, e.hasBomb, e.kills, e.assists, e.deaths, e.armor, e.money, e.action,
			e.primary, e.seconday, e.smoke, e.fire, e.he, e.decoy, e.flash1, e.flash2, e.flashDur, e.x, e.y, e.z, e.tick); err != nil {
			tx.Rollback()
			panic(err)
		}
	}

	if err := tx.Commit(); err != nil {
		panic(err)
	}
}

func fireFlush(db *sql.DB, entries []FireEntry) {
	tx, err := db.Begin()
	if err != nil {
		panic(err)
	}
	stmt, err := tx.Prepare("INSERT INTO FIRE_VERTICES (MATCHID,ROUND_NO,TICK,ENTITYID,FIREID,XPOS,YPOS,ENTSTATE)" +
		"VALUES (?,?,?,?,?,?,?,?) ON DUPLICATE KEY UPDATE XPOS=VALUES(XPOS), YPOS=VALUES(YPOS)")
	if err != nil {
		panic(err)
	}
	defer stmt.Close()
	for _, e := range entries {
		if _, err := stmt.Exec(e.matchID, e.roundNo, e.tick, e.entid, e.fireid, e.x, e.y, e.state); err != nil {
			tx.Rollback()
			panic(err)
		}
	}

	if err := tx.Commit(); err != nil {
		panic(err)
	}
}
func eventFlush(db *sql.DB, entries []EventEntry) {
	tx, err := db.Begin()
	if err != nil {
		panic(err)
	}
	for _, e := range entries {
		_, err = tx.Exec("INSERT IGNORE INTO ROUNDS (MATCHID, ROUND_NO) VALUES (?,?)", e.matchid, e.roundNo)
		if err != nil {
			panic(err)
		}
		break
	}
	stmt, err := tx.Prepare("INSERT IGNORE INTO ROUND_EVENTS (MATCHID,ROUND_NO,TICK,EVENT_TYPE,PLAYER1ID,PLAYER2ID)" +
		"VALUES (?,?,?,?,?,?)")
	// ON DUPLICATE KEY UPDATE MATCHID=VALUES(MATCHID), ROUND_NO=VALUES(ROUND_NO), TICK=VALUES(TICK)
	if err != nil {
		panic(err)
	}
	defer stmt.Close()
	for _, e := range entries {
		if _, err := stmt.Exec(e.matchid, e.roundNo, e.tick, e.event, e.steamid1, e.steamid2); err != nil {
			tx.Rollback()
			panic(err)
		}
	}

	if err := tx.Commit(); err != nil {
		panic(err)
	}
}
func grenadeFlush(db *sql.DB, entries []GrenadeEntry) {
	tx, err := db.Begin()
	if err != nil {
		panic(err)
	}

	stmt, err := tx.Prepare("INSERT INTO GRENADE_EVENTS (MATCHID,ROUND_NO,PLAYERID,ENTITYID,TICK,XPOS,YPOS,ZPOS,GRENADE,ENTSTATE) VALUES (?,?,?,?,?,?,?,?,?,?)" +
		"ON DUPLICATE KEY UPDATE GRENADE=(GRENADE), XPOS=VALUES(XPOS), YPOS=VALUES(YPOS), ZPOS=VALUES(ZPOS)")
	if err != nil {
		panic(err)
	}
	defer stmt.Close()

	for _, e := range entries {
		if _, err := stmt.Exec(e.matchID, e.roundNo, e.steamID, e.entid, e.tick, e.x, e.y, e.z, e.grenade, e.state); err != nil {
			tx.Rollback()
			panic(err)
		}
	}

	if err := tx.Commit(); err != nil {
		panic(err)
	}
}
