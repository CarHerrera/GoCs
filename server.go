package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang/geo/r3"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/static"
	"github.com/gofiber/template/html/v2"
	"github.com/joho/godotenv"
	ex "github.com/markus-wa/demoinfocs-golang/v5/examples"
	dem "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/msg"
)

// var downloaded string = "/Users/carlosherrera/Documents/CS2DEMOS"
// var downloaded string = "/workspaces/GoCs/uploads"

var DB *sql.DB

func Connect() {
	// Use your environment variables here!

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
}

func main() {
	engine := html.New("./views", ".html")
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using system environment variables")
	}
	Connect()
	defer DB.Close()
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
		entries, err := os.ReadDir(getDemoPath())
		if err != nil {
			log.Fatal(err)
			return c.Status(500).JSON(fiber.Map{"error": "Could not read directory"})
		}
		file := []BaseDemo{}
		for _, e := range entries {
			if lastthree := e.Name()[len(e.Name())-3:]; lastthree != "dem" {
				continue
			}
			var row BaseDemo
			if err := DB.QueryRow("SELECT DEMO_NAME, SAVED_DATE, MAP FROM MATCHES WHERE DEMO_NAME = ?", e.Name()).Scan(&row.FileName, &row.ModDate, &row.Map); err != nil {
				if err == sql.ErrNoRows {
					path := getDemoPath() + e.Name()
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
					year, month, day := info.ModTime().Local().Date()
					timefmted := fmt.Sprintf("%v-%v-%v", year, int(month), day)
					_, err := DB.Exec("INSERT IGNORE INTO MATCHES (DEMO_NAME, MAP, SAVED_DATE, PARSED_STATS, PARSED_2D) VALUES (?,?,?,0,0)", e.Name(), GameMap, timefmted)
					if err != nil {
						panic(err)
					}
					infoSend := BaseDemo{
						FileName: e.Name(),
						ModDate:  timefmted,
						Map:      GameMap,
					}
					file = append(file, infoSend)
				}
			} else {
				file = append(file, row)
			}

		}
		return c.Status(200).JSON(file)
	})

	app.Get("/2DPlayback/:demoName", func(c fiber.Ctx) error {
		var matchid, parsed2d, rounds int
		var gamemap string

		if err := DB.QueryRow("SELECT MATCHID, MAP, PARSED_2D, (TEAM_A_FINAL_SCORE+ TEAM_B_FINAL_SCORE) as ROUND_TOTAL FROM MATCHES WHERE DEMO_NAME = ?", c.Params("demoName")).Scan(&matchid, &gamemap, &parsed2d, &rounds); err != nil {
			panic(err)
		} else {
			if (parsed2d) == 1 {
				var me MatchEvents
				// fmt.Print("HAS BEEN PARSED")
				query := `
				SELECT p.PLAYERNAME, p.PLAYERID, p.TEAMNAME as TEAM 
				FROM PLAYERS as p 
				LEFT JOIN MATCHES m ON (p.TEAMNAME = m.TEAM_A_NAME OR m.TEAM_B_NAME = p.TEAMNAME) 
				WHERE MATCHID = ? 
				ORDER BY TEAM ASC`
				team_response, err := DB.Query(query, matchid)
				me.Teams = make(map[string]map[int64]string)
				if err != nil {
					panic(err)
				}
				for team_response.Next() {
					var playername, team string
					var playerid int64
					team_response.Scan(&playername, &playerid, &team)
					if _, ok := me.Teams[team]; !ok {
						me.Teams[team] = make(map[int64]string)
					}
					me.Teams[team][int64(playerid)] = playername
				}
				me.RoundPositions = make(map[int]RoundEvents)
				me.MapMeta = ex.GetMapMetadata(gamemap)

				query = `SELECT p.PLAYERID, p.PLAYERNAME, re.WEAPON, re.XPOS, re.YPOS, re.ZPOS, re.TICK, rp.SIDE 
					from ROUND_EVENTS as re 
					JOIN PLAYERS p on p.PLAYERID = re.PLAYERID 
					JOIN ROUND_PARTICIPANTS rp ON re.MATCHID = rp.MATCHID AND re.ROUND_NO = rp.ROUND_NO AND re.PLAYERID = rp.PLAYERID 
					WHERE re.MATCHID = ? AND re.ROUND_NO = ? ORDER BY TICK ASC`
				for r := range rounds + 1 {
					if r == 0 {
						continue
					}
					var RE RoundEvents
					RE.PlayerPositions = make(map[int]map[int64]PlayerState)
					RE.PlayerNames = make(map[int64]PlayerInfo)
					rows, err := DB.Query(query, matchid, r)
					if err != nil {
						panic(err)
					}
					for rows.Next() {
						var Name, weapon string
						var tick, side int
						var x, y, z float64
						var playerid int64
						rows.Scan(&playerid, &Name, &weapon, &x, &y, &z, &tick, &side)
						position := r3.Vector{X: x, Y: y, Z: z}
						if _, ok := RE.PlayerNames[playerid]; !ok {
							RE.PlayerNames[playerid] = PlayerInfo{Name: Name, Side: side}
						}
						// No Tick has been created
						if len(RE.PlayerPositions[tick]) == 0 {
							RE.PlayerPositions[tick] = make(map[int64]PlayerState)
							RE.PlayerPositions[tick][playerid] = PlayerState{
								Position: position, Weapon: weapon,
							}
						} else {
							RE.PlayerPositions[tick][playerid] = PlayerState{
								Position: position, Weapon: weapon,
							}
						}
					}
					defer rows.Close()
					me.RoundPositions[r] = RE

				}

				// fmt.Printf("Out:%v", me.RoundPositions[1])
				return c.Status(200).JSON(me)
			} else {
				info := Parse2D(c.Params("demoName"))
				return c.Status(200).JSON(info)
			}

		}
	})
	app.Get("/advancedStats/:demoName", func(c fiber.Ctx) error {
		demo := c.Params("demoName")
		var row BaseDemo
		var parsed int
		if err := DB.QueryRow("SELECT DEMO_NAME, MAP, MATCHID, PARSED_STATS FROM MATCHES WHERE DEMO_NAME = ?", demo).Scan(
			&row.FileName, &row.Map, &row.ID, &parsed); err != nil {

			panic(err)
		} else {
			// log.Printf("File found in DB. MATCH ID: %v", row.ID)
			if parsed == 0 {
				demo_stats := parse_demo_stats(demo, row.ID)
				return c.Status(200).JSON(demo_stats.TeamStats)
			} else {
				query := `
					SELECT 
						m.TEAM_A_NAME,
						m.TEAM_A_T_SCORE,
						m.TEAM_A_CT_SCORE,
						m.TEAM_A_FINAL_SCORE,
						m.TEAM_B_T_SCORE,
						m.TEAM_B_CT_SCORE,
						m.TEAM_B_FINAL_SCORE,
						p.TEAMNAME,
						p.PLAYERNAME,
						p.PLAYERID,
						ms.TOTAL_KILLS,
						ms.TOTAL_ASSISTS,
						ms.TOTAL_DEATHS
					FROM MATCHES m
					JOIN MATCH_STATS ms ON m.MATCHID = ms.MATCHID
					JOIN PLAYERS p ON ms.PLAYERID = p.PLAYERID
					WHERE m.MATCHID = ?
					ORDER BY p.TEAMNAME, ms.TOTAL_KILLS DESC`
				rows, err := DB.Query(query, row.ID)
				if err != nil {
					panic(err)
				}
				var stats [2]Team
				for rows.Next() {
					var teamA string
					var teamName string
					var playerName string
					var playerID int64
					var kills int
					var assists int
					var deaths int
					var A_T_SCORE int
					var A_CT_SCORE int
					var A_FINAL_SCORE int
					var B_T_SCORE int
					var B_CT_SCORE int
					var B_FINAL_SCORE int
					rows.Scan(
						&teamA,
						&A_T_SCORE, &A_CT_SCORE, &A_FINAL_SCORE,
						&B_T_SCORE, &B_CT_SCORE, &B_FINAL_SCORE,
						&teamName, &playerName, &playerID, &kills, &assists, &deaths)

					if teamName == teamA {
						if !stats[0].inited {
							stats[0].ClanName = teamName
							stats[0].ID = 1
							stats[0].inited = true
							stats[0].EndScore = A_FINAL_SCORE
							stats[0].CTScore = A_CT_SCORE
							stats[0].TScore = A_T_SCORE
							stats[0].PlayingPlayers = make(map[string]Player)
						}

						player := Player{
							Name: playerName,
							ID:   playerID,
							Stats: PlayerStats{
								Kills:   kills,
								Deaths:  deaths,
								Assists: assists,
							},
						}
						stats[0].PlayingPlayers[playerName] = player
					} else {
						if !stats[1].inited {
							stats[1].ClanName = teamName
							stats[1].ID = 1
							stats[1].inited = true
							stats[1].EndScore = B_FINAL_SCORE
							stats[1].CTScore = B_CT_SCORE
							stats[1].TScore = B_T_SCORE
							stats[1].PlayingPlayers = make(map[string]Player)
						}
						player := Player{
							Name: playerName,
							ID:   playerID,
							Stats: PlayerStats{
								Kills:   kills,
								Deaths:  deaths,
								Assists: assists,
							},
						}
						stats[1].PlayingPlayers[playerName] = player
					}
				}
				defer rows.Close()
				return c.Status(200).JSON(stats)
			}
		}
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
