package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	// "server/internal/auth"
	database "server/DB"
	"server/auth"
	"server/matches"
	"server/model"
	"server/player"
	"server/season"
	"server/team"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	fiberRecover "github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/static"
	"github.com/joho/godotenv"
	dem "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/msg"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	database.Connect()
	defer database.DB.Close()

	app := fiber.New(fiber.Config{
		BodyLimit: 1 * 1024 * 1024 * 1024,
	})

	app.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:6969", "http://127.0.0.1:6969/", "https://localhost:6969"},
		AllowMethods:     []string{"GET", "POST", "HEAD", "PUT", "DELETE", "PATCH"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		AllowCredentials: true,
	}))
	app.Use("/static", static.New("./static"))
	app.Use(fiberRecover.New())

	auth.RegisterRoutes(app)
	player.RegisterRoutes(app)
	team.RegisterRoutes(app)
	season.RegisterRoutes(app)
	matches.RegisterRoutes(app)
	// for _, r := range app.GetRoutes() {
	// 	log.Printf("%s %s", r.Method, r.Path)
	// }
	// demo.RegisterRoutes(app)
	app.Get("/AllDemos", func(c fiber.Ctx) error {
		entries, err := os.ReadDir(auth.GetDemoPath())
		// log.Printf("%v", auth.GetDemoPath())
		if err != nil {
			log.Fatal(err)
			return c.Status(500).JSON(fiber.Map{"error": "Could not read directory"})
		}
		file := []model.BaseDemo{}
		for _, e := range entries {
			
			
			if lastthree := e.Name()[len(e.Name())-3:]; lastthree != "dem" {
				continue
			}
			// fmt.Printf("Name: %v\n", e)
			
			var row model.BaseDemo
			
			if err := database.DB.QueryRow("SELECT DEMO_NAME, MATCHID, COALESCE(MATCH_DATE, ''), COALESCE(SAVED_DATE, ''), MAP, PARSED_STATS, PARSED_2D FROM MATCHES WHERE DEMO_NAME = ? ORDER BY MATCH_DATE DESC", e.Name()).Scan(&row.FileName, &row.ID, &row.SavedDate, &row.ModDate, &row.Map, &row.BaseStats, &row.Parsed); err != nil {
				
				if err == sql.ErrNoRows {
					
					path := auth.GetDemoPath() + e.Name()
					demofile, _ := os.Open(path)
					// fmt.Printf("Name: %v\n", path)
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
					year, month, day := info.ModTime().Local().Date()
					timefmted := fmt.Sprintf("%v-%v-%v", year, int(month), day)
					res, err := database.DB.Exec("INSERT IGNORE INTO MATCHES (DEMO_NAME, MAP, SAVED_DATE, PARSED_STATS, PARSED_2D) VALUES (?,?,?,0,0)", e.Name(), GameMap, timefmted)
					if err != nil {
						panic(err)
					}
					x, _ := res.LastInsertId()

					infoSend := model.BaseDemo{
						FileName:  e.Name(),
						ModDate:   timefmted,
						Map:       GameMap,
						Parsed:    false,
						BaseStats: false,
						ID:        x,
					}
					file = append(file, infoSend)
				} else {
					fmt.Printf("%v", err)
				}
			} else {
				file = append(file, row)
			}

		}
		return c.Status(200).JSON(file)
	})

	log.Fatal(app.Listen(":4000"))
}
