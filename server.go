package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/static"
	"github.com/gofiber/template/html/v2"
	"github.com/golang/geo/r3"
	dem "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/msg"
)

// var downloaded string = "/Users/carlosherrera/Documents/CS2DEMOS"
// var downloaded string = "/workspaces/GoCs/uploads"
var downloaded string = "/home/carlos/Documents/Gitstiff/GoCs/uploads"

type BaseDemo struct {
	FileName string `json:"filename"`
	ModDate  string `json:"date"`
	FileSize string `json:"filesize"`
	Map      string `json:"map"`
}

type PlayerStats struct {
	Kills   int `json:"kills"`
	Deaths  int `json:"deaths"`
	Assists int `json:"assists"`
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
	StartingSide   common.Team       `json:"startside"`
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

func main() {
	engine := html.New("./views", ".html")

	app := fiber.New(fiber.Config{
		// Provide a template engine
		BodyLimit: 1 * 1024 * 1024 * 1024,
		Views:     engine,
	})
	// had to add this for the fetch to work
	app.Use(cors.New(cors.Config{
		// Since it is running through github codespace/ssh specify both urls
		AllowOrigins: []string{"http://localhost:5173", "http://127.0.0.1:5173/"},
		AllowMethods: []string{"GET", "POST", "HEAD", "PUT", "DELETE", "PATCH"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept"},
	}))
	app.Use("/static", static.New("./static"))
	port := ":4000"
	app.Get("/AllDemos", func(c fiber.Ctx) error {
		entries, err := os.ReadDir(downloaded)
		if err != nil {
			log.Fatal(err)
			return c.Status(500).JSON(fiber.Map{"error": "Could not read directory"})
		}
		file := []BaseDemo{}
		for _, e := range entries {
			path := downloaded + "/" + e.Name()
			demofile, _ := os.Open(path)
			p := dem.NewParser(demofile)
			defer p.Close()
			defer demofile.Close()
			var GameMap string
			p.RegisterNetMessageHandler(func(msg *msg.CSVCMsg_ServerInfo) {
				GameMap = *msg.MapName
				p.Cancel()
			})
			p.ParseToEnd()
			info, _ := e.Info()
			infoSend := BaseDemo{
				FileName: e.Name(),
				ModDate:  info.ModTime().Local().String(),
				FileSize: fmt.Sprintf("%.2f", float64(info.Size())/1024.0/1024.0*1.00),
				Map:      GameMap,
			}
			file = append(file, infoSend)

		}
		return c.Status(200).JSON(file)
	})

	app.Get("/2DPlayback/:demoName", func(c fiber.Ctx) error {
		path := downloaded + "/" + c.Params("demoName")
		file, _ := os.Open(path)
		p := dem.NewParser(file)
		defer p.Close()
		defer file.Close()
		match := make(map[int]Rounds)
		round := make(map[string]InGame)
		roundCount := 1
		p.RegisterEventHandler(func(r events.RoundEnd) {
			match[roundCount] = Rounds{
				Players: round,
			}
			roundCount += 1
		})
		resp := Match{
			GameRounds: match,
		}
		p.RegisterEventHandler(func(r events.RoundEnd) {
			// log.Print("ROUND ENDED")
			match[roundCount] = Rounds{
				Players: round,
			}
			roundCount += 1
		})
		pf, err := p.ParseNextFrame()
		for pf {
			GS := p.GameState()
			players := GS.Participants().Playing()
			for _, player := range players {
				pos := player.Position()
				if val, ok := round[player.Name]; ok {
					oldLs := val.Position
					val.Position = append(oldLs, pos)
					round[player.Name] = val
				} else {
					positions := []r3.Vector{pos}
					start := InGame{
						Position: positions,
					}
					round[player.Name] = start
				}
			}
			pf, err = p.ParseNextFrame()
		}
		if err != nil {
			panic(err)
		}
		log.Print("DONE")
		return c.Status(200).JSON(resp)
	})
	app.Get("/advancedStats/:demoName", func(c fiber.Ctx) error {
		path := downloaded + "/" + c.Params("demoName")
		file, _ := os.Open(path)
		p := dem.NewParser(file)
		defer p.Close()
		defer file.Close()
		var TeamStats [2]Team

		lrth := false
		catch := true

		// start := false
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
		return c.Status(200).JSON(TeamStats)
	})
	app.Post("/testFile", func(c fiber.Ctx) error {
		file, err := c.FormFile("myfile")
		if err != nil {
			return err
		}
		// Save the file to ./uploads/ directory
		err = c.SaveFile(file, "./uploads/"+file.Filename)
		if err != nil {
			return err
		}
		return c.SendStatus(200)
	})
	app.Listen(port)
}
