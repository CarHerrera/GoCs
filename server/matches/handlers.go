package matches

import (
	"database/sql"
	"fmt"
	"log"
	database "server/DB"
	"server/model"
	"server/parser"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/golang/geo/r2"
	"github.com/golang/geo/r3"
	ex "github.com/markus-wa/demoinfocs-golang/v5/examples"
)

func AdvancedStats(c fiber.Ctx) error {
	_, ok := c.Locals("accountID").(int64)
	if !ok {
		return c.Status(401).SendString("Unauthorized")
	}

	demoId, err := strconv.ParseInt(c.Params("matchId"), 10, 64)
	if err != nil {
		fmt.Printf("Error: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "invalid season id"})
	}
	var row model.BaseDemo
	var parsed int
	if err := database.DB.QueryRow("SELECT DEMO_NAME, MAP, MATCHID, PARSED_STATS FROM MATCHES WHERE MATCHID = ?", demoId).Scan(
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
			demo_stats, err := parser.ParseDemoStats(row.FileName, int(row.ID))
			if err != nil {
				log.Printf("%v failed to parse. %v", row.FileName, err)
				return c.Status(500).JSON(fiber.Map{
					"message": "Failed to parse error",
					"success": "false",
				})
			}
			log.Printf("Succesfully parsed basic stats of %v", row.FileName)
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
						ms.TEAMNAME,
						p.PLAYERNAME,
						p.PLAYERID,
						ms.TOTAL_KILLS,
						ms.TOTAL_ASSISTS,
						ms.TOTAL_DEATHS
					FROM MATCHES m
					JOIN MATCH_STATS ms ON m.MATCHID = ms.MATCHID
					JOIN PLAYERS p ON ms.PLAYERID = p.PLAYERID
					WHERE m.MATCHID = ?
					ORDER BY ms.TEAMNAME, ms.TOTAL_KILLS DESC`
			rows, err := database.DB.Query(query, row.ID)
			if err != nil {
				panic(err)
			}
			var stats [2]model.Team
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
					if !stats[0].Inited {
						stats[0].ClanName = teamName
						stats[0].ID = 1
						stats[0].Inited = true
						stats[0].EndScore = A_FINAL_SCORE
						stats[0].CTScore = A_CT_SCORE
						stats[0].TScore = A_T_SCORE
						stats[0].PlayingPlayers = make(map[int64]model.Player)
					}

					player := model.Player{
						Name: playerName,
						ID:   playerID,
						Stats: model.PlayerStats{
							Kills:   kills,
							Deaths:  deaths,
							Assists: assists,
						},
					}
					stats[0].PlayingPlayers[playerID] = player
				} else {
					if !stats[1].Inited {
						stats[1].ClanName = teamName
						stats[1].ID = 1
						stats[1].Inited = true
						stats[1].EndScore = B_FINAL_SCORE
						stats[1].CTScore = B_CT_SCORE
						stats[1].TScore = B_T_SCORE
						stats[1].PlayingPlayers = make(map[int64]model.Player)
					}
					player := model.Player{
						Name: playerName,
						ID:   playerID,
						Stats: model.PlayerStats{
							Kills:   kills,
							Deaths:  deaths,
							Assists: assists,
						},
					}
					stats[1].PlayingPlayers[playerID] = player
				}
			}
			defer rows.Close()
			return c.Status(200).JSON(fiber.Map{
				"stats":   stats,
				"mapName": row.Map,
			})
		}
	}
}
func TwoDPlayBack(c fiber.Ctx) error {
	var parsed2d, rounds int
	combined := c.Params("matchRound") // "86-1"
	parts := strings.Split(combined, "-")
	if len(parts) != 2 {
		return c.Status(400).JSON(fiber.Map{"error": "expected matchId-roundNo"})
	}

	matchid, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid match id"})
	}
	roundNo, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid round number"})
	}

	var gamemap string
	log.Printf("RECEIVED")
	if err := database.DB.QueryRow("SELECT MAP, PARSED_2D, (TEAM_A_FINAL_SCORE+ TEAM_B_FINAL_SCORE) as ROUND_TOTAL FROM MATCHES WHERE MATCHID = ?", matchid).Scan(&gamemap, &parsed2d, &rounds); err != nil {
		if err == sql.ErrNoRows {
			log.Print("SQL ERROR")
			return c.SendStatus(fiber.StatusNotFound)
		} else {
			panic(err)
		}

	} else {
		if (parsed2d) == 1 {
			var me model.MatchEvents
			log.Printf("GETTING SQL RESULTS")
			query := `
				SELECT p.PLAYERNAME, p.PLAYERID, ms.TEAMNAME as TEAM
				FROM PLAYERS as p
				JOIN MATCH_STATS ms ON ms.PLAYERID = p.PLAYERID
				WHERE ms.MATCHID = ?
				ORDER BY TEAM ASC`
			team_response, err := database.DB.Query(query, matchid)
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
			var R model.RoundInfo
			R.PlayerPositions = make(map[int]map[int64]model.PlayerState)
			R.PlayerNames = make(map[int64]model.PlayerInfo)
			R.GrenadeEvents = make(map[int]map[int]model.GrenadeState)
			R.FirePositions = make(map[int]map[int]model.FireState)
			R.RoundTimeline = make(map[int]model.RoundEvent)
			rows, err := database.DB.Query(query, matchid, roundNo)
			if err != nil {
				panic(err)
			}
			for rows.Next() {
				var Name string
				var hasBomb bool
				var tick, side, hp, kills, assist, deaths, armor, dinero int
				var primary, secondary, smoke, fire, he, decoy, flash1, flash2, weapon int
				var action model.PlayerAction
				var x, y, z, flashedDur float64
				var playerid int64
				var view_angle float32
				rows.Scan(&playerid, &Name, &hp, &weapon, &hasBomb, &action, &kills, &assist, &deaths, &armor,
					&dinero, &primary, &secondary, &smoke, &fire, &he, &decoy, &flash1, &flash2, &flashedDur, &view_angle,
					&x, &y, &z, &tick, &side)
				position := r3.Vector{X: x, Y: y, Z: z}
				if _, ok := R.PlayerNames[playerid]; !ok {
					R.PlayerNames[playerid] = model.PlayerInfo{Name: Name, Side: side}
				}
				if len(R.PlayerPositions[tick]) == 0 {
					R.PlayerPositions[tick] = make(map[int64]model.PlayerState)
				}
				R.PlayerPositions[tick][playerid] = model.PlayerState{
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
			rows, err = database.DB.Query(query, matchid, roundNo)
			for rows.Next() {
				var Name, status string
				var tick, grenadeid, grenade int
				var x, y, z float64
				var playerid int64
				rows.Scan(&playerid, &grenadeid, &Name, &grenade, &x, &y, &z, &tick, &status)
				position := r3.Vector{X: x, Y: y, Z: z}

				if len(R.GrenadeEvents[tick]) == 0 {
					R.GrenadeEvents[tick] = make(map[int]model.GrenadeState)
				}
				R.GrenadeEvents[tick][grenadeid] = model.GrenadeState{
					Position: position, Grenade: grenade, ThrownByName: Name, ThrownByid: playerid, Status: status,
				}

				defer rows.Close()
			}

			query = `SELECT ENTITYID, FIREID, TICK, XPOS, YPOS, ENTSTATE
				FROM FIRE_VERTICES
				WHERE MATCHID = ? AND ROUND_NO = ?
				ORDER BY TICK, FIREID`
			rows, err = database.DB.Query(query, matchid, roundNo)
			for rows.Next() {
				var entid, fireid, tick int
				var x, y float64
				var state string
				rows.Scan(&entid, &fireid, &tick, &x, &y, &state)
				position := r2.Point{X: x, Y: y}
				if len(R.FirePositions[tick]) == 0 {
					R.FirePositions[tick] = make(map[int]model.FireState)
				}

				R.FirePositions[tick][entid] = model.FireState{
					Vertices: append(R.FirePositions[tick][entid].Vertices, position), Status: state,
				}
			}
			query = `SELECT TICK, EVENT_TYPE, PLAYER1ID, PLAYER2ID
				FROM ROUND_EVENTS
				WHERE MATCHID = ? AND ROUND_NO = ?
				ORDER BY TICK`
			rows, err = database.DB.Query(query, matchid, roundNo)
			for rows.Next() {
				var tick, etype int
				var p1id, p2id int64
				rows.Scan(&tick, &etype, &p1id, &p2id)
				if len(R.FirePositions[tick]) == 0 {
					R.FirePositions[tick] = make(map[int]model.FireState)
				}

				R.RoundTimeline[tick] = model.RoundEvent{
					Player1: int64(p1id), Player2: int64(p2id), Event: model.TrackedEvents(etype),
				}
			}
			me.RoundPositions = R
			me.Rounds = rounds
			return c.Status(200).JSON(me)
		} else {
			log.Printf("PARSING")

			var info model.MatchEvents
			info = parser.Parse2D(int(matchid))
			log.Printf("DONE")
			return c.Status(200).JSON(info)
		}

	}
}
