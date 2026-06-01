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

type BaseDemo struct {
	FileName  string  `json:"filename,string"`
	ModDate   string  `json:"date,string"`
	FileSize  string  `json:"filesize,string"`
	Map       string  `json:"map,string"`
	TeamStats [2]Team `json:"team_stats"`
	Parsed    bool    `json:"parsed"`
	BaseStats bool    `json:"stats"`
	ID        int
}

type PlayerStats struct {
	Kills   int `json:"kills"`
	Deaths  int `json:"deaths"`
	Assists int `json:"assists"`
}
type MatchEvents struct {
	RoundPositions RoundEvents                 `json:"round_events"`
	Rounds         int                         `json:"rounds"`
	MapMeta        ex.Map                      `json:"map"`
	Teams          map[string]map[int64]string `json:"teams"`
}
type RoundEvents struct {
	// map[TICK] -> Map(ID) i.e playerid or ent id -> State/Info
	PlayerPositions map[int]map[int64]PlayerState `json:"player_positions"`
	PlayerNames     map[int64]PlayerInfo          `json:"player_info"`
	GrenadeEvents   map[int]map[int]GrenadeState  `json:"grenade_events"`
	FirePositions   map[int]map[int]FireState     `json:"fire_events"`
}
type PlayerInfo struct {
	Name string `json:"name"`
	Side int    `json:"side"`
}
type Player struct {
	Name  string      `json:"name"`
	ID    int64       `json:"ID"`
	Stats PlayerStats `json:"stats"`
}
type Team struct {
	ID             int               `json:"ID"`
	ClanName       string            `json:"Clanname"`
	EndScore       int               `json:"Endscore"`
	TScore         int               `json:"TScore"`
	CTScore        int               `json:"CTScore"`
	PlayingPlayers map[string]Player `json:"Playing"`
	inited         bool
}
type PlayerState struct {
	Position r3.Vector    `json:"vector"`
	Weapon   string       `json:"weapon"`
	HP       int          `json:"hp"`
	Kills    int          `json:"kills"`
	Assists  int          `json:"assists"`
	Deaths   int          `json:"deaths"`
	Armor    int          `json:"armor"`
	Money    int          `json:"dinero"`
	Action   PlayerAction `json:"action"`
	HasBomb  bool         `json:"hasBomb"`
}
type PlayerAction int

const (
	isMoving PlayerAction = iota
	beginPlanting
	donePlanting
	abortedPlant
	beginDefusing
	doneDefusing
	abortedDefuse
)

type GrenadeState struct {
	Position     r3.Vector `json:"vector"`
	Grenade      string    `json:"grenade"`
	ThrownByName string    `json:"thrownBy"`
	ThrownByid   int64     `json:"thrownById"`
	Status       string    `json:"status"`
}
type FireState struct {
	Vertices []r2.Point `json:"vertices"`
	Status   string     `json:"status"`
}
type posEntry struct {
	matchID, roundNo, tick, side             int
	steamID                                  uint64
	hp, kills, assists, deaths, armor, money int
	hasBomb                                  bool
	x, y, z                                  float64
	weapon                                   string
	action                                   PlayerAction
}
type GrenadeEntry struct {
	matchID, roundNo, tick int
	entid                  int
	steamID                uint64
	x, y, z                float64
	grenade, state         string
}
type FireEntry struct {
	matchID, roundNo, tick, entid, fireid int
	x, y                                  float64
	state                                 string
}

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
	var playerBuffer []posEntry
	var grenadeBuffer []GrenadeEntry
	var fireBuffer []FireEntry
	posBatch := make(chan []posEntry, 300)
	grenadeBatch := make(chan []GrenadeEntry, 300)
	fireBatch := make(chan []FireEntry, 300)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for batch := range posBatch {
			flushToDB(DB, batch) // Your helper function with the transaction
		}
		for batch := range grenadeBatch {
			grenadeFlush(DB, batch)
		}
		for batch := range fireBatch {
			fireFlush(DB, batch)
		}
	}()
	var playback MatchEvents
	// playback.Teams = make(map[string]map[int64]string)
	var RoundPositions map[int]RoundEvents
	var FireParticles map[int][]r2.Point
	live := false

	rounds := 0

	// Helper to ensure round entry exists before accessing

	log.Printf("STARTING PARSE")

	p.RegisterEventHandler(func(e events.MatchStartedChanged) {
		if rounds > 1 {
			return
		}
		live = true
		rounds = 1
		RoundPositions = make(map[int]RoundEvents)
		FireParticles = make(map[int][]r2.Point)
		playerBuffer = playerBuffer[:0]
		grenadeBuffer = grenadeBuffer[:0]
		fireBuffer = fireBuffer[:0]
		// playback = MatchEvents{Teams: make(map[string]map[int64]string)}
		playback.Teams = make(map[string]map[int64]string)
		gs := p.GameState()
		_, err := DB.Exec("INSERT IGNORE INTO ROUNDS (MATCHID, ROUND_NO) VALUES (?,?)", matchid, rounds)
		for _, player := range gs.Participants().Playing() {
			name := player.TeamState.ClanName()
			if _, ok := playback.Teams[name]; !ok {
				playback.Teams[name] = make(map[int64]string)
			}
			playback.Teams[name][int64(player.SteamID64)] = player.Name
			_, err = DB.Exec("INSERT IGNORE INTO ROUND_PARTICIPANTS (MATCHID, ROUND_NO, PLAYERID, SIDE) VALUES (?,?,?,?)", matchid, rounds, player.SteamID64, int(player.GetTeam()))
			if err != nil {
				panic(err)
			}
		}
		RoundPositions[rounds] = RoundEvents{
			PlayerPositions: make(map[int]map[int64]PlayerState),
			PlayerNames:     make(map[int64]PlayerInfo),
			GrenadeEvents:   make(map[int]map[int]GrenadeState),
			FirePositions:   make(map[int]map[int]FireState),
		}
	})
	// p.RegisterEventHandler(func(events.M))
	p.RegisterEventHandler(func(events.FrameDone) {
		if live {
			gs := p.GameState()
			tick := gs.IngameTick()
			flames := gs.Infernos()
			if tick%8 != 0 {
				return
			}
			for _, player := range gs.Participants().Playing() {
				pos := player.Position()
				if player.IsAlive() {
					inv := player.Weapons()
					hasBomb := false
					for _, wep := range inv {
						if wep.Type == common.EqBomb {
							hasBomb = true
						}
					}
					playerBuffer = append(playerBuffer, posEntry{
						matchid, rounds, tick, int(player.GetTeam()), player.SteamID64,
						player.Health(), player.Kills(), player.Assists(), player.Deaths(), player.Armor(), player.Money(), hasBomb,
						pos.X, pos.Y, pos.Z, player.ActiveWeapon().String(), isMoving,
					})
					// Check to see if player is added
					position := r3.Vector{X: pos.X, Y: pos.Y, Z: pos.Z}
					if _, ok := RoundPositions[rounds].PlayerNames[int64(player.SteamID64)]; !ok {
						RoundPositions[rounds].PlayerNames[int64(player.SteamID64)] = PlayerInfo{Name: player.Name, Side: int(player.GetTeam())}
					}
					// log.Printf("%v", player.Weapons())
					if len(RoundPositions[rounds].PlayerPositions[tick]) == 0 {
						RoundPositions[rounds].PlayerPositions[tick] = make(map[int64]PlayerState)

					}
					RoundPositions[rounds].PlayerPositions[tick][int64(player.SteamID64)] = PlayerState{
						Position: position, Weapon: player.ActiveWeapon().String(), HP: player.Health(),
						Kills: player.Kills(), Assists: player.Assists(), Deaths: player.Deaths(),
						Armor: player.Armor(), Money: player.Money(), Action: isMoving, HasBomb: hasBomb,
					}
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
	p.RegisterEventHandler(func(e events.RoundStart) {
		if !live {
			return
		}
		// Creating connecting tables
		RoundPositions[rounds] = RoundEvents{
			PlayerPositions: make(map[int]map[int64]PlayerState),
			PlayerNames:     make(map[int64]PlayerInfo),
			GrenadeEvents:   make(map[int]map[int]GrenadeState),
			FirePositions:   make(map[int]map[int]FireState),
		}
		gs := p.GameState()
		_, err := DB.Exec("INSERT IGNORE INTO ROUNDS (MATCHID, ROUND_NO) VALUES (?,?)", matchid, rounds)

		if err != nil {
			panic(err)
		}
		for _, players := range gs.Participants().Playing() {
			_, err = DB.Exec("INSERT IGNORE INTO ROUND_PARTICIPANTS (MATCHID, ROUND_NO, PLAYERID, SIDE) VALUES (?,?,?,?)", matchid, rounds, players.SteamID64, int(players.GetTeam()))
			if err != nil {
				panic(err)
			}
		}
	})
	p.RegisterEventHandler(func(e events.RoundEndOfficial) {
		if !live {
			return
		}
		bufferSize := len(playerBuffer) + len(grenadeBuffer) + len(fireBuffer)
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
	})

	p.RegisterEventHandler(func(be events.BombPlantBegin) {
		if !live {
			return
		}
		// log.Printf("round %v planted begin", rounds)
		player := be.Player
		pos := player.Position()
		tick := p.GameState().IngameTick()
		// log.Printf("About to create entry at %v", tick)
		playerBuffer = append(playerBuffer, posEntry{
			matchid, rounds, tick, int(player.GetTeam()), player.SteamID64,
			player.Health(), player.Kills(), player.Assists(), player.Deaths(), player.Armor(), player.Money(), true,
			pos.X, pos.Y, pos.Z, player.ActiveWeapon().String(), beginPlanting,
		})
		// log.Print("Entry Created")
		// log.Printf("%v", player.Weapons())
		if len(RoundPositions[rounds].PlayerPositions[tick]) == 0 {
			// log.Print("Empty Thing")
			RoundPositions[rounds].PlayerPositions[tick] = make(map[int64]PlayerState)
		}
		// log.Printf("%v", RoundPositions[rounds].PlayerPositions[tick])
		test := PlayerState{
			Position: pos, Weapon: player.ActiveWeapon().String(), HP: player.Health(), HasBomb: true,
			Kills: player.Kills(), Assists: player.Assists(), Deaths: player.Deaths(),
			Armor: player.Armor(), Money: player.Money(), Action: beginPlanting,
		}
		// log.Printf("%v", RoundPositions[rounds])
		RoundPositions[rounds].PlayerPositions[tick][int64(player.SteamID64)] = test

	})
	p.RegisterEventHandler(func(pe events.BombPlantAborted) {
		if !live {
			return
		}
		player := pe.Player
		pos := player.Position()
		tick := p.GameState().IngameTick()
		playerBuffer = append(playerBuffer, posEntry{
			matchid, rounds, tick, int(player.GetTeam()), player.SteamID64,
			player.Health(), player.Kills(), player.Assists(), player.Deaths(), player.Armor(), player.Money(), true,
			pos.X, pos.Y, pos.Z, "", abortedPlant,
		})
		// log.Printf("%v", player.Weapons())
		if len(RoundPositions[rounds].PlayerPositions[tick]) == 0 {
			RoundPositions[rounds].PlayerPositions[tick] = make(map[int64]PlayerState)
		}
		RoundPositions[rounds].PlayerPositions[tick][int64(player.SteamID64)] = PlayerState{
			Position: pos, Weapon: "", HP: player.Health(), HasBomb: true,
			Kills: player.Kills(), Assists: player.Assists(), Deaths: player.Deaths(),
			Armor: player.Armor(), Money: player.Money(), Action: abortedPlant,
		}
	})
	p.RegisterEventHandler(func(be events.BombPlanted) {
		if !live {
			return
		}
		player := be.Player
		pos := player.Position()
		tick := p.GameState().IngameTick()
		playerBuffer = append(playerBuffer, posEntry{
			matchid, rounds, tick, int(player.GetTeam()), player.SteamID64,
			player.Health(), player.Kills(), player.Assists(), player.Deaths(), player.Armor(), player.Money(), false,
			pos.X, pos.Y, pos.Z, "", donePlanting,
		})
		// ADD BOMB TO GRENADE ENTRY. IS ON FLOOR
		grenadeBuffer = append(grenadeBuffer, GrenadeEntry{
			matchid, rounds, tick, -1, player.SteamID64, pos.X, pos.Y, pos.Z, "BOMB", "PLANTED",
		})
		if len(RoundPositions[rounds].GrenadeEvents[tick]) == 0 {
			RoundPositions[rounds].GrenadeEvents[tick] = make(map[int]GrenadeState)
		}
		RoundPositions[rounds].GrenadeEvents[tick][-1] = GrenadeState{
			Position: pos, Grenade: "BOMB", ThrownByName: player.Name, ThrownByid: int64(player.SteamID64),
		}
		// log.Print("Entry created")
		if len(RoundPositions[rounds].PlayerPositions[tick]) == 0 {
			RoundPositions[rounds].PlayerPositions[tick] = make(map[int64]PlayerState)
		}
		RoundPositions[rounds].PlayerPositions[tick][int64(player.SteamID64)] = PlayerState{
			Position: pos, Weapon: "", HP: player.Health(), HasBomb: false,
			Kills: player.Kills(), Assists: player.Assists(), Deaths: player.Deaths(),
			Armor: player.Armor(), Money: player.Money(), Action: donePlanting,
		}
	})
	p.RegisterEventHandler(func(be events.BombDefuseStart) {
		if !live {
			return
		}
		player := be.Player
		pos := player.Position()
		tick := p.GameState().IngameTick()
		playerBuffer = append(playerBuffer, posEntry{
			matchid, rounds, tick, int(player.GetTeam()), player.SteamID64,
			player.Health(), player.Kills(), player.Assists(), player.Deaths(), player.Armor(), player.Money(), false,
			pos.X, pos.Y, pos.Z, player.ActiveWeapon().String(), beginDefusing,
		})
		if len(RoundPositions[rounds].PlayerPositions[tick]) == 0 {
			RoundPositions[rounds].PlayerPositions[tick] = make(map[int64]PlayerState)
		}
		test := PlayerState{
			Position: pos, Weapon: player.ActiveWeapon().String(), HP: player.Health(), HasBomb: false,
			Kills: player.Kills(), Assists: player.Assists(), Deaths: player.Deaths(),
			Armor: player.Armor(), Money: player.Money(), Action: beginDefusing,
		}
		RoundPositions[rounds].PlayerPositions[tick][int64(player.SteamID64)] = test
	})
	p.RegisterEventHandler(func(pe events.BombDefuseAborted) {
		if !live {
			return
		}
		player := pe.Player
		pos := player.Position()
		tick := p.GameState().IngameTick()
		playerBuffer = append(playerBuffer, posEntry{
			matchid, rounds, tick, int(player.GetTeam()), player.SteamID64,
			player.Health(), player.Kills(), player.Assists(), player.Deaths(), player.Armor(), player.Money(), false,
			pos.X, pos.Y, pos.Z, "", abortedDefuse,
		})
		// log.Printf("%v", player.Weapons())
		if len(RoundPositions[rounds].PlayerPositions[tick]) == 0 {
			RoundPositions[rounds].PlayerPositions[tick] = make(map[int64]PlayerState)
		}
		RoundPositions[rounds].PlayerPositions[tick][int64(player.SteamID64)] = PlayerState{
			Position: pos, Weapon: "", HP: player.Health(), HasBomb: false,
			Kills: player.Kills(), Assists: player.Assists(), Deaths: player.Deaths(),
			Armor: player.Armor(), Money: player.Money(), Action: abortedDefuse,
		}
	})
	p.RegisterEventHandler(func(be events.BombDefused) {
		if !live {
			return
		}
		player := be.Player
		pos := player.Position()
		tick := p.GameState().IngameTick()
		playerBuffer = append(playerBuffer, posEntry{
			matchid, rounds, tick, int(player.GetTeam()), player.SteamID64,
			player.Health(), player.Kills(), player.Assists(), player.Deaths(), player.Armor(), player.Money(), false,
			pos.X, pos.Y, pos.Z, "", doneDefusing,
		})
		// ADD BOMB TO GRENADE ENTRY.
		// BOMB IS ON FLOOR
		grenadeBuffer = append(grenadeBuffer, GrenadeEntry{
			matchid, rounds, tick, -1, player.SteamID64, pos.X, pos.Y, pos.Z, "BOMB", "DEFUSED",
		})
		if len(RoundPositions[rounds].GrenadeEvents[tick]) == 0 {
			RoundPositions[rounds].GrenadeEvents[tick] = make(map[int]GrenadeState)
		}
		RoundPositions[rounds].GrenadeEvents[tick][-1] = GrenadeState{
			Position: pos, Grenade: "BOMB", ThrownByName: player.Name, ThrownByid: int64(player.SteamID64),
		}
		// log.Print("Entry created")
		if len(RoundPositions[rounds].PlayerPositions[tick]) == 0 {
			RoundPositions[rounds].PlayerPositions[tick] = make(map[int64]PlayerState)
		}
		RoundPositions[rounds].PlayerPositions[tick][int64(player.SteamID64)] = PlayerState{
			Position: pos, Weapon: "", HP: player.Health(), HasBomb: false,
			Kills: player.Kills(), Assists: player.Assists(), Deaths: player.Deaths(),
			Armor: player.Armor(), Money: player.Money(), Action: doneDefusing,
		}
	})
	p.RegisterEventHandler(func(be events.BombDropped) {
		player := be.Player
		pos := player.Position()
		tick := p.GameState().IngameTick()
		grenadeBuffer = append(grenadeBuffer, GrenadeEntry{
			matchid, rounds, tick, -1, player.SteamID64, pos.X, pos.Y, pos.Z, "BOMB", "DROPPED",
		})
		if len(RoundPositions[rounds].GrenadeEvents[tick]) == 0 {
			RoundPositions[rounds].GrenadeEvents[tick] = make(map[int]GrenadeState)
		}
		RoundPositions[rounds].GrenadeEvents[tick][-1] = GrenadeState{
			Position: pos, Grenade: "BOMB", ThrownByName: player.Name, ThrownByid: int64(player.SteamID64),
		}
	})
	p.RegisterEventHandler(func(bp events.BombPickup) {
		player := bp.Player
		pos := player.Position()
		tick := p.GameState().IngameTick()
		grenadeBuffer = append(grenadeBuffer, GrenadeEntry{
			matchid, rounds, tick, -1, player.SteamID64, pos.X, pos.Y, pos.Z, "BOMB", "GRABBED",
		})
		if len(RoundPositions[rounds].GrenadeEvents[tick]) == 0 {
			RoundPositions[rounds].GrenadeEvents[tick] = make(map[int]GrenadeState)
		}
		RoundPositions[rounds].GrenadeEvents[tick][-1] = GrenadeState{
			Position: pos, Grenade: "BOMB", ThrownByName: player.Name, ThrownByid: int64(player.SteamID64),
		}
	})
	p.RegisterEventHandler(func(g events.GrenadeProjectileThrow) {
		if live != true {
			return
		}
		gs := p.GameState()
		tick := gs.IngameTick()
		player := g.Projectile.Thrower
		grenade := g.Projectile.Entity
		id := g.Projectile.Entity.ID()
		grenadeBuffer = append(grenadeBuffer, GrenadeEntry{
			matchid, rounds, tick, id, player.SteamID64, grenade.Position().X, grenade.Position().Y, grenade.Position().Z, g.Projectile.WeaponInstance.String(), "FLYING",
		})
		position := r3.Vector{X: grenade.Position().X, Y: grenade.Position().Y, Z: grenade.Position().Z}
		if len(RoundPositions[rounds].GrenadeEvents[tick]) == 0 {
			RoundPositions[rounds].GrenadeEvents[tick] = make(map[int]GrenadeState)
		}
		RoundPositions[rounds].GrenadeEvents[tick][id] = GrenadeState{
			Position: position, Grenade: g.Projectile.WeaponInstance.String(), ThrownByName: player.Name, ThrownByid: int64(player.SteamID64),
		}
		g.Projectile.Entity.OnPositionUpdate(func(pos r3.Vector) {
			upTick := p.GameState().IngameTick()
			if upTick%4 == 0 {
				grenadeBuffer = append(grenadeBuffer, GrenadeEntry{
					matchid, rounds, upTick, id, player.SteamID64, pos.X, pos.Y, pos.Z, g.Projectile.WeaponInstance.String(), "FLYING",
				})
				position := r3.Vector{X: pos.X, Y: pos.Y, Z: pos.Z}
				if len(RoundPositions[rounds].PlayerPositions[upTick]) == 0 {
					RoundPositions[rounds].GrenadeEvents[upTick] = make(map[int]GrenadeState)
					RoundPositions[rounds].GrenadeEvents[upTick][id] = GrenadeState{
						Position: position, Grenade: g.Projectile.WeaponInstance.String(), ThrownByName: player.Name, ThrownByid: int64(player.SteamID64),
					}
				} else {
					RoundPositions[rounds].GrenadeEvents[upTick][id] = GrenadeState{
						Position: position, Grenade: g.Projectile.WeaponInstance.String(), ThrownByName: player.Name, ThrownByid: int64(player.SteamID64),
					}
				}
			}

		})
	})
	p.RegisterEventHandler(func(g events.GrenadeProjectileDestroy) {
		if live != true {
			return
		}
		if g.Projectile.WeaponInstance.String() == "HE Grenade" || g.Projectile.WeaponInstance.String() == "Flashbang" || g.Projectile.WeaponInstance.String() == "Smoke Grenade" {
			return
		}
		gs := p.GameState()
		tick := gs.IngameTick()
		player := g.Projectile.Thrower
		id := g.Projectile.Entity.ID()
		if len(RoundPositions[rounds].GrenadeEvents[tick]) == 0 {
			RoundPositions[rounds].GrenadeEvents[tick] = make(map[int]GrenadeState)
		}
		position := g.Projectile.Entity.Position()
		RoundPositions[rounds].GrenadeEvents[tick][id] = GrenadeState{
			Position: position, Grenade: g.Projectile.WeaponInstance.String(), ThrownByName: player.Name, ThrownByid: int64(player.SteamID64),
		}
		grenadeBuffer = append(grenadeBuffer, GrenadeEntry{
			matchid, rounds, tick, id, player.SteamID64, position.X, position.Y, position.Z, g.Projectile.WeaponInstance.String(), "LANDED",
		})
	})
	p.RegisterEventHandler(func(g events.SmokeStart) {
		if live != true {
			return
		}
		gs := p.GameState()
		tick := gs.IngameTick()
		id := g.GrenadeEntityID
		player := g.Thrower
		grenadeBuffer = append(grenadeBuffer, GrenadeEntry{
			matchid, rounds, tick, id, player.SteamID64, g.Position.X, g.Position.Y, g.Position.Z, g.Grenade.String(), "BLOOMED",
		})
		if len(RoundPositions[rounds].PlayerPositions[tick]) == 0 {
			RoundPositions[rounds].GrenadeEvents[tick] = make(map[int]GrenadeState)

		}
		position := r3.Vector{X: g.Position.X, Y: g.Position.Y, Z: g.Position.Z}
		RoundPositions[rounds].GrenadeEvents[tick][id] = GrenadeState{
			Position: position, Grenade: g.Grenade.String(), ThrownByName: player.Name, ThrownByid: int64(player.SteamID64),
		}
	})
	p.RegisterEventHandler(func(g events.SmokeExpired) {
		if live != true {
			return
		}
		gs := p.GameState()
		tick := gs.IngameTick()
		id := g.GrenadeEntityID
		player := g.Thrower
		grenadeBuffer = append(grenadeBuffer, GrenadeEntry{
			matchid, rounds, tick, id, player.SteamID64, g.Position.X, g.Position.Y, g.Position.Z, g.Grenade.String(), "EXPIRED",
		})
		if len(RoundPositions[rounds].PlayerPositions[tick]) == 0 {
			RoundPositions[rounds].GrenadeEvents[tick] = make(map[int]GrenadeState)

		}
		position := r3.Vector{X: g.Position.X, Y: g.Position.Y, Z: g.Position.Z}
		RoundPositions[rounds].GrenadeEvents[tick][id] = GrenadeState{
			Position: position, Grenade: g.Grenade.String(), ThrownByName: player.Name, ThrownByid: int64(player.SteamID64),
		}
	})

	p.RegisterEventHandler(func(g events.HeExplode) {
		if live != true {
			return
		}
		gs := p.GameState()
		tick := gs.IngameTick()
		id := g.GrenadeEntityID
		player := g.Thrower
		grenadeBuffer = append(grenadeBuffer, GrenadeEntry{
			matchid, rounds, tick, id, player.SteamID64, g.Position.X, g.Position.Y, g.Position.Z, g.Grenade.String(), "EXPIRED",
		})
		if len(RoundPositions[rounds].PlayerPositions[tick]) == 0 {
			RoundPositions[rounds].GrenadeEvents[tick] = make(map[int]GrenadeState)

		}
		position := r3.Vector{X: g.Position.X, Y: g.Position.Y, Z: g.Position.Z}
		RoundPositions[rounds].GrenadeEvents[tick][id] = GrenadeState{
			Position: position, Grenade: g.Grenade.String(), ThrownByName: player.Name, ThrownByid: int64(player.SteamID64),
		}
	})

	p.RegisterEventHandler(func(g events.FlashExplode) {
		if live != true {
			return
		}
		gs := p.GameState()
		tick := gs.IngameTick()
		id := g.GrenadeEntityID
		player := g.Thrower
		grenadeBuffer = append(grenadeBuffer, GrenadeEntry{
			matchid, rounds, tick, id, player.SteamID64, g.Position.X, g.Position.Y, g.Position.Z, g.Grenade.String(), "EXPIRED",
		})
		if len(RoundPositions[rounds].PlayerPositions[tick]) == 0 {
			RoundPositions[rounds].GrenadeEvents[tick] = make(map[int]GrenadeState)

		}
		position := r3.Vector{X: g.Position.X, Y: g.Position.Y, Z: g.Position.Z}
		RoundPositions[rounds].GrenadeEvents[tick][id] = GrenadeState{
			Position: position, Grenade: g.Grenade.String(), ThrownByName: player.Name, ThrownByid: int64(player.SteamID64),
		}
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
	bufferSize := len(playerBuffer) + len(grenadeBuffer) + len(fireBuffer)
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
	close(posBatch)
	close(grenadeBatch)
	close(fireBatch)
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
func flushToDB(db *sql.DB, entries []posEntry) {
	tx, err := db.Begin()
	if err != nil {
		panic(err)
	}

	Players := make(map[uint64]int)

	for _, e := range entries {
		_, err = DB.Exec("INSERT IGNORE INTO ROUNDS (MATCHID, ROUND_NO) VALUES (?,?)", e.matchID, e.roundNo)
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

	stmt, err := tx.Prepare("INSERT INTO PLAYER_EVENTS (MATCHID,ROUND_NO,PLAYERID,HP,WEAPON,HAS_BOMB,P_ACTION,KILLS,ASSIST,DEATHS,ARMOR,DINERO,XPOS,YPOS,ZPOS,TICK) VALUES" +
		"(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)" +
		"ON DUPLICATE KEY UPDATE WEAPON=(WEAPON), XPOS=VALUES(XPOS), YPOS=VALUES(YPOS), ZPOS=VALUES(ZPOS)")
	if err != nil {
		panic(err)
	}
	defer stmt.Close()

	for _, e := range entries {
		// fmt.Printf("MID:%v, ROUND_NO:%v, STEAM:%v, WEAPON:%v, X:%v, Y:%v, Z:%v, TICK:%v\n", e.matchID, e.roundNo, e.steamID, e.weapon, e.x, e.y, e.z, e.tick)
		if _, err := stmt.Exec(e.matchID, e.roundNo, e.steamID, e.hp, e.weapon, e.hasBomb, e.action, e.kills, e.assists, e.deaths, e.armor, e.money, e.x, e.y, e.z, e.tick); err != nil {
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
