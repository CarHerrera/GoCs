package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	fiberRecover "github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/static"
	"github.com/golang-jwt/jwt/v5"
	"github.com/golang/geo/r2"
	"github.com/golang/geo/r3"
	"github.com/joho/godotenv"
	ex "github.com/markus-wa/demoinfocs-golang/v5/examples"
	dem "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/msg"
	"golang.org/x/crypto/bcrypt"
)

func getPlayerPageData(db *sql.DB, accountID int64) (*PlayerPageData, error) {
	data := &PlayerPageData{}

	err := db.QueryRow(`
        SELECT 
            u.ACCOUNTID,
            u.USERNAME,
            u.STEAMID,
            u.STEAM_VER,
            p.PLAYERNAME,
            p.TEAMNAME
        FROM USER_ACCOUNTS u
        LEFT JOIN PLAYERS p ON p.PLAYERID = u.STEAMID
        WHERE u.ACCOUNTID = ?
    `, accountID).Scan(
		&data.AccountID,
		&data.Username,
		&data.SteamID, // sql.NullInt64 handles NULL safely
		&data.SteamVer,
		&data.PlayerName, // sql.NullString handles NULL safely
		&data.TeamName,
	)
	if err != nil {
		return nil, fmt.Errorf("getPlayerPageData: %w", err)
	}

	return data, nil
}

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
	// engine := html.New()
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using system environment variables")
	}
	Connect()
	defer DB.Close()
	app := fiber.New(fiber.Config{
		// Provide a template engine
		BodyLimit: 1 * 1024 * 1024 * 1024,
		// Views:     engine,
	})
	// had to add this for the fetch to work
	app.Use(cors.New(cors.Config{
		// Since it is running through github codespace/ssh specify both urls
		AllowOrigins:     []string{"http://localhost:5173", "http://127.0.0.1:5173/", "https://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "HEAD", "PUT", "DELETE", "PATCH"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		AllowCredentials: true,
	}))
	app.Use("/static", static.New("./static"))
	app.Use(fiberRecover.New())
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
			if err := DB.QueryRow("SELECT DEMO_NAME, SAVED_DATE, MAP, PARSED_STATS, PARSED_2D FROM MATCHES WHERE DEMO_NAME = ?", e.Name()).Scan(&row.FileName, &row.ModDate, &row.Map, &row.BaseStats, &row.Parsed); err != nil {
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
						FileName:  e.Name(),
						ModDate:   timefmted,
						Map:       GameMap,
						Parsed:    false,
						BaseStats: false,
					}
					file = append(file, infoSend)
				}
			} else {
				file = append(file, row)
			}

		}
		return c.Status(200).JSON(file)
	})

	app.Get("/2DPlayback/:demoName-:roundNo<int>", func(c fiber.Ctx) error {
		var matchid, parsed2d, rounds int
		var gamemap string
		log.Printf("RECEIVED")
		if c.Params("demoName") == "" {
			log.Printf("Empty String: %s", c.Params("demoName"))
			return c.SendStatus(fiber.StatusNotFound)
		}
		log.Printf("%s", c.Params("demoName"))
		if err := DB.QueryRow("SELECT MATCHID, MAP, PARSED_2D, (TEAM_A_FINAL_SCORE+ TEAM_B_FINAL_SCORE) as ROUND_TOTAL FROM MATCHES WHERE DEMO_NAME = ?", c.Params("demoName")).Scan(&matchid, &gamemap, &parsed2d, &rounds); err != nil {
			if err == sql.ErrNoRows {
				log.Print("SQL ERROR")
				return c.SendStatus(fiber.StatusNotFound)
			} else {
				panic(err)
			}

		} else {
			if (parsed2d) == 1 {
				var me MatchEvents
				// fmt.Print("HAS BEEN PARSED")
				log.Printf("GETTING SQL RESULTS")
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
				me.MapMeta = ex.GetMapMetadata(gamemap)

				query = `SELECT p.PLAYERID, p.PLAYERNAME, pe.HP, pe.ACTIVE_WEAPON, pe.HAS_BOMB, pe.P_ACTION,
							pe.KILLS, pe.ASSISTS, pe.DEATHS, pe.ARMOR, pe.DINERO,
							pe.PRIMARY_SLOT, pe.SECONDARY_SLOT, pe.SMOKE_SLOT, pe.FIRE_SLOT, pe.HE_SLOT, pe.DECOY_SLOT, 
							pe.FLASH_SLOT1, pe.FLASH_SLOT2, pe.FLASHED_DURATION, pe.VIEW_ANGLE,
							pe.XPOS, pe.YPOS, pe.ZPOS, pe.TICK, rp.SIDE
					from PLAYER_EVENTS as pe 
					JOIN PLAYERS p on p.PLAYERID = pe.PLAYERID 
					JOIN ROUND_PARTICIPANTS rp ON pe.MATCHID = rp.MATCHID AND pe.ROUND_NO = rp.ROUND_NO AND pe.PLAYERID = rp.PLAYERID 
					WHERE pe.MATCHID = ? AND pe.ROUND_NO = ? ORDER BY TICK ASC`
				var R RoundInfo
				R.PlayerPositions = make(map[int]map[int64]PlayerState)
				R.PlayerNames = make(map[int64]PlayerInfo)
				R.GrenadeEvents = make(map[int]map[int]GrenadeState)
				R.FirePositions = make(map[int]map[int]FireState)
				R.RoundTimeline = make(map[int]RoundEvent)
				rows, err := DB.Query(query, matchid, c.Params("roundNo"))
				if err != nil {
					panic(err)
				}
				for rows.Next() {
					var Name string
					var hasBomb bool
					var tick, side, hp, kills, assist, deaths, armor, dinero int
					var primary, secondary, smoke, fire, he, decoy, flash1, flash2, weapon int
					var action PlayerAction
					var x, y, z, flashedDur float64
					var playerid int64
					var view_angle float32
					rows.Scan(&playerid, &Name, &hp, &weapon, &hasBomb, &action, &kills, &assist, &deaths, &armor,
						&dinero, &primary, &secondary, &smoke, &fire, &he, &decoy, &flash1, &flash2, &flashedDur, &view_angle,
						&x, &y, &z, &tick, &side)
					position := r3.Vector{X: x, Y: y, Z: z}
					if _, ok := R.PlayerNames[playerid]; !ok {
						R.PlayerNames[playerid] = PlayerInfo{Name: Name, Side: side}
					}
					// No Tick has been created
					if len(R.PlayerPositions[tick]) == 0 {
						R.PlayerPositions[tick] = make(map[int64]PlayerState)
					}
					R.PlayerPositions[tick][playerid] = PlayerState{
						Position: position, Active_Weapon: weapon, HP: hp,
						Kills: kills, Assists: assist, Deaths: deaths,
						Armor: armor, Money: dinero, HasBomb: hasBomb,
						Action: action, Primary: primary, Secondary: secondary, SmokeSlot: smoke,
						FireSlot: fire, HESlot: he, DecoySlot: decoy, Flashslot1: flash1, FlashSlot2: flash2,
						BlindDuration: flashedDur, ViewAngle: view_angle,
					}

					defer rows.Close()
				}
				query = `SELECT p.PLAYERID, GE.ENTITYID, p.PLAYERNAME, GE.GRENADE, GE.XPOS, GE.YPOS, GE.ZPOS, GE.TICK, GE.ENTSTATE
				from GRENADE_EVENTS as GE 
				JOIN PLAYERS p on p.PLAYERID = GE.PLAYERID 
				WHERE GE.MATCHID = ? AND GE.ROUND_NO = ? ORDER BY TICK ASC;`
				rows, err = DB.Query(query, matchid, c.Params("roundNo"))
				for rows.Next() {
					var Name, status string
					var tick, grenadeid, grenade int
					var x, y, z float64
					var playerid int64
					rows.Scan(&playerid, &grenadeid, &Name, &grenade, &x, &y, &z, &tick, &status)
					position := r3.Vector{X: x, Y: y, Z: z}

					// No Tick has been created
					if len(R.GrenadeEvents[tick]) == 0 {
						R.GrenadeEvents[tick] = make(map[int]GrenadeState)
					}
					R.GrenadeEvents[tick][grenadeid] = GrenadeState{
						Position: position, Grenade: grenade, ThrownByName: Name, ThrownByid: playerid, Status: status,
					}

					defer rows.Close()
				}

				query = `SELECT ENTITYID, FIREID, TICK, XPOS, YPOS, ENTSTATE 
				FROM FIRE_VERTICES
				WHERE MATCHID = ? AND ROUND_NO = ?
				ORDER BY TICK, FIREID`
				rows, err = DB.Query(query, matchid, c.Params("roundNo"))
				for rows.Next() {
					var entid, fireid, tick int
					var x, y float64
					var state string
					rows.Scan(&entid, &fireid, &tick, &x, &y, &state)
					position := r2.Point{X: x, Y: y}
					if len(R.FirePositions[tick]) == 0 {
						R.FirePositions[tick] = make(map[int]FireState)
					}

					R.FirePositions[tick][entid] = FireState{
						Vertices: append(R.FirePositions[tick][entid].Vertices, position), Status: state,
					}
				}
				query = `SELECT TICK, EVENT_TYPE, PLAYER1ID, PLAYER2ID
				FROM ROUND_EVENTS
				WHERE MATCHID = ? AND ROUND_NO = ?
				ORDER BY TICK`
				rows, err = DB.Query(query, matchid, c.Params("roundNo"))
				for rows.Next() {
					var tick, etype int
					var p1id, p2id int64
					rows.Scan(&tick, &etype, &p1id, &p2id)
					if len(R.FirePositions[tick]) == 0 {
						R.FirePositions[tick] = make(map[int]FireState)
					}

					R.RoundTimeline[tick] = RoundEvent{
						Player1: int64(p1id), Player2: int64(p2id), Event: TrackedEvents(etype),
					}
				}
				me.RoundPositions = R
				me.Rounds = rounds
				// fmt.Printf("Out:%v", me.RoundPositions[1])
				return c.Status(200).JSON(me)
			} else {
				log.Printf("PARSING")

				// ← ADD THIS RECOVERY FUNCTION (lines below)
				var info MatchEvents
				info = Parse2D(c.Params("demoName"))
				log.Printf("DONE")
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
			log.Printf("SQL ERROR")
			return c.Status(500).JSON(fiber.Map{
				"message": "Failed to SQL Error",
				"success": "false",
			})
		} else {
			log.Printf("File found in DB. MATCH ID: %v", row.ID)
			if parsed == 0 {
				log.Printf("Parsing Stats")
				demo_stats, err := parse_demo_stats(demo, row.ID)
				if err != nil {
					log.Printf("%v failed to parse. %v", demo, err)
					return c.Status(500).JSON(fiber.Map{
						"message": "Failed to parse error",
						"success": "false",
					})
				}
				return c.Status(200).JSON(demo_stats.TeamStats)
			} else {
				log.Printf("Grabbing from SQL")
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

	app.Get("/auth/steam", func(c fiber.Ctx) error {
		params := url.Values{
			"openid.ns":         {"http://specs.openid.net/auth/2.0"},
			"openid.mode":       {"checkid_setup"},
			"openid.return_to":  {"http://localhost:4000/auth/steam/callback"},
			"openid.realm":      {"http://localhost:4000"},
			"openid.identity":   {"http://specs.openid.net/auth/2.0/identifier_select"},
			"openid.claimed_id": {"http://specs.openid.net/auth/2.0/identifier_select"},
		}
		steamurl := "https://steamcommunity.com/openid/login?"
		steamLoginURL := steamurl + params.Encode()
		return c.Redirect().Status(303).To(steamLoginURL)
	})
	app.Get("/auth/steam/callback", func(c fiber.Ctx) error {

		// fullURL := "http://localhost:4000" + c.OriginalURL()
		// log.Println("Full URL:", fullURL)
		params, err := url.ParseQuery(string(c.Request().URI().QueryString()))
		if err != nil {
			log.Println("Failed to parse query params:", err)
			return c.Status(400).SendString("Bad request")
		}
		steamAPIKey := os.Getenv("STEAM_API")
		// Steam uses "dumb mode" — change openid.mode to check_authentication
		// and POST it back to Steam. This is the manual verification step.
		params.Set("openid.mode", "check_authentication")

		// Make the server-to-server POST back to Steam
		resp, err := http.PostForm("https://steamcommunity.com/openid/login", params)
		if err != nil {
			log.Println("Failed to contact Steam for verification:", err)
			return c.Status(500).SendString("Could not contact Steam")
		}
		defer resp.Body.Close()

		// Read Steam's response — it's plain text
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Println("Failed to read Steam response:", err)
			return c.Status(500).SendString("Could not read Steam response")
		}

		log.Println("Steam verification response:", string(body))

		// Check if Steam confirmed the login is valid
		if !strings.Contains(string(body), "is_valid:true") {
			log.Println("Steam said login is invalid")
			return c.Status(401).SendString("Invalid login")
		}

		// Extract the Steam ID from the claimed_id URL
		// Format: https://steamcommunity.com/openid/id/76561198XXXXXXXXX
		claimedID := params.Get("openid.claimed_id")
		steamID := path.Base(claimedID)
		log.Println("Verified Steam ID:", steamID)
		url := fmt.Sprintf(
			"https://api.steampowered.com/ISteamUser/GetPlayerSummaries/v0002/?key=%s&steamids=%s",
			steamAPIKey,
			steamID,
		)

		// Create an HTTP client with a timeout
		client := &http.Client{Timeout: 10 * time.Second}

		resp, err = client.Get(url)
		if err != nil {
			return c.Status(500).SendString("Failed to get steam profile")
		}

		defer resp.Body.Close()
		var result SteamResponse

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return c.Status(500).SendString("Failed to decode steam profile")
		}
		var accountID int64
		if err := DB.QueryRow("SELECT ACCOUNTID FROM USER_ACCOUNTS WHERE STEAMID = ?", steamID).Scan(&accountID); err != nil {
			if err == sql.ErrNoRows {
				c.Cookie(&fiber.Cookie{
					Name:     "new_account",
					Value:    "true",
					HTTPOnly: false,
					Expires:  time.Now().Add(1 * time.Minute),
					SameSite: "Lax",
				})
				return c.Redirect().Status(303).To("http://localhost:5173/")
			}
		}
		token, err := generateJWT(accountID)
		if err != nil {
			log.Println("Failed to generate JWT:", err)
			return c.Status(500).SendString("Failed to generate token")
		}
		if len(result.Response.Players) == 0 {
			return c.Status(500).SendString("no players found for steamID:" + steamID)
		}
		log.Printf("RESPONSE: %v", result)
		c.Cookie(&fiber.Cookie{
			Name:     "auth_token",
			Value:    token,
			HTTPOnly: true,
			Expires:  time.Now().Add(24 * time.Hour),
			// SameSite must be Lax here because we're redirecting from Steam's domain
			// Strict would block the cookie since the request is cross-site
			SameSite: "Lax",
		})
		return c.Redirect().Status(303).To("http://localhost:5173/accountHome")
	})
	app.Post("/auth/register", func(c fiber.Ctx) error {
		var result AccountRegister
		if err := c.Bind().Body(&result); err != nil {
			return c.Status(400).SendString("Failed to decode steam profile")
		}
		var email string
		err := DB.QueryRow("SELECT EMAIL FROM USER_ACCOUNTS WHERE EMAIL = ?", result.Email).Scan(&email)
		if err == nil {
			log.Printf("DB error checking email: %v val:%v", err, result)
			return c.Status(409).SendString("Email already in use. Please choose another.")

		}
		if err != sql.ErrNoRows {
			return c.Status(500).SendString("Something went wrong. Please try again.")
		}
		hash, err := hashPassword(result.Password)
		if err != nil {
			log.Printf("Hash error: %v", err)
			return c.Status(500).SendString("Something went wrong. Please try again.")
		}

		// Insert and get the new user's ID back
		res, err := DB.Exec(
			"INSERT INTO USER_ACCOUNTS (EMAIL, USER_PASSWORD, USERNAME) VALUES (?, ?, ?)",
			result.Email, hash, result.Username,
		)
		if err != nil {
			log.Printf("Insert error: %v", err)
			return c.Status(500).SendString("Something went wrong while creating the account.")
		}

		userID, err := res.LastInsertId()
		if err != nil {
			log.Printf("LastInsertId error: %v", err)
			return c.Status(500).SendString("Something went wrong. Please try again.")
		}

		// Create JWT
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"user_id": userID,
			"exp":     time.Now().Add(7 * 24 * time.Hour).Unix(), // expires in 7 days
		})
		tokenString, err := token.SignedString([]byte(os.Getenv("SECURE_TOKEN")))
		if err != nil {
			log.Printf("JWT error: %v", err)
			return c.Status(500).SendString("Something went wrong. Please try again.")
		}

		// Send token as httpOnly cookie
		c.Cookie(&fiber.Cookie{
			Name:     "auth_token",
			Value:    tokenString,
			HTTPOnly: true, // JS can't read it — protects against XSS
			Secure:   true, // HTTPS only
			SameSite: "Lax",
			MaxAge:   7 * 24 * 60 * 60,
		})

		return c.Status(200).JSON(fiber.Map{
			"message":  "Account created successfully",
			"username": result.Username,
		})
	})

	app.Post("/api/player/link-steam", func(c fiber.Ctx) error {
		userID, err := getUserIDFromCookie(c)
		log.Printf("SUBMITTED ID: %v", userID)
		if err != nil {
			log.Printf("Error: %v", err)
			return c.Status(401).SendString("Unauthorized")
		}
		var req LinkSteamRequest
		if err := c.Bind().Body(&req); err != nil {
			log.Printf("Error: %v", err)
			return c.Status(400).SendString("Invalid request body")
		}
		if len(req.SteamID) != 17 {
			return c.Status(400).SendString("Invalid Steam ID — must be 17 digits")
		}
		for _, ch := range req.SteamID {
			if ch < '0' || ch > '9' {
				return c.Status(400).SendString("Invalid Steam ID — must be numeric")
			}
		}

		// 4. Check that Steam ID isn't already linked to another account
		var existingUser int64
		err = DB.QueryRow(
			"SELECT STEAMID FROM USER_ACCOUNTS WHERE STEAMID = ?", req.SteamID,
		).Scan(&existingUser)
		if err == nil {
			// Scan succeeded → someone already has this Steam ID
			return c.Status(409).SendString("This Steam ID is already linked to another account")
		}
		if err != sql.ErrNoRows {

			log.Printf("DB error checking steamid: %v", err)
			return c.Status(500).SendString("Something went wrong. Please try again.")
		}

		// 5. Update the record
		_, err = DB.Exec(
			"UPDATE USER_ACCOUNTS SET STEAMID = ? WHERE ACCOUNTID = ?",
			req.SteamID, userID,
		)
		if err != nil {
			log.Printf("Update error: %v", err)
			return c.Status(500).SendString("Failed to link Steam account")
		}

		return c.Status(200).JSON(fiber.Map{
			"message": "Steam account linked successfully",
		})
	})
	app.Get("/api/player/me", func(c fiber.Ctx) error {
		accountID, err := getUserIDFromCookie(c)
		if err != nil {
			return c.Status(401).SendString("Unauthorized")
		}

		data, err := getPlayerPageData(DB, accountID)
		if err != nil {
			return c.Status(500).SendString("Failed to load player data")
		}

		// Tell the frontend exactly what state they're in
		response := fiber.Map{
			"username":    data.Username,
			"steamLinked": data.SteamID.Valid,
			"hasMatches":  data.PlayerName.Valid,
		}

		if data.SteamID.Valid {
			steamAPIKey := os.Getenv("STEAM_API")
			url := fmt.Sprintf(
				"https://api.steampowered.com/ISteamUser/GetPlayerSummaries/v0002/?key=%s&steamids=%d",
				steamAPIKey,
				data.SteamID.Int64,
			)
			client := &http.Client{Timeout: 10 * time.Second}
			resp, err := client.Get(url)
			if err == nil {
				defer resp.Body.Close()
				var result SteamResponse
				if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
					if len(result.Response.Players) > 0 {
						response["profilePic"] = result.Response.Players[0].Avatar
					}
				}
			}

			query := `SELECT SUM(TOTAL_KILLS) AS KILLS, SUM(TOTAL_DEATHS) AS DEATHS, SUM(TOTAL_ASSISTS) AS ASSISTS, COUNT(*) AS APPEARANCES FROM MATCH_STATS WHERE PLAYERID = ?`
			var k, a, d, app int
			erm := DB.QueryRow(query, data.SteamID.Int64).Scan(&k, &d, &a, &app)
			if erm == nil {
				// Create the object to send
				response["stats"] = fiber.Map{
					"appearances": app, "kills": k, "assists": a, "deaths": d, "KD": float64(k) / float64(d),
				}
			} else {
				log.Printf("Err :%v", erm)
			}
			query = `SELECT
				m.DEMO_NAME,
				m.MAP,
				m.TEAM_A_NAME as 'Team A',
				m.TEAM_B_NAME as 'Team B',
				m.TEAM_A_FINAL_SCORE,
				m.TEAM_B_FINAL_SCORE,
				ms.TOTAL_KILLS,
				ms.TOTAL_DEATHS,
				ms.TOTAL_ASSISTS,
				p.PLAYERNAME,
				CASE 
					WHEN p.TEAMNAME = m.TEAM_A_NAME THEN 'Team A'
					WHEN p.TEAMNAME = m.TEAM_B_NAME THEN 'Team B'
					ELSE 'Unknown'
				END AS PLAYER_TEAM
			FROM MATCHES m
			JOIN MATCH_STATS ms ON m.MATCHID = ms.MATCHID
			JOIN PLAYERS p ON ms.PLAYERID = p.PLAYERID
			WHERE ms.PLAYERID = ?
			ORDER BY m.SAVED_DATE DESC
			LIMIT 5`
			player_matches, err := DB.Query(query, data.SteamID.Int64)
			if err == nil {
				recentMatches := []PlayerMatch{}
				for player_matches.Next() {
					var filename, gamemap, teama, teamb, name, team string
					var teama_score, teamb_score, k, a, d int
					player_matches.Scan(&filename, &gamemap, &teama, &teamb, &teama_score, &teamb_score, &k, &a, &d, &name, &team)
					var pm PlayerMatch
					if team == "Team A" {
						var res string
						if teama_score > teamb_score {
							res = "win"
						} else {
							res = "loss"
						}
						pm = PlayerMatch{
							Opponent: teamb, Score: fmt.Sprintf("%v-%v", teama_score, teamb_score), Result: res, Map: gamemap,
							Kills: k, Assists: a, Deaths: d, FileName: filename,
						}
					} else {
						var res string
						if teama_score > teamb_score {
							res = "loss"
						} else {
							res = "win"
						}
						pm = PlayerMatch{
							Opponent: teama, Score: fmt.Sprintf("%v-%v", teamb_score, teama_score), Result: res, Map: gamemap,
							Kills: k, Assists: a, Deaths: d, FileName: filename,
						}
					}
					recentMatches = append(recentMatches, pm)
				}
				response["recentMatches"] = recentMatches
			} else {
				log.Printf("ERROR :%v", err)
			}
			response["steamId"] = data.SteamID.Int64
		}
		if data.PlayerName.Valid {
			response["playerName"] = data.PlayerName.String
			response["teamName"] = data.TeamName.String
		}
		return c.JSON(response)
	})
	app.Post("/auth/login", func(c fiber.Ctx) error {
		var req AccountRegister
		if err := c.Bind().Body(&req); err != nil {
			return c.Status(400).SendString("Invalid request body")
		}

		// Look up the account by email
		var accountID int64
		var storedHash string
		err := DB.QueryRow(
			"SELECT ACCOUNTID, USER_PASSWORD FROM USER_ACCOUNTS WHERE EMAIL = ?",
			req.Email,
		).Scan(&accountID, &storedHash)

		if err == sql.ErrNoRows {
			return c.Status(401).SendString("Invalid email or password")
		}
		if err != nil {
			log.Printf("Login DB error: %v", err)
			return c.Status(500).SendString("Something went wrong. Please try again.")
		}

		// Check the password against the stored hash
		if !checkPassword(req.Password, storedHash) {
			return c.Status(401).SendString("Invalid email or password")
		}

		// Generate JWT using the numeric ACCOUNTID
		tokenString, err := generateJWT(accountID)
		if err != nil {
			log.Printf("JWT error: %v", err)
			return c.Status(500).SendString("Something went wrong. Please try again.")
		}

		c.Cookie(&fiber.Cookie{
			Name:     "auth_token",
			Value:    tokenString,
			HTTPOnly: true,
			Secure:   true,
			SameSite: "Lax",
			MaxAge:   7 * 24 * 60 * 60,
		})

		return c.Status(200).JSON(fiber.Map{
			"message": "Logged in successfully",
		})
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
func hashPassword(password string) (string, error) {
	// Cost of 12 is a reasonable default (2^12 iterations)
	// Higher = slower = more secure, but more CPU
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)

	return string(bytes), err
}

// When user logs in:
func checkPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
func generateJWT(id int64) (string, error) {
	key := []byte(os.Getenv("SECURE_TOKEN"))
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss":     "carlos-goback-server",
		"user_id": id, // now numeric — matches getUserIDFromCookie
		"exp":     time.Now().Add(7 * 24 * time.Hour).Unix(),
	})
	return t.SignedString(key)
}
func getUserIDFromCookie(c fiber.Ctx) (int64, error) {
	tokenString := c.Cookies("auth_token")
	if tokenString == "" {
		return 0, fmt.Errorf("no token")
	}

	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(os.Getenv("SECURE_TOKEN")), nil
	})
	if err != nil || !token.Valid {
		return 0, fmt.Errorf("invalid token: %w", err)
	}

	claims := token.Claims.(jwt.MapClaims)
	userID := int64(claims["user_id"].(float64)) // JWT numbers are float64
	return userID, nil
}
