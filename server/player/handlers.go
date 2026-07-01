package player

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"

	database "server/DB"
	"server/auth"
	"server/model"

	"github.com/gofiber/fiber/v3"
)

func Me(c fiber.Ctx) error {
	accountID, ok := c.Locals("accountID").(int64)
	if !ok {
		return c.Status(401).SendString("Unauthorized")
	}
	data, err := auth.GetPlayerPageData(database.DB, accountID)
	if err != nil {
		return c.Status(500).SendString("Failed to load player data")
	}

	response := fiber.Map{
		"username":    data.Username,
		"steamLinked": data.SteamID.Valid,
		"hasMatches":  data.PlayerName.Valid,
	}

	if data.SteamID.Valid {
		var avatarSmall, avatarFull sql.NullString
		if err := database.DB.QueryRow(
			"SELECT AVATAR_URL, AVATAR_URL_FULL FROM PLAYERS WHERE PLAYERID = ?",
			data.SteamID.Int64,
		).Scan(&avatarSmall, &avatarFull); err == nil {
			if avatarSmall.Valid {
				response["profilePic"] = avatarSmall.String
			}
			if avatarFull.Valid {
				response["profilePicfull"] = avatarFull.String
			}
		}
		response["steamId"] = strconv.FormatInt(data.SteamID.Int64, 10)
	}
	if data.PlayerName.Valid {
		response["playerName"] = data.PlayerName.String
		response["teamName"] = data.TeamName.String
	}
	return c.JSON(response)
}

func MeStats(c fiber.Ctx) error {
	accountID := c.Locals("accountID").(int64)
	data, err := auth.GetPlayerPageData(database.DB, accountID)
	if err != nil {
		return c.Status(500).SendString("Failed to load player data")
	}

	if !data.SteamID.Valid {
		return c.Status(400).SendString("No Steam account linked")
	}

	response := fiber.Map{}

	var kd float64
	query := `
		SELECT
			COALESCE(SUM(TOTAL_KILLS), 0) AS KILLS,
			COALESCE(SUM(TOTAL_DEATHS), 0) AS DEATHS,
			COALESCE(SUM(TOTAL_ASSISTS), 0) AS ASSISTS,
			COUNT(*) AS APPEARANCES
		FROM MATCH_STATS WHERE PLAYERID = ?`
	var k, a, d, app int
	if erm := database.DB.QueryRow(query, data.SteamID.Int64).Scan(&k, &d, &a, &app); erm == nil {
		if d > 0 {
			kd = float64(k) / float64(d)
		}
		response["stats"] = fiber.Map{
			"appearances": app, "kills": k, "assists": a, "deaths": d, "KD": kd,
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
				WHEN ms.TEAMNAME = m.TEAM_A_NAME THEN 'Team A'
				WHEN ms.TEAMNAME = m.TEAM_B_NAME THEN 'Team B'
				ELSE 'Unknown'
			END AS PLAYER_TEAM
		FROM MATCHES m
		JOIN MATCH_STATS ms ON m.MATCHID = ms.MATCHID
		JOIN PLAYERS p ON ms.PLAYERID = p.PLAYERID
		WHERE ms.PLAYERID = ?
		ORDER BY m.MATCH_DATE DESC
		LIMIT 5`
	player_matches, err := database.DB.Query(query, data.SteamID.Int64)
	if err == nil {
		recentMatches := []model.PlayerMatch{}
		for player_matches.Next() {
			var filename, gamemap, teama, teamb, name, team string
			var teama_score, teamb_score, k, a, d int
			player_matches.Scan(&filename, &gamemap, &teama, &teamb, &teama_score, &teamb_score, &k, &a, &d, &name, &team)
			var pm model.PlayerMatch
			if team == "Team A" {
				var res string
				if teama_score > teamb_score {
					res = "win"
				} else {
					res = "loss"
				}
				pm = model.PlayerMatch{
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
				pm = model.PlayerMatch{
					Opponent: teama, Score: fmt.Sprintf("%v-%v", teamb_score, teama_score), Result: res, Map: gamemap,
					Kills: k, Assists: a, Deaths: d, FileName: filename,
				}
			}
			recentMatches = append(recentMatches, pm)
			defer player_matches.Close()
		}
		response["recentMatches"] = recentMatches
		response["hasMatches"] = len(recentMatches) > 0
	} else {
		log.Printf("ERROR :%v", err)
	}

	return c.JSON(response)
}

func LinkSteam(c fiber.Ctx) error {
	accountID := c.Locals("accountID").(int64)
	var req model.LinkSteamRequest
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

	var existingUser int64
	err := database.DB.QueryRow(
		"SELECT STEAMID FROM USER_ACCOUNTS WHERE STEAMID = ?", req.SteamID,
	).Scan(&existingUser)
	if err == nil {
		return c.Status(409).SendString("This Steam ID is already linked to another account")
	}
	if err != sql.ErrNoRows {

		log.Printf("DB error checking steamid: %v", err)
		return c.Status(500).SendString("Something went wrong. Please try again.")
	}

	_, err = database.DB.Exec(
		"UPDATE USER_ACCOUNTS SET STEAMID = ? WHERE ACCOUNTID = ?",
		req.SteamID, accountID,
	)
	if err != nil {
		log.Printf("Update error: %v", err)
		return c.Status(500).SendString("Failed to link Steam account")
	}

	return c.Status(200).JSON(fiber.Map{
		"message": "Steam account linked successfully",
	})
}

func UpdateRole(c fiber.Ctx) error {
	accountID := c.Locals("accountID").(int64)
	data, err := auth.GetPlayerPageData(database.DB, accountID)
	if err != nil || !data.TeamName.Valid {
		return c.Status(400).JSON(fiber.Map{"error": "No team linked"})
	}
	playerID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid player id"})
	}
	var req struct {
		Role string `json:"role"`
	}
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(400).SendString("Invalid body")
	}
	var count int
	if err := database.DB.QueryRow(
		"SELECT COUNT(*) FROM MATCH_STATS WHERE PLAYERID = ? AND TEAMNAME = ?",
		playerID, data.TeamName.String,
	).Scan(&count); err != nil || count == 0 {
		return c.Status(403).SendString("Player not on your team")
	}
	if _, err := database.DB.Exec("UPDATE PLAYERS SET ROLE = ? WHERE PLAYERID = ?", req.Role, playerID); err != nil {
		return c.Status(500).SendString("Failed to update role")
	}
	return c.JSON(fiber.Map{"message": "Role updated"})
}
