package team

import (
	"log"
	"math"
	"os"
	"path"
	database "server/DB"
	"server/auth"
	"server/model"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
)

func Summary(c fiber.Ctx) error {
	accountID, ok := c.Locals("accountID").(int64)
	if !ok {
		return c.Status(401).SendString("Unauthorized")
	}
	data, err := auth.GetPlayerPageData(database.DB, accountID)
	if err != nil {
		return c.Status(500).SendString("Failed to load player data")
	}
	if !data.TeamName.Valid {
		return c.Status(400).JSON(fiber.Map{"error": "No team linked to this account"})
	}

	playerQuery := `
		SELECT
		p.PLAYERID,
        p.PLAYERNAME,
		COALESCE(p.ROLE, '')                       AS ROLE,
        COALESCE(SUM(ms.TOTAL_KILLS), 0)           AS KILLS,
        COALESCE(SUM(ms.TOTAL_DEATHS), 0)          AS DEATHS,
        COALESCE(SUM(ms.TOTAL_ASSISTS), 0)         AS ASSISTS,
        COUNT(DISTINCT ms.MATCHID)                 AS MATCHES_PLAYED,
        COALESCE(SUM(ms.TOTAL_DAMAGE), 0)          AS TOTAL_DAMAGE,
        COALESCE(SUM(ms.HEADSHOTS), 0)             AS HEADSHOTS,
        COALESCE(SUM(ms.ENTRY_KILLS), 0)           AS ENTRY_KILLS,
        COALESCE(SUM(ms.ENTRY_DEATHS), 0)          AS ENTRY_DEATHS,
		COALESCE(SUM(ms.ONE_KILL_COUNT), 0)        AS ONEK,
		COALESCE(SUM(ms.TWO_KILL_COUNT), 0)        AS TWOK,
		COALESCE(SUM(ms.THREE_KILL_COUNT), 0)      AS THREEK,
		COALESCE(SUM(ms.FOUR_KILL_COUNT), 0)       AS FOURK,
		COALESCE(SUM(ms.FIVE_KILL_COUNT), 0)       AS ACE,
        COALESCE(SUM(ms.CLUTCHES_WON), 0)          AS CLUTCHES_WON,
        COALESCE(SUM(ms.CLUTCHES_COUNT), 0)        AS CLUTCHES_COUNT,
        COALESCE(SUM(ms.UTILITY_DAMAGE), 0)        AS UTILITY_DAMAGE,
        COALESCE(SUM(ms.FLASH_ASSISTS), 0)         AS FLASH_ASSISTS,
        COALESCE(SUM(rp_stats.ROUNDS), 0)          AS ROUNDS_PLAYED,
        COALESCE(SUM(rp_stats.KAST), 0)            AS KAST_ROUNDS,
        SUM(CASE
            WHEN ms.TEAMNAME = m.TEAM_A_NAME AND m.TEAM_A_FINAL_SCORE > m.TEAM_B_FINAL_SCORE THEN 1
            WHEN ms.TEAMNAME = m.TEAM_B_NAME AND m.TEAM_B_FINAL_SCORE > m.TEAM_A_FINAL_SCORE THEN 1
            ELSE 0
        END) AS WINS,
        SUM(CASE
            WHEN ms.TEAMNAME = m.TEAM_A_NAME AND m.TEAM_A_FINAL_SCORE < m.TEAM_B_FINAL_SCORE THEN 1
            WHEN ms.TEAMNAME = m.TEAM_B_NAME AND m.TEAM_B_FINAL_SCORE < m.TEAM_A_FINAL_SCORE THEN 1
            ELSE 0
        END) AS LOSSES
    FROM PLAYERS p
    JOIN MATCH_STATS ms ON p.PLAYERID = ms.PLAYERID
    JOIN MATCHES m ON ms.MATCHID = m.MATCHID
    LEFT JOIN (
        SELECT
        rp.PLAYERID,
        rp.MATCHID,
        COUNT(DISTINCT rp.ROUND_NO) AS ROUNDS,
        SUM(CASE
            WHEN rp.GOT_KILL = 1 OR rp.GOT_ASSIST = 1
              OR rp.SURVIVED = 1  OR rp.GOT_TRADED = 1
            THEN 1 ELSE 0
        END) AS KAST
    FROM ROUND_PARTICIPANTS rp
    GROUP BY rp.PLAYERID, rp.MATCHID
    ) rp_stats ON rp_stats.PLAYERID = p.PLAYERID AND rp_stats.MATCHID = ms.MATCHID
	 WHERE ms.TEAMNAME = ?
	 GROUP BY p.PLAYERID, p.PLAYERNAME
	 ORDER BY KILLS DESC
	 `

	max_wins := 0
	max_loss := 0

	team_players, err := database.DB.Query(playerQuery, data.TeamName.String)
	if err != nil {
		log.Printf("Team player query error: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to load team data"})
	}
	roster := []fiber.Map{}
	defer team_players.Close()
	for team_players.Next() {
		var playerid int64
		var name, role string
		var kills, deaths, assists, app int
		var totalDamage, headshots int
		var entryKills, entryDeaths int
		var onek, twok, threek, fourk, ace int
		var clutchesWon, clutchesCount int
		var utilDamage, flashAssists int
		var roundsPlayed, kastRounds int
		var wins, loss int

		if err := team_players.Scan(
			&playerid, &name, &role,
			&kills, &deaths, &assists, &app,
			&totalDamage, &headshots,
			&entryKills, &entryDeaths,
			&onek, &twok, &threek, &fourk, &ace,
			&clutchesWon, &clutchesCount,
			&utilDamage, &flashAssists,
			&roundsPlayed, &kastRounds,
			&wins, &loss,
		); err != nil {
			log.Printf("Scan error: %v", err)
			continue
		}

		var adr, hsPct, kd, entryPct, clutchPct, kast, rating float64

		if kills > 0 {
			hsPct = float64(headshots) / float64(kills) * 100
		}
		if deaths > 0 {
			kd = float64(kills) / float64(deaths)
		}
		openingDuels := entryKills + entryDeaths
		if openingDuels > 0 {
			entryPct = float64(entryKills) / float64(openingDuels) * 100
		}
		if clutchesCount > 0 {
			clutchPct = float64(clutchesWon) / float64(clutchesCount) * 100
		}

		if wins > max_wins {
			max_wins = wins
		}
		if loss > max_loss {
			max_loss = loss
		}
		kpr := model.SafeDiv(kills, roundsPlayed) / 100
		dpr := model.SafeDiv(deaths, roundsPlayed) / 100
		apr := model.SafeDiv(assists, roundsPlayed) / 100
		adr = model.SafeDiv(totalDamage, roundsPlayed) / 100
		kast = model.SafeDiv(kastRounds, roundsPlayed) / 100

		impact := 2.13*kpr + 0.42*apr - 0.41
		rating = 0.0073*kast*100 + 0.3591*kpr - 0.5329*dpr + 0.2372*impact + 0.0032*adr + 0.1587

		roster = append(roster, fiber.Map{
			"id":         playerid,
			"role":       role,
			"name":       name,
			"kills":      kills,
			"assists":    assists,
			"deaths":     deaths,
			"matches":    wins + loss,
			"wins":       wins,
			"losses":     loss,
			"adr":        math.Round(adr*10) / 10,
			"dmg":        totalDamage,
			"hs":         math.Round(hsPct*10) / 10,
			"kd":         math.Round(kd*100) / 100,
			"kast":       math.Round(kast*1000) / 10,
			"rating":     math.Round(rating*100) / 100,
			"clutchWon":  clutchesWon,
			"onek":       onek,
			"twok":       twok,
			"threek":     threek,
			"fourk":      fourk,
			"ace":        ace,
			"clutchPct":  math.Round(clutchPct*10) / 10,
			"entryKills": entryKills,
			"openingPct": math.Round(entryPct*10) / 10,
			"utilDmg":    utilDamage,
			"flashAsts":  flashAssists,
			"current":    true,
		})
	}

	ts, err := auth.GetTeamRoundStats(database.DB, data.TeamName.String)
	if err != nil {
		log.Printf("Team stats query error: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to load team stats"})
	}

	TeamResponse := fiber.Map{
		"players": roster,
		"wins":    max_wins,
		"loss":    max_loss,
		"teamStats": fiber.Map{
			"totalRounds":  ts.TotalRounds,
			"roundsWon":    ts.RoundsWon,
			"roundsLost":   ts.RoundsLost,
			"tRounds":      ts.TRounds,
			"ctRounds":     ts.CTRounds,
			"tWins":        ts.TWins,
			"ctWins":       ts.CTWins,
			"pistolRounds": ts.PistolRounds,
			"pistolWins":   ts.PistolWins,
			"pistolPct":    math.Round(model.SafeDiv(ts.PistolWins, ts.PistolRounds)),
			"ecoRounds":    ts.EcoRounds,
			"ecoWins":      ts.EcoWins,
			"ecoPct":       math.Round(model.SafeDiv(ts.EcoWins, ts.EcoRounds)),
			"forceRounds":  ts.ForceRounds,
			"forceWins":    ts.ForceWins,
			"forcePct":     math.Round(model.SafeDiv(ts.ForceWins, ts.ForceRounds)),
			"fullRounds":   ts.FullBuyRounds,
			"fullWins":     ts.FullBuyWins,
			"fullPct":      math.Round(model.SafeDiv(ts.FullBuyWins, ts.FullBuyRounds)),
		},
	}
	return c.JSON(TeamResponse)
}

func Advanced(c fiber.Ctx) error {
	accountID, ok := c.Locals("accountID").(int64)
	if !ok {
		return c.Status(401).SendString("Unauthorized")
	}
	data, err := auth.GetPlayerPageData(database.DB, accountID)
	if err != nil {
		return c.Status(500).SendString("Failed to load player data")
	}
	if !data.TeamName.Valid {
		return c.Status(400).JSON(fiber.Map{"error": "No team linked to this account"})
	}

	ts, err := auth.GetTeamRoundStats(database.DB, data.TeamName.String)
	if err != nil {
		log.Printf("Team stats query error: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to load team stats"})
	}

	mapQuery := `
			SELECT
				m.MAP,
				COUNT(*) AS TOTAL_ROUNDS,
				COALESCE(ROUND(
					SUM(CASE WHEN tr.SIDE = 2 AND r.WINNING_SIDE = 2 THEN 1 ELSE 0 END) * 100.0 /
					NULLIF(SUM(CASE WHEN tr.SIDE = 2 THEN 1 ELSE 0 END), 0)
				, 1), 0) AS T_WIN_PCT,
				COALESCE(ROUND(
					SUM(CASE WHEN tr.SIDE = 3 AND r.WINNING_SIDE = 3 THEN 1 ELSE 0 END) * 100.0 /
					NULLIF(SUM(CASE WHEN tr.SIDE = 3 THEN 1 ELSE 0 END), 0)
				, 1), 0) AS CT_WIN_PCT,
				COALESCE(ROUND(
					SUM(CASE WHEN r.WINNING_SIDE = tr.SIDE AND
						((tr.SIDE = 2 AND r.BUY_TYPE_T = 1) OR (tr.SIDE = 3 AND r.BUY_TYPE_CT = 1))
						THEN 1 ELSE 0 END) * 100.0 /
					NULLIF(SUM(CASE WHEN
						(tr.SIDE = 2 AND r.BUY_TYPE_T = 1) OR (tr.SIDE = 3 AND r.BUY_TYPE_CT = 1)
						THEN 1 ELSE 0 END), 0)
				, 1), 0) AS PISTOL_WIN_PCT
			FROM (
				SELECT DISTINCT rp.MATCHID, rp.ROUND_NO, rp.SIDE
				FROM ROUND_PARTICIPANTS rp
				JOIN MATCH_STATS ms ON ms.PLAYERID = rp.PLAYERID AND ms.MATCHID = rp.MATCHID
				WHERE ms.TEAMNAME = ?
			) tr
			JOIN ROUNDS r ON r.MATCHID = tr.MATCHID AND r.ROUND_NO = tr.ROUND_NO
			JOIN MATCHES m ON m.MATCHID = tr.MATCHID
			GROUP BY m.MAP
			ORDER BY TOTAL_ROUNDS DESC
		`
	rows, err := database.DB.Query(mapQuery, data.TeamName.String)
	if err != nil {
		log.Printf("Map stats query error: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to load map stats"})
	}
	defer rows.Close()

	maps := []fiber.Map{}
	for rows.Next() {
		var mapName string
		var rounds int
		var tWinPct, ctWinPct, pistolWinPct float64
		if err := rows.Scan(&mapName, &rounds, &tWinPct, &ctWinPct, &pistolWinPct); err != nil {
			log.Printf("Map row scan error: %v", err)
			continue
		}
		maps = append(maps, fiber.Map{
			"map":          mapName,
			"rounds":       rounds,
			"tWinPct":      tWinPct,
			"ctWinPct":     ctWinPct,
			"pistolWinPct": pistolWinPct,
		})
	}

	return c.JSON(fiber.Map{
		"economy": fiber.Map{
			"pistolPct": math.Round(model.SafeDiv(ts.PistolWins, ts.PistolRounds)),
			"tSidePct":  math.Round(model.SafeDiv(ts.TWins, ts.TRounds)),
			"ctSidePct": math.Round(model.SafeDiv(ts.CTWins, ts.CTRounds)),
			"ecoPct":    math.Round(model.SafeDiv(ts.EcoWins, ts.EcoRounds)),
			"forcePct":  math.Round(model.SafeDiv(ts.ForceWins, ts.ForceRounds)),
			"fullPct":   math.Round(model.SafeDiv(ts.FullBuyWins, ts.FullBuyRounds)),
		},
		"maps": maps,
	})
}

func PlayerStats(c fiber.Ctx) error {
	accountID, ok := c.Locals("accountID").(int64)
	if !ok {
		return c.Status(401).SendString("Unauthorized")
	}
	data, err := auth.GetPlayerPageData(database.DB, accountID)
	if err != nil {
		return c.Status(500).SendString("Failed to load player data")
	}
	if !data.TeamName.Valid {
		return c.Status(400).JSON(fiber.Map{"error": "No team linked to this account"})
	}

	query := `
			SELECT
				p.PLAYERID,
				p.PLAYERNAME,
				COALESCE(p.ROLE, '')                       AS ROLE,
				COALESCE(SUM(ms.TOTAL_KILLS), 0)           AS KILLS,
				COALESCE(SUM(ms.TOTAL_DEATHS), 0)          AS DEATHS,
				COALESCE(SUM(ms.TOTAL_ASSISTS), 0)         AS ASSISTS,
				COALESCE(SUM(ms.TOTAL_DAMAGE), 0)          AS TOTAL_DAMAGE,
				COALESCE(SUM(ms.HEADSHOTS), 0)             AS HEADSHOTS,
				COALESCE(SUM(ms.ENTRY_KILLS), 0)           AS ENTRY_KILLS,
				COALESCE(SUM(ms.ENTRY_DEATHS), 0)          AS ENTRY_DEATHS,
				COALESCE(SUM(ms.CLUTCHES_WON), 0)          AS CLUTCHES_WON,
				COALESCE(SUM(ms.CLUTCHES_COUNT), 0)        AS CLUTCHES_COUNT,
				COALESCE(SUM(ms.UTILITY_DAMAGE), 0)        AS UTILITY_DAMAGE,
				COALESCE(SUM(ms.TRADED_DEATHS), 0)         AS TRADED_DEATHS,
				COALESCE(SUM(rp_stats.ROUNDS), 0)          AS ROUNDS_PLAYED,
				COALESCE(SUM(rp_stats.KAST), 0)            AS KAST_ROUNDS,
				SUM(CASE
					WHEN ms.TEAMNAME = m.TEAM_A_NAME AND m.TEAM_A_FINAL_SCORE > m.TEAM_B_FINAL_SCORE THEN 1
					WHEN ms.TEAMNAME = m.TEAM_B_NAME AND m.TEAM_B_FINAL_SCORE > m.TEAM_A_FINAL_SCORE THEN 1
					ELSE 0
				END) AS WINS,
				SUM(CASE
					WHEN ms.TEAMNAME = m.TEAM_A_NAME AND m.TEAM_A_FINAL_SCORE < m.TEAM_B_FINAL_SCORE THEN 1
					WHEN ms.TEAMNAME = m.TEAM_B_NAME AND m.TEAM_B_FINAL_SCORE < m.TEAM_A_FINAL_SCORE THEN 1
					ELSE 0
				END) AS LOSSES
			FROM PLAYERS p
			JOIN MATCH_STATS ms ON p.PLAYERID = ms.PLAYERID
			JOIN MATCHES m ON ms.MATCHID = m.MATCHID
			LEFT JOIN (
				SELECT
				rp.PLAYERID,
				rp.MATCHID,
				COUNT(DISTINCT rp.ROUND_NO) AS ROUNDS,
				SUM(CASE
					WHEN rp.GOT_KILL = 1 OR rp.GOT_ASSIST = 1
					  OR rp.SURVIVED = 1  OR rp.GOT_TRADED = 1
					THEN 1 ELSE 0
				END) AS KAST
			FROM ROUND_PARTICIPANTS rp
			GROUP BY rp.PLAYERID, rp.MATCHID
			) rp_stats ON rp_stats.PLAYERID = p.PLAYERID AND rp_stats.MATCHID = ms.MATCHID
			WHERE ms.TEAMNAME = ?
			GROUP BY p.PLAYERID, p.PLAYERNAME, p.ROLE
			ORDER BY KILLS DESC
		`
	rows, err := database.DB.Query(query, data.TeamName.String)
	if err != nil {
		log.Printf("Player stats query error: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to load player stats"})
	}
	defer rows.Close()

	players := []fiber.Map{}
	for rows.Next() {
		var playerid int64
		var name, role string
		var kills, deaths, assists int
		var totalDamage, headshots int
		var entryKills, entryDeaths int
		var clutchesWon, clutchesCount int
		var utilDamage, tradedDeaths int
		var roundsPlayed, kastRounds int
		var wins, loss int

		if err := rows.Scan(
			&playerid, &name, &role,
			&kills, &deaths, &assists,
			&totalDamage, &headshots,
			&entryKills, &entryDeaths,
			&clutchesWon, &clutchesCount,
			&utilDamage, &tradedDeaths,
			&roundsPlayed, &kastRounds,
			&wins, &loss,
		); err != nil {
			log.Printf("Scan error: %v", err)
			continue
		}

		var adr, hsPct, kd, entryPct, clutchPct, kast, rating, tradePct, utilPerRound float64

		if kills > 0 {
			hsPct = float64(headshots) / float64(kills) * 100
		}
		if deaths > 0 {
			kd = float64(kills) / float64(deaths)
			tradePct = float64(tradedDeaths) / float64(deaths) * 100
		}
		openingDuels := entryKills + entryDeaths
		if openingDuels > 0 {
			entryPct = float64(entryKills) / float64(openingDuels) * 100
		}
		if clutchesCount > 0 {
			clutchPct = float64(clutchesWon) / float64(clutchesCount) * 100
		}

		kpr := model.SafeDiv(kills, roundsPlayed) / 100
		dpr := model.SafeDiv(deaths, roundsPlayed) / 100
		apr := model.SafeDiv(assists, roundsPlayed) / 100
		adr = model.SafeDiv(totalDamage, roundsPlayed) / 100
		kast = model.SafeDiv(kastRounds, roundsPlayed) / 100
		utilPerRound = model.SafeDiv(utilDamage, roundsPlayed) / 100

		impact := 2.13*kpr + 0.42*apr - 0.41
		rating = 0.0073*kast*100 + 0.3591*kpr - 0.5329*dpr + 0.2372*impact + 0.0032*adr + 0.1587

		players = append(players, fiber.Map{
			"id":           strconv.FormatInt(playerid, 10),
			"name":         name,
			"role":         role,
			"matches":      wins + loss,
			"wins":         wins,
			"losses":       loss,
			"kills":        kills,
			"deaths":       deaths,
			"assists":      assists,
			"adr":          math.Round(adr*10) / 10,
			"hs":           math.Round(hsPct*10) / 10,
			"kd":           math.Round(kd*100) / 100,
			"kast":         math.Round(kast*1000) / 10,
			"rating":       math.Round(rating*100) / 100,
			"clutchWon":    clutchesWon,
			"clutchPct":    math.Round(clutchPct*10) / 10,
			"entryKills":   entryKills,
			"openingPct":   math.Round(entryPct*10) / 10,
			"tradePct":     math.Round(tradePct*10) / 10,
			"utilDmgPerRd": math.Round(utilPerRound*10) / 10,
			"roundsPlayed": roundsPlayed,
			"current":      true,
		})
	}

	return c.JSON(players)
}

func Info(c fiber.Ctx) error {
	accountID, ok := c.Locals("accountID").(int64)
	if !ok {
		return c.Status(401).SendString("Unauthorized")
	}
	data, err := auth.GetPlayerPageData(database.DB, accountID)
	if err != nil || !data.TeamName.Valid {
		return c.Status(400).JSON(fiber.Map{"error": "No team linked"})
	}
	team := data.TeamName.String

	var playerCount, matchCount, seasonCount int
	database.DB.QueryRow(`SELECT COUNT(*) FROM TEAM_MEMBERS WHERE TEAMNAME = ?`, team).Scan(&playerCount)
	database.DB.QueryRow(`SELECT COUNT(*) FROM MATCHES WHERE TEAM_A_NAME = ? OR TEAM_B_NAME = ?`, team, team).Scan(&matchCount)
	database.DB.QueryRow(`SELECT COUNT(*) FROM SEASONS WHERE TEAMNAME = ?`, team).Scan(&seasonCount)

	return c.JSON(fiber.Map{
		"teamName":    team,
		"playerCount": playerCount,
		"matchCount":  matchCount,
		"seasonCount": seasonCount,
	})
}

func Logo(c fiber.Ctx) error {
	accountID, ok := c.Locals("accountID").(int64)
	if !ok {
		return c.Status(401).SendString("Unauthorized")
	}
	data, err := auth.GetPlayerPageData(database.DB, accountID)
	if err != nil || !data.TeamName.Valid {
		return c.Status(400).JSON(fiber.Map{"error": "No team linked"})
	}
	file, err := c.FormFile("logo")
	if err != nil {
		return c.Status(400).SendString("No file provided")
	}
	os.MkdirAll("./static/logos", 0755)
	ext := path.Ext(file.Filename)
	safe := strings.ReplaceAll(data.TeamName.String, " ", "_")
	filename := safe + ext
	if err := c.SaveFile(file, "./static/logos/"+filename); err != nil {
		return c.Status(500).SendString("Failed to save logo")
	}
	logoURL := "/static/logos/" + filename
	database.DB.Exec("UPDATE TEAMS SET TEAM_LOGO_URL = ? WHERE TEAMNAME = ?", logoURL, data.TeamName.String)
	return c.JSON(fiber.Map{"url": logoURL})
}

func UploadDemo(c fiber.Ctx) error {
	_, ok := c.Locals("accountID").(int64)
	if !ok {
		return c.Status(401).SendString("Unauthorized")
	}
	file, err := c.FormFile("demo")
	if err != nil {
		return c.Status(400).SendString("No file provided")
	}
	season := c.FormValue("season")
	matchLabel := c.FormValue("match_label")
	savePath := auth.GetDemoPath() + file.Filename
	if err := c.SaveFile(file, savePath); err != nil {
		return c.Status(500).SendString("Failed to save demo")
	}
	database.DB.Exec("UPDATE MATCHES SET SEASON = ?, NOTES = ? WHERE DEMO_NAME = ?",
		season, matchLabel, file.Filename)
	return c.JSON(fiber.Map{"message": "Demo uploaded", "filename": file.Filename})
}

func Demos(c fiber.Ctx) error {
	accountID, ok := c.Locals("accountID").(int64)
	if !ok {
		return c.Status(401).SendString("Unauthorized")
	}
	data, err := auth.GetPlayerPageData(database.DB, accountID)
	if err != nil || !data.TeamName.Valid {
		return c.Status(400).JSON(fiber.Map{"error": "No team linked"})
	}
	type DemoRow struct {
		DemoName string `json:"demoName"`
		Map      string `json:"map"`
		Date     string `json:"date"`
		Season   string `json:"season"`
		Notes    string `json:"notes"`
	}
	rows, err := database.DB.Query(`
			SELECT DEMO_NAME, MAP,
			       COALESCE(SAVED_DATE, ''),
			       COALESCE(SEASON, ''),
			       COALESCE(NOTES, '')
			FROM MATCHES
			WHERE TEAM_A_NAME = ? OR TEAM_B_NAME = ?
			ORDER BY MATCHID DESC
		`, data.TeamName.String, data.TeamName.String)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to query demos"})
	}
	defer rows.Close()
	demos := []DemoRow{}
	for rows.Next() {
		var d DemoRow
		if err := rows.Scan(&d.DemoName, &d.Map, &d.Date, &d.Season, &d.Notes); err != nil {
			continue
		}
		demos = append(demos, d)
	}
	return c.JSON(demos)
}
