package main

import (
	"database/sql"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/golang/geo/r3"
	dem "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/msg"
)

type BaseDemo struct {
	FileName  string  `json:"filename"`
	ModDate   string  `json:"date"`
	FileSize  string  `json:"filesize"`
	Map       string  `json:"map"`
	TeamStats [2]Team `json:"team_stats"`
	ID        int
}

type PlayerStats struct {
	Kills   int `json:"kills"`
	Deaths  int `json:"deaths"`
	Assists int `json:"assists"`
}
type MatchEvents struct {
	RoundPositions map[int]RoundEvents `json:"rounds"`
}
type RoundEvents struct {
	Tick            int                         `json:"tick"`
	PlayerPositions map[int64][]PlayerPositions `json:"player_positions"`
	PlayerNames     map[int64]string            `json:"player_names"`
}
type PlayerPositions struct {
	Position r3.Vector `json:"position"`
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
type InGame struct {
	Position []r3.Vector `json:"Positions"`
}
type Match struct {
	GameRounds map[int]Rounds `json:"Rounds"`
}
type Rounds struct {
	Players map[string]InGame `json:"InGamePlayers"`
}

type posEntry struct {
	matchID, roundNo, tick int
	steamID                uint64
	x, y, z                float64
}

func getDemoPath() string {
	return os.Getenv("DEMO_PATH")
}

func parse_demo_stats(fileName string) BaseDemo {
	demo := getDemoPath() + fileName
	file, err := os.Open(demo)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	info, err := file.Stat()
	var TeamStats [2]Team
	lrth := false
	catch := true
	p := dem.NewParser(file)
	defer p.Close()
	var GameMap string
	p.RegisterNetMessageHandler(func(msg *msg.CSVCMsg_ServerInfo) {
		GameMap = *msg.MapName
	})
	p.RegisterEventHandler(func(e events.MatchStartedChanged) {
		GS := p.GameState()
		ctside := GS.TeamCounterTerrorists()
		tside := GS.TeamTerrorists()
		var teamname string
		// start = true
		if GS.GamePhase() == common.GamePhaseStartGamePhase {

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
								panic(err)
							}
						} else {
							panic(err)
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
								panic(err)
							}
						} else {
							panic(err)
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
	})
	// Included the following 3 to help debug why trackers weren't working.
	p.RegisterEventHandler(func(h events.TeamSideSwitch) {
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
		killer := kill.Killer
		asssiter := kill.Assister
		victim := kill.Victim
		if killer != nil {
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
	err = p.ParseToEnd()
	if err != nil {
		panic(err)
	}
	time := fmt.Sprintf("%v", info.ModTime().Format(time.DateOnly))
	q, err := DB.Exec("INSERT INTO MATCHES (DEMO_NAME,SAVED_DATE,PARSED_STATS,PARSED_2D,"+
		"TEAM_A_NAME,TEAM_A_T_SCORE, TEAM_A_CT_SCORE, TEAM_A_FINAL_SCORE,"+
		"TEAM_B_NAME,TEAM_B_T_SCORE, TEAM_B_CT_SCORE, TEAM_B_FINAL_SCORE, MAP)"+
		"VALUES (?,?,1,0,?,?,?,?,?,?,?,?,?)", info.Name(), time,
		TeamStats[0].ClanName, TeamStats[0].TScore, TeamStats[0].CTScore, TeamStats[0].EndScore,
		TeamStats[1].ClanName, TeamStats[1].TScore, TeamStats[1].CTScore, TeamStats[1].EndScore,
		GameMap,
	)
	if err != nil {
		panic(err)
	}
	lastId, err := q.LastInsertId()
	for i, team := range TeamStats {
		for _, player := range team.PlayingPlayers {
			var steamid int
			if err := DB.QueryRow("SELECT PLAYERID FROM PLAYERS WHERE PLAYERID = ?", player.ID).Scan(&steamid); err != nil {
				if err == sql.ErrNoRows {
					_, err := DB.Exec("INSERT INTO PLAYERS (PLAYERID,PLAYERNAME,TEAMNAME) VALUES (?,?,?)", player.ID, player.Name, TeamStats[i].ClanName)
					if err != nil {
						panic(err)
					}
				} else {
					panic(err)
				}
			}

			_, err := DB.Exec("INSERT INTO MATCH_STATS (MATCHID,PLAYERID,TOTAL_KILLS,TOTAL_DEATHS,TOTAL_ASSISTS) VALUES (?,?,?,?,?)", lastId, player.ID, player.Stats.Kills, player.Stats.Deaths, player.Stats.Assists)
			if err != nil {
				panic(err)
			}
		}
	}
	if err != nil {
		panic(err)
	}
	infoSend := BaseDemo{
		FileName:  info.Name(),
		ModDate:   info.ModTime().Local().String(),
		FileSize:  fmt.Sprintf("%.2f", float64(info.Size())/1024.0/1024.0*1.00),
		Map:       GameMap,
		TeamStats: TeamStats,
	}
	return infoSend
}

func Parse2D(filename string) {
	demo := getDemoPath() + filename
	file, err := os.Open(demo)
	if err != nil {
		panic(err)
	}
	p := dem.NewParser(file)
	defer p.Close()
	defer file.Close()
	var matchid int
	if err := DB.QueryRow("SELECT MATCHID FROM MATCHES WHERE DEMO_NAME = ?", filename).Scan(&matchid); err != nil {
		panic(err)
	}
	const batchSize = 5000
	var buffer []posEntry
	batchChan := make(chan []posEntry, 100)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for batch := range batchChan {
			flushToDB(DB, batch) // Your helper function with the transaction
		}
	}()
	live := false
	rounds := 0
	p.RegisterEventHandler(func(e events.MatchStartedChanged) {
		live = true
	})
	p.RegisterEventHandler(func(events.FrameDone) {
		if live {
			gs := p.GameState()
			tick := gs.IngameTick()
			if tick%16 != 0 {
				return
			}
			for _, player := range gs.Participants().Playing() {
				pos := player.Position()
				if player.IsAlive() {
					buffer = append(buffer, posEntry{
						matchid, rounds, tick, player.SteamID64, pos.X, pos.Y, pos.Z,
					})
				}

			}

		}

	})
	p.RegisterEventHandler(func(e events.RoundStart) {
		rounds += 1
		gs := p.GameState()
		_, err := DB.Exec("INSERT IGNORE INTO ROUNDS (MATCHID, ROUND_NO) VALUES (?,?)", matchid, rounds)

		if err != nil {
			panic(err)
		}
		for _, players := range gs.Participants().Playing() {
			_, err = DB.Exec("INSERT IGNORE INTO ROUND_PARTICIPANTS (MATCHID, ROUND_NO, PLAYERID) VALUES (?,?,?)", matchid, rounds, players.SteamID64)
			if err != nil {
				panic(err)
			}
		}
	})
	p.RegisterEventHandler(func(e events.RoundEnd) {
		if len(buffer) == 0 {
			return
		}

		fmt.Printf("Round %d ended. Flushing %d rows to DB...\n", rounds, len(buffer))

		// IMPORTANT: Clear the buffer for the next round
		batchToSend := make([]posEntry, len(buffer))
		copy(batchToSend, buffer)

		batchChan <- batchToSend
		buffer = buffer[:0]
	})
	err = p.ParseToEnd()
	if err != nil {
		panic(err)
	}
	if len(buffer) > 0 {
		finalBatch := make([]posEntry, len(buffer))
		copy(finalBatch, buffer)
		batchChan <- finalBatch
	}
	close(batchChan)
	wg.Wait()
	_, err = DB.Exec("UPDATE MATCHES SET PARSED_2D = 1 WHERE MATCHID = ?", matchid)

	if err != nil {
		panic(err)
	}
}
func flushToDB(db *sql.DB, entries []posEntry) {
	tx, err := db.Begin()
	if err != nil {
		panic(err)
	}

	stmt, err := tx.Prepare("INSERT INTO ROUND_EVENTS (MATCHID,ROUND_NO,PLAYERID,XPOS,YPOS,ZPOS,TICK) VALUES (?,?,?,?,?,?,?)" +
		"ON DUPLICATE KEY UPDATE XPOS=VALUES(XPOS), YPOS=VALUES(YPOS), ZPOS=VALUES(ZPOS)")
	if err != nil {
		panic(err)
	}
	defer stmt.Close()

	for _, e := range entries {
		if _, err := stmt.Exec(e.matchID, e.roundNo, e.steamID, e.x, e.y, e.z, e.tick); err != nil {
			tx.Rollback()
			panic(err)
		}
	}

	if err := tx.Commit(); err != nil {
		panic(err)
	}
}
