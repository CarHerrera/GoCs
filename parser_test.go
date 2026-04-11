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
	fileName := "Game2Season57.dem"
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
	parse_demo_stats(fileName, 2)
	// demo := `/home/carlos/NAS/CS2DEMOS/` + fileName
	// file, err := os.Open(demo)
	// if err != nil {
	// 	t.Error("Error opening file")
	// }
	// p := dem.NewParser(file)

	// defer p.Close()
	// var TeamStats [2]Team
	// defer file.Close()
	// live := false
	// p.RegisterEventHandler(func(e events.MatchStartedChanged) {

	// 	GS := p.GameState()
	// 	ctside := GS.TeamCounterTerrorists()
	// 	tside := GS.TeamTerrorists()

	// 	// start = true
	// 	if GS.GamePhase() == common.GamePhaseStartGamePhase {
	// 		live = true
	// 		var teamname string
	// 		for _, player := range tside.Members() {
	// 			team1Name := tside.ClanName()
	// 			if team1Name == "" {
	// 				team1Name = "Team 1"
	// 			}
	// 			if !TeamStats[0].inited {
	// 				TeamStats[0] = Team{
	// 					ID:             tside.ID(),
	// 					EndScore:       -1,
	// 					CTScore:        0,
	// 					TScore:         0,
	// 					ClanName:       team1Name,
	// 					PlayingPlayers: make(map[string]Player),
	// 					inited:         true,
	// 				}
	// 				if err := DB.QueryRow("SELECT TEAMNAME FROM TEAMS WHERE TEAMNAME = ?", team1Name).Scan(&teamname); err != nil {
	// 					if err == sql.ErrNoRows {
	// 						_, err := DB.Exec("INSERT INTO TEAMS (TEAMNAME) VALUES (?)", team1Name)
	// 						if err != nil {
	// 							panic(err)
	// 						}
	// 					} else {
	// 						panic(err)
	// 					}
	// 				}
	// 			}
	// 			TeamStats[0].PlayingPlayers[player.Name] = Player{
	// 				Name: player.Name,
	// 				ID:   int64(player.SteamID64),
	// 				Stats: PlayerStats{
	// 					Kills:   0,
	// 					Assists: 0,
	// 					Deaths:  0,
	// 				},
	// 			}
	// 		}
	// 		for _, player := range ctside.Members() {
	// 			team1Name := ctside.ClanName()
	// 			if team1Name == "" {
	// 				team1Name = "Team 2"
	// 			}
	// 			if !TeamStats[1].inited {
	// 				TeamStats[1] = Team{
	// 					ID:             ctside.ID(),
	// 					EndScore:       -1,
	// 					CTScore:        0,
	// 					TScore:         0,
	// 					ClanName:       team1Name,
	// 					PlayingPlayers: make(map[string]Player),
	// 					inited:         true,
	// 				}
	// 				if err := DB.QueryRow("SELECT TEAMNAME FROM TEAMS WHERE TEAMNAME = ?", team1Name).Scan(&teamname); err != nil {
	// 					if err == sql.ErrNoRows {
	// 						_, err := DB.Exec("INSERT INTO TEAMS (TEAMNAME) VALUES (?)", team1Name)
	// 						if err != nil {
	// 							panic(err)
	// 						}
	// 					} else {
	// 						panic(err)
	// 					}
	// 				}
	// 			}
	// 			TeamStats[1].PlayingPlayers[player.Name] = Player{
	// 				Name: player.Name,
	// 				ID:   int64(player.SteamID64),
	// 				Stats: PlayerStats{
	// 					Kills:   0,
	// 					Assists: 0,
	// 					Deaths:  0,
	// 				},
	// 			}

	// 		}
	// 		t.Logf("GAME STARTED")
	// 	}
	// 	// t.Logf("# MATCH STARTED %v", TeamStats)

	// })

	// p.RegisterEventHandler(func(kill events.Kill) {
	// 	if !live {
	// 		return
	// 	}
	// 	killer := kill.Killer
	// 	asssiter := kill.Assister
	// 	victim := kill.Victim
	// 	t.Logf("Player: %s killed %s  with %v", kill.Killer, kill.Victim.Name, kill.Weapon)
	// 	if killer != nil && killer.Name != victim.Name {
	// 		team := killer.TeamState
	// 		if team.ID() == TeamStats[0].ID {
	// 			p, _ := TeamStats[0].PlayingPlayers[killer.Name]
	// 			p.Stats.Kills++
	// 			TeamStats[0].PlayingPlayers[killer.Name] = p
	// 		} else {
	// 			p, _ := TeamStats[1].PlayingPlayers[killer.Name]
	// 			p.Stats.Kills++
	// 			TeamStats[1].PlayingPlayers[killer.Name] = p
	// 		}
	// 	}
	// 	if asssiter != nil {
	// 		team := asssiter.TeamState
	// 		if team.ID() == TeamStats[0].ID {
	// 			p, _ := TeamStats[0].PlayingPlayers[asssiter.Name]
	// 			p.Stats.Assists++
	// 			TeamStats[0].PlayingPlayers[asssiter.Name] = p
	// 		} else {
	// 			p, _ := TeamStats[1].PlayingPlayers[asssiter.Name]
	// 			p.Stats.Assists++
	// 			TeamStats[1].PlayingPlayers[asssiter.Name] = p
	// 		}
	// 	}
	// 	if victim != nil {
	// 		team := victim.TeamState
	// 		if team.ID() == TeamStats[0].ID {
	// 			p, _ := TeamStats[0].PlayingPlayers[victim.Name]
	// 			p.Stats.Deaths++
	// 			TeamStats[0].PlayingPlayers[victim.Name] = p
	// 		} else {
	// 			p, _ := TeamStats[1].PlayingPlayers[victim.Name]
	// 			p.Stats.Deaths++
	// 			TeamStats[1].PlayingPlayers[victim.Name] = p
	// 		}
	// 	}

	// })

	// err = p.ParseToEnd()
	// t.Logf("%v", TeamStats)
	// if err != nil {
	// 	t.Errorf("Error %v", err)
	// }

	// 	log.Print("DONE")
	// 	return c.Status(20).JSON(resp)
}
