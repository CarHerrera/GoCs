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
	dem "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
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

	// Test the connection to the database
	if err := db.Ping(); err != nil {
		t.Errorf("Error connecting to the database: %v", err)
	}
	path := "/home/carlos/NAS/CS2DEMOS/"
	demo := path + "furia-vs-vitality-m1-mirage.dem"
	file, err := os.Open(demo)
	if err != nil {
		t.Error("Error opening file")
	}

	defer file.Close()
	info, err := file.Stat()
	var demoname SQLMatch
	if err := db.QueryRow("SELECT MATCHID, DEMO_NAME, SAVED_DATE, PARSED_STATS, PARSED_2D FROM MATCHES WHERE DEMO_NAME = ?", info.Name()).Scan(&demoname.MATCHID, &demoname.DEMO_NAME, &demoname.SAVED_DATE, &demoname.PARSED_STATS, &demoname.PARSED_2D); err != nil {

		if err == sql.ErrNoRows {
			// t.Errorf("No rows found for %v", info.Name())
			var TeamStats [2]Team
			lrth := false
			catch := true
			p := dem.NewParser(file)
			defer p.Close()
			p.RegisterEventHandler(func(e events.MatchStartedChanged) {
				GS := p.GameState()
				ctside := GS.TeamCounterTerrorists()
				tside := GS.TeamTerrorists()
				// start = true
				if GS.GamePhase() == common.GamePhaseStartGamePhase {
					log.Print("DEBUG MATCH STARTED")

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
								StartingSide:   common.TeamTerrorists,
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
								StartingSide:   common.TeamCounterTerrorists,
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
				// log.Print("SIDES HAVE SWITCHED")
				temp := TeamStats[0].ID
				TeamStats[0].ID = TeamStats[1].ID
				TeamStats[1].ID = temp
				catch = true
			})
			p.RegisterEventHandler(func(lr events.AnnouncementLastRoundHalf) {
				// log.Print("LAST ROUND TILL HALF")
				lrth = true
			})
			p.RegisterEventHandler(func(r events.RoundEnd) {
				// log.Print("ROUND ENDED")
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
				team2 := score.TeamState.Opponent
				log.Printf("%v %s %v - %v %s %v", team1.ID(), team1.ClanName(), team1.Score(),
					team2.Score(), team2.ClanName(), team2.ID())

				// Check to make sure it isn't null
				if TeamStats[0].inited && catch {
					// team1 (non opp) will always have the score incremented
					// log.Printf("%v", team1.Team())

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
					// log.Printf("DEBUG %v %s CT: %v T:%v", TeamStats[0].ID, TeamStats[0].ClanName, TeamStats[0].CTScore, TeamStats[0].TScore)
					// log.Printf("DEBUG %v %s CT: %v T:%v", TeamStats[1].ID, TeamStats[1].ClanName, TeamStats[1].CTScore, TeamStats[1].TScore)
				}
			})
			err := p.ParseToEnd()
			if err != nil {
				panic(err)
			}
			// We found a demo that has yet to be parsed. We went through the demo to get all relevant stats
			// First find if the team is in the DB
			// Intialize the MATCHES with the info from TEAM_STATS
			// Then intialize the player stats
			var teamname string
			if err := db.QueryRow("SELECT TEAMNAME FROM TEAMS WHERE TEAMNAME = ?", TeamStats[0].ClanName).Scan(&teamname); err != nil {
				if err == sql.ErrNoRows {
					_, err := db.Exec("INSERT INTO TEAMS (TEAMNAME) VALUES (?)", TeamStats[0].ClanName)
					if err != nil {
						t.Errorf("Error: %v", err)
					}
				} else {
					t.Errorf("Error: %v", err)
				}
			}

			if err := db.QueryRow("SELECT TEAMNAME FROM TEAMS WHERE TEAMNAME = ?", TeamStats[1].ClanName).Scan(&teamname); err != nil {
				if err == sql.ErrNoRows {
					_, err := db.Exec("INSERT INTO TEAMS (TEAMNAME) VALUES (?)", TeamStats[1].ClanName)
					if err != nil {
						t.Errorf("Error: %v", err)
					}
				} else {
					t.Errorf("Error: %v", err)
				}
			}

			time := fmt.Sprintf("%v", info.ModTime().Format(time.DateOnly))
			// t.Errorf("%v", time)
			q, err := db.Exec("INSERT INTO MATCHES (DEMO_NAME,SAVED_DATE,PARSED_STATS,PARSED_2D,"+
				"TEAM_A_NAME,TEAM_A_T_SCORE, TEAM_A_CT_SCORE, TEAM_A_FINAL_SCORE,"+
				"TEAM_B_NAME,TEAM_B_T_SCORE, TEAM_B_CT_SCORE, TEAM_B_FINAL_SCORE)"+
				"VALUES (?,?,1,0,?,?,?,?,?,?,?,?)", info.Name(), time,
				TeamStats[0].ClanName, TeamStats[0].TScore, TeamStats[0].CTScore, TeamStats[0].EndScore,
				TeamStats[1].ClanName, TeamStats[1].TScore, TeamStats[1].CTScore, TeamStats[1].EndScore,
			)
			if err != nil {
				t.Errorf("Error: %v", err)

			}
			lastId, err := q.LastInsertId()
			// This is to ensure that it has the most up to date ID
			if err != nil {
				t.Errorf("Could not get last insert ID: %v", err)
			}
			demoname.MATCHID = int(lastId)
			for i, team := range TeamStats {

				for _, player := range team.PlayingPlayers {
					var steamid int
					if err := db.QueryRow("SELECT PLAYERID FROM PLAYERS WHERE PLAYERID = ?", player.ID).Scan(&steamid); err != nil {
						if err == sql.ErrNoRows {
							_, err := db.Exec("INSERT INTO PLAYERS (PLAYERID,PLAYERNAME,TEAMNAME) VALUES (?,?,?)", player.ID, player.Name, TeamStats[i].ClanName)
							if err != nil {
								t.Errorf("Error: %v", err)
							}
						} else {
							t.Errorf("Error %v", err)
						}
					}

					_, err := db.Exec("INSERT INTO MATCH_STATS (MATCHID,PLAYERID,TOTAL_KILLS,TOTAL_DEATHS,TOTAL_ASSISTS) VALUES (?,?,?,?,?)", demoname.MATCHID, player.ID, player.Stats.Kills, player.Stats.Deaths, player.Stats.Assists)
					if err != nil {
						t.Errorf("Error: %v", err)
					}
				}
			}

		} else {
			t.Errorf("Error %v", err)
		}

	}

	// Now you can perform database operations (CRUD)
	// Example: Query data, insert rows, etc.
}
