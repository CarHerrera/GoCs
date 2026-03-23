package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/static"
	"github.com/gofiber/template/html/v2"
	dem "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/msg"
)

// var downloaded string = "/Users/carlosherrera/Documents/CS2DEMOS"
var downloaded string = "/workspaces/GoCs/uploads"

type BaseDemo struct {
	FileName string `json:"filename"`
	ModDate  string `json:"date"`
	FileSize string `json:"filesize"`
	Map      string `json:"map"`
}

type PlayerStats struct {
	Kills   int16 `json:"kills"`
	Deaths  int16 `json:"deaths"`
	Assists int16 `json:"assists"`
}

type Player struct {
	Name  string      `json:"name"`
	ID    int64       `json:"ID"`
	Stats PlayerStats `json:"stats"`
}
type Team struct {
	ID             int               `json:"ID"`
	ClanName       string            `json:"Clanname"`
	EndScore       int16             `json:"Endscore"`
	TScore         int16             `json:"TScore"`
	CTScore        int16             `json:"CTScore"`
	PlayingPlayers map[string]Player `json:"Playing"`
	inited         bool
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
	app.Get("/advancedStats/:demoName", func(c fiber.Ctx) error {
		path := downloaded + "/" + c.Params("demoName")
		file, _ := os.Open(path)
		p := dem.NewParser(file)
		defer p.Close()
		defer file.Close()
		var TeamStats [2]Team
		p.RegisterEventHandler(func(e events.MatchStart) {
			GS := p.GameState()
			players := GS.Participants().Playing()
			// log.Printf("%v", players)
			for _, player := range players {
				// This should run the first time everytime
				// Generates both Teams
				// log.Printf("%v", TeamStats[0])
				if !TeamStats[0].inited {
					state := player.TeamState
					opps := player.TeamState.Opponent
					TeamStats[0] = Team{
						ID:             state.ID(),
						EndScore:       -1,
						CTScore:        0,
						TScore:         0,
						ClanName:       state.ClanName(),
						PlayingPlayers: make(map[string]Player),
						inited:         true,
					}
					TeamStats[1] = Team{
						ID:             opps.ID(),
						EndScore:       -1,
						CTScore:        0,
						TScore:         0,
						ClanName:       opps.ClanName(),
						PlayingPlayers: make(map[string]Player),
						inited:         true,
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
				} else {
					state := player.TeamState
					if TeamStats[0].ClanName == state.ClanName() {
						TeamStats[0].PlayingPlayers[player.Name] = Player{
							Name: player.Name,
							ID:   int64(player.SteamID64),
							Stats: PlayerStats{
								Kills:   0,
								Assists: 0,
								Deaths:  0,
							},
						}
					} else {
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
			}
		})

		p.ParseToEnd()
		// log.Printf("%v", TeamStats)
		return c.Status(200).JSON(TeamStats)
	})
	// app.Post("/testFile", func(c fiber.Ctx) error {
	// 	file, err := c.FormFile("myfile")
	// 	if err != nil {
	// 		return err
	// 	}
	// 	flog.Debug("Successfuk")
	// 	// Save the file to ./uploads/ directory
	// 	err = c.SaveFile(file, "./uploads/"+file.Filename)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	demo := dem.ParseFile("./uploads/"+file.Filename, func(p dem.Parser) error {
	// 		p.RegisterEventHandler(onKill)

	// 		return nil
	// 	})
	// 	if demo != nil {
	// 		log.Panic("failed to parse demo: ", demo)
	// 	}
	// 	return c.Render("index", fiber.Map{
	// 		"Title": "File uploaded successfully!",
	// 	})
	// })

	app.Listen(port)
}
