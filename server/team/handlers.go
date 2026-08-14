package team

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"os"
	"path"
	database "server/DB"
	"server/auth"
	"server/model"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	dem "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/msg"
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
		COALESCE(p.AVATAR_URL, ''), 
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
		var name, role, avatar string
		var kills, deaths, assists, app int
		var totalDamage, headshots int
		var entryKills, entryDeaths int
		var onek, twok, threek, fourk, ace int
		var clutchesWon, clutchesCount int
		var utilDamage, flashAssists int
		var roundsPlayed, kastRounds int
		var wins, loss int

		if err := team_players.Scan(
			&playerid, &name, &avatar, &role,
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
			"avatar":     avatar,
		})
	}

	ts, err := auth.GetTeamRoundStats(database.DB, data.TeamName.String)
	if err != nil {
		log.Printf("Team stats query error: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to load team stats"})
	}
	plantWinPct := 0.0
	if ts.TPlantWinPct.Valid {
		plantWinPct = ts.TPlantWinPct.Float64
	}
	retakeWinPct := 0.0
	if ts.RetakeSuccessPct.Valid {
		retakeWinPct = ts.RetakeSuccessPct.Float64
	}

	TeamResponse := fiber.Map{
		"players": roster,
		"wins":    max_wins,
		"loss":    max_loss,
		"teamStats": fiber.Map{
			"totalRounds":         ts.TotalRounds,
			"roundsWon":           ts.RoundsWon,
			"roundsLost":          ts.RoundsLost,
			"tRounds":             ts.TRounds,
			"ctRounds":            ts.CTRounds,
			"tWins":               ts.TWins,
			"ctWins":              ts.CTWins,
			"pistolRounds":        ts.PistolRounds,
			"pistolWins":          ts.PistolWins,
			"pistolPct":           math.Round(model.SafeDiv(ts.PistolWins, ts.PistolRounds)),
			"ecoRounds":           ts.EcoRounds,
			"ecoWins":             ts.EcoWins,
			"ecoPct":              math.Round(model.SafeDiv(ts.EcoWins, ts.EcoRounds)),
			"forceRounds":         ts.ForceRounds,
			"forceWins":           ts.ForceWins,
			"forcePct":            math.Round(model.SafeDiv(ts.ForceWins, ts.ForceRounds)),
			"fullRounds":          ts.FullBuyRounds,
			"fullWins":            ts.FullBuyWins,
			"fullPct":             math.Round(model.SafeDiv(ts.FullBuyWins, ts.FullBuyRounds)),
			"gunRounds":           ts.GunRounds,
			"gunWins":             ts.GunWins,
			"gunPct":              math.Round(model.SafeDiv(ts.GunWins, ts.GunRounds)),
			"antiEcoRounds":       ts.AntiEcoRounds,
			"antiEcoWins":         ts.AntiEcoWins,
			"antiEcoPct":          math.Round(model.SafeDiv(ts.AntiEcoWins, ts.AntiEcoRounds)),
			"tPlants":             ts.TPlants,
			"tPlantWins":          ts.TPlantWins,
			"tPlantWinPct":        plantWinPct,
			"retakeOpportunities": ts.RetakeOpportunities,
			"retakeWins":          ts.RetakeWins,
			"retakeWinPct":        retakeWinPct,
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

	plantWinPct := 0.0
	if ts.TPlantWinPct.Valid {
		plantWinPct = ts.TPlantWinPct.Float64
	}
	retakeWinPct := 0.0
	if ts.RetakeSuccessPct.Valid {
		retakeWinPct = ts.RetakeSuccessPct.Float64
	}

	return c.JSON(fiber.Map{
		"economy": fiber.Map{
			"pistolPct":  math.Round(model.SafeDiv(ts.PistolWins, ts.PistolRounds)),
			"tSidePct":   math.Round(model.SafeDiv(ts.TWins, ts.TRounds)),
			"ctSidePct":  math.Round(model.SafeDiv(ts.CTWins, ts.CTRounds)),
			"ecoPct":     math.Round(model.SafeDiv(ts.EcoWins, ts.EcoRounds)),
			"forcePct":   math.Round(model.SafeDiv(ts.ForceWins, ts.ForceRounds)),
			"fullPct":    math.Round(model.SafeDiv(ts.FullBuyWins, ts.FullBuyRounds)),
			"gunPct":     math.Round(model.SafeDiv(ts.GunWins, ts.GunRounds)),
			"antiEcoPct": math.Round(model.SafeDiv(ts.AntiEcoWins, ts.AntiEcoRounds)),
		},
		"bombRetakes": fiber.Map{
			"plants":              ts.TPlants,
			"plantWins":           ts.TPlantWins,
			"plantWinPct":         plantWinPct,
			"retakeOpportunities": ts.RetakeOpportunities,
			"retakeWins":          ts.RetakeWins,
			"retakeWinPct":        retakeWinPct,
		},
		"winConditions": fiber.Map{
			"eliminationPct": math.Round(model.SafeDiv(ts.EliminationWins, ts.RoundsWon)),
			"bombPct":        math.Round(model.SafeDiv(ts.BombWins, ts.RoundsWon)),
			"timePct":        math.Round(model.SafeDiv(ts.TimeWins, ts.RoundsWon)),
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

// quickMapName does a minimal parse of a demo file just far enough to read
// the map name off the ServerInfo net message, then cancels — mirrors the
// lazy-scan logic in main.go's /AllDemos handler.
func quickMapName(path string) string {
	demofile, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer demofile.Close()
	p := dem.NewParser(demofile)
	defer p.Close()
	var gameMap string
	p.RegisterNetMessageHandler(func(m *msg.CSVCMsg_ServerInfo) {
		gameMap = *m.MapName
		p.Cancel()
	})
	p.ParseToEnd()
	return gameMap
}

func UploadDemo(c fiber.Ctx) error {
	accountID, ok := c.Locals("accountID").(int64)
	if !ok {
		return c.Status(401).SendString("Unauthorized")
	}
	data, err := auth.GetPlayerPageData(database.DB, accountID)
	if err != nil || !data.TeamName.Valid {
		return c.Status(400).JSON(fiber.Map{"error": "No team linked"})
	}

	seasonID, err := strconv.ParseInt(c.FormValue("season_id"), 10, 64)
	if err != nil {
		return c.Status(400).SendString("A season must be selected")
	}
	var seasonName string
	if err := database.DB.QueryRow(
		"SELECT NAME FROM SEASONS WHERE SEASON_ID = ? AND TEAMNAME = ?",
		seasonID, data.TeamName.String,
	).Scan(&seasonName); err != nil {
		return c.Status(400).SendString("Invalid season")
	}

	file, err := c.FormFile("demo")
	if err != nil {
		return c.Status(400).SendString("No file provided")
	}
	matchLabel := c.FormValue("match_label")
	savePath := auth.GetDemoPath() + file.Filename
	if err := c.SaveFile(file, savePath); err != nil {
		return c.Status(500).SendString("Failed to save demo")
	}

	res, updateErr := database.DB.Exec(
		"UPDATE MATCHES SET SEASON_ID = ?, SEASON = ?, NOTES = ? WHERE DEMO_NAME = ?",
		seasonID, seasonName, matchLabel, file.Filename,
	)
	var rowsAffected int64
	if updateErr == nil {
		rowsAffected, _ = res.RowsAffected()
	}
	if updateErr != nil || rowsAffected == 0 {
		// No MATCHES row exists yet for this demo (it's normally created
		// lazily by the /AllDemos scan) — create it now so the season
		// assignment isn't silently lost.
		gameMap := quickMapName(savePath)
		savedDate := time.Now().Local().Format("2006-1-2")
		if _, err := database.DB.Exec(
			`INSERT INTO MATCHES (DEMO_NAME, MAP, SAVED_DATE, PARSED_STATS, PARSED_2D, SEASON_ID, SEASON, NOTES)
			 VALUES (?, ?, ?, 0, 0, ?, ?, ?)`,
			file.Filename, gameMap, savedDate, seasonID, seasonName, matchLabel,
		); err != nil {
			log.Printf("UploadDemo insert error: %v", err)
			return c.Status(500).SendString("Demo saved but failed to register match")
		}
	}
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

func Roster(c fiber.Ctx) error {
	accountID, ok := c.Locals("accountID").(int64)
	if !ok {
		return c.Status(401).SendString("Unauthorized")
	}
	data, err := auth.GetPlayerPageData(database.DB, accountID)
	if err != nil || !data.TeamName.Valid {
		return c.Status(400).JSON(fiber.Map{"error": "No team linked"})
	}
	query := `
		SELECT 
			tm.PLAYERID,
			p.PLAYERNAME,
			tm.IS_ACTIVE,
			tm.ROSTER_ORDER,
			tm.JOINED_AT,
			COALESCE(p.AVATAR_URL, '')
		FROM TEAM_MEMBERS tm
		JOIN PLAYERS p ON p.PLAYERID = tm.PLAYERID
		WHERE tm.TEAMNAME = ?
		ORDER BY tm.ROSTER_ORDER
		`
	rows, err := database.DB.Query(query, data.TeamName.String)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to query roster"})
	}
	type RosterMember struct {
		PlayerId   string `json:"id"`
		Playername string `json:"name"`
		IsActive   bool   `json:"isActive"`
		Order      int    `json:"rosterOrder"`
		JoinedAt   string `json:"joined_at,string"`
		Avatar     string `json:"avatarUrl"`
	}
	var roster []RosterMember
	for rows.Next() {
		var p RosterMember
		if err := rows.Scan(&p.PlayerId, &p.Playername, &p.IsActive, &p.Order, &p.JoinedAt, &p.Avatar); err != nil {
			continue
		}
		roster = append(roster, p)
	}
	defer rows.Close()
	return c.JSON(roster)
}
func Seasons(c fiber.Ctx) error {
	accountID, ok := c.Locals("accountID").(int64)
	if !ok {
		return c.Status(401).SendString("Unauthorized")
	}
	data, err := auth.GetPlayerPageData(database.DB, accountID)
	if err != nil || !data.TeamName.Valid {
		return c.Status(400).JSON(fiber.Map{"error": "No team linked"})
	}
	query := `
		SELECT SEASON_ID, NAME,
		COALESCE(START_DATE, ''), COALESCE(END_DATE, ''),
		COALESCE(NOTES, '')
		FROM SEASONS
		WHERE TEAMNAME = ?
	`
	rows, err := database.DB.Query(query, data.TeamName.String)
	fmt.Printf("Team Nam: %v, %v", data.TeamName.String, data)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to query seasons"})
	}
	type SeasonsResp struct {
		SeasonId   int    `json:"id"`
		SeasonName string `json:"name"`
		StartDate  string `json:"startDate"`
		EndDate    string `json:"endDate"`
		Notes      string `json:"notes"`
	}
	var seasons []SeasonsResp
	for rows.Next() {
		var s SeasonsResp
		if err := rows.Scan(&s.SeasonId, &s.SeasonName, &s.StartDate, &s.EndDate, &s.Notes); err != nil {
			continue
		} else {
			fmt.Printf("%v", err)
		}
		seasons = append(seasons, s)
	}

	return c.Status(200).JSON(seasons)
}

type seasonRequest struct {
	Name      string `json:"name"`
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
	Notes     string `json:"notes"`
}

func CreateSeason(c fiber.Ctx) error {
	accountID, ok := c.Locals("accountID").(int64)
	if !ok {
		return c.Status(401).SendString("Unauthorized")
	}
	data, err := auth.GetPlayerPageData(database.DB, accountID)
	if err != nil || !data.TeamName.Valid {
		return c.Status(400).JSON(fiber.Map{"error": "No team linked"})
	}
	var req seasonRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(400).SendString("Invalid request body")
	}
	if strings.TrimSpace(req.Name) == "" {
		return c.Status(400).SendString("Season name is required")
	}

	res, err := database.DB.Exec(
		`INSERT INTO SEASONS (TEAMNAME, NAME, START_DATE, END_DATE, NOTES)
		 VALUES (?, ?, NULLIF(?, ''), NULLIF(?, ''), ?)`,
		data.TeamName.String, req.Name, req.StartDate, req.EndDate, req.Notes,
	)
	if err != nil {
		log.Printf("CreateSeason insert error: %v", err)
		return c.Status(500).SendString("Failed to create season")
	}
	id, _ := res.LastInsertId()
	return c.Status(200).JSON(fiber.Map{"id": id, "message": "Season created"})
}

func EditSeason(c fiber.Ctx) error {
	accountID, ok := c.Locals("accountID").(int64)
	if !ok {
		return c.Status(401).SendString("Unauthorized")
	}
	data, err := auth.GetPlayerPageData(database.DB, accountID)
	if err != nil || !data.TeamName.Valid {
		return c.Status(400).JSON(fiber.Map{"error": "No team linked"})
	}
	seasonID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid season id"})
	}
	var req seasonRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(400).SendString("Invalid request body")
	}
	if strings.TrimSpace(req.Name) == "" {
		return c.Status(400).SendString("Season name is required")
	}

	res, err := database.DB.Exec(
		`UPDATE SEASONS SET NAME = ?, START_DATE = NULLIF(?, ''), END_DATE = NULLIF(?, ''), NOTES = ?
		 WHERE SEASON_ID = ? AND TEAMNAME = ?`,
		req.Name, req.StartDate, req.EndDate, req.Notes, seasonID, data.TeamName.String,
	)
	if err != nil {
		log.Printf("EditSeason update error: %v", err)
		return c.Status(500).SendString("Failed to update season")
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return c.Status(404).SendString("Season not found")
	}
	return c.JSON(fiber.Map{"message": "Season updated"})
}

type unassignedPlayer struct {
	PlayerID   string `json:"playerId"`
	PlayerName string `json:"playerName"`
}

type unassignedGroup struct {
	MatchID     int                `json:"matchId"`
	SideIndex   int                `json:"sideIndex"`
	DefaultName string             `json:"defaultName"`
	DemoName    string             `json:"demoName"`
	Map         string             `json:"map"`
	Players     []unassignedPlayer `json:"players"`
}

// UnassignedPlayers lists demo sides that had no clan tag set and no
// existing roster to fall back to (see resolveMatchTeam in the parser
// package) — a data-quality queue any logged-in user can help triage,
// not scoped to a single team since the players in it belong to none.
func UnassignedPlayers(c fiber.Ctx) error {
	if _, ok := c.Locals("accountID").(int64); !ok {
		return c.Status(401).SendString("Unauthorized")
	}

	query := `
		SELECT u.MATCHID, u.SIDE_INDEX, u.DEFAULT_TEAM_NAME,
		       u.PLAYERID, p.PLAYERNAME,
		       m.DEMO_NAME, m.MAP
		FROM UNASSIGNED_MATCH_PLAYERS u
		JOIN PLAYERS p ON p.PLAYERID = u.PLAYERID
		JOIN MATCHES m ON m.MATCHID = u.MATCHID
	`
	args := []interface{}{}
	if matchIDParam := c.Query("matchId"); matchIDParam != "" {
		matchID, err := strconv.ParseInt(matchIDParam, 10, 64)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid matchId"})
		}
		query += " WHERE u.MATCHID = ?"
		args = append(args, matchID)
	}
	query += " ORDER BY u.MATCHID, u.SIDE_INDEX, p.PLAYERNAME"

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		log.Printf("UnassignedPlayers query error: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to load unassigned players"})
	}
	defer rows.Close()

	type groupKey struct {
		MatchID   int
		SideIndex int
	}
	groups := map[groupKey]*unassignedGroup{}
	var order []groupKey

	for rows.Next() {
		var matchID, sideIndex int
		var defaultName, demoName, mapName string
		var playerID int64
		var playerName string
		if err := rows.Scan(&matchID, &sideIndex, &defaultName, &playerID, &playerName, &demoName, &mapName); err != nil {
			continue
		}
		key := groupKey{matchID, sideIndex}
		g, ok := groups[key]
		if !ok {
			g = &unassignedGroup{
				MatchID: matchID, SideIndex: sideIndex,
				DefaultName: defaultName, DemoName: demoName, Map: mapName,
			}
			groups[key] = g
			order = append(order, key)
		}
		g.Players = append(g.Players, unassignedPlayer{
			PlayerID: strconv.FormatInt(playerID, 10), PlayerName: playerName,
		})
	}

	result := make([]*unassignedGroup, 0, len(order))
	for _, key := range order {
		result = append(result, groups[key])
	}
	return c.JSON(result)
}

// ResolveUnassigned assigns every player on one unresolved demo side to a
// real team for that match, then clears them from the unassigned queue.
// It intentionally does NOT touch TEAM_MEMBERS — that's a separate,
// deliberate roster-management action, not something a one-off match
// resolution should imply.
func ResolveUnassigned(c fiber.Ctx) error {
	if _, ok := c.Locals("accountID").(int64); !ok {
		return c.Status(401).SendString("Unauthorized")
	}
	var req struct {
		MatchID   int    `json:"matchId"`
		SideIndex int    `json:"sideIndex"`
		TeamName  string `json:"teamName"`
	}
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(400).SendString("Invalid request body")
	}
	if strings.TrimSpace(req.TeamName) == "" {
		return c.Status(400).SendString("Team name is required")
	}

	tx, err := database.DB.Begin()
	if err != nil {
		return c.Status(500).SendString("Failed to start transaction")
	}
	defer tx.Rollback()

	if _, err := tx.Exec("INSERT IGNORE INTO TEAMS (TEAMNAME) VALUES (?)", req.TeamName); err != nil {
		log.Printf("ResolveUnassigned team insert error: %v", err)
		return c.Status(500).SendString("Failed to register team")
	}
	if _, err := tx.Exec(
		`UPDATE MATCH_STATS SET TEAMNAME = ?
		 WHERE MATCHID = ? AND PLAYERID IN (
		     SELECT PLAYERID FROM UNASSIGNED_MATCH_PLAYERS WHERE MATCHID = ? AND SIDE_INDEX = ?
		 )`,
		req.TeamName, req.MatchID, req.MatchID, req.SideIndex,
	); err != nil {
		log.Printf("ResolveUnassigned update error: %v", err)
		return c.Status(500).SendString("Failed to assign team")
	}
	// MATCHES.TEAM_A_NAME/TEAM_B_NAME is a display label for the match
	// overview — only overwrite it if it's still showing the generic
	// placeholder, so a mixed side that already picked up a real name at
	// parse time isn't clobbered by resolving its remaining stragglers.
	nameColumn := "TEAM_A_NAME"
	if req.SideIndex == 1 {
		nameColumn = "TEAM_B_NAME"
	}
	if _, err := tx.Exec(
		"UPDATE MATCHES SET "+nameColumn+" = ? WHERE MATCHID = ? AND "+nameColumn+" IN ('Team 1', 'Team 2')",
		req.TeamName, req.MatchID,
	); err != nil {
		log.Printf("ResolveUnassigned match name update error: %v", err)
		return c.Status(500).SendString("Failed to update match display name")
	}
	if _, err := tx.Exec(
		`DELETE FROM UNASSIGNED_MATCH_PLAYERS WHERE MATCHID = ? AND SIDE_INDEX = ?`,
		req.MatchID, req.SideIndex,
	); err != nil {
		log.Printf("ResolveUnassigned cleanup error: %v", err)
		return c.Status(500).SendString("Failed to clear unassigned entry")
	}
	if err := tx.Commit(); err != nil {
		log.Printf("ResolveUnassigned commit error: %v", err)
		return c.Status(500).SendString("Failed to save changes")
	}
	return c.JSON(fiber.Map{"message": "Team assigned"})
}

// DismissUnassigned drops one unresolved demo side from the review queue
// without assigning it a team — e.g. an opponent side from a solo/premier
// match the user doesn't care to track. Their MATCH_STATS.TEAMNAME simply
// stays NULL, which already keeps them out of every team-scoped query.
func DismissUnassigned(c fiber.Ctx) error {
	if _, ok := c.Locals("accountID").(int64); !ok {
		return c.Status(401).SendString("Unauthorized")
	}
	var req struct {
		MatchID   int `json:"matchId"`
		SideIndex int `json:"sideIndex"`
	}
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(400).SendString("Invalid request body")
	}

	if _, err := database.DB.Exec(
		`DELETE FROM UNASSIGNED_MATCH_PLAYERS WHERE MATCHID = ? AND SIDE_INDEX = ?`,
		req.MatchID, req.SideIndex,
	); err != nil {
		log.Printf("DismissUnassigned error: %v", err)
		return c.Status(500).SendString("Failed to dismiss unassigned entry")
	}
	return c.JSON(fiber.Map{"message": "Dismissed"})
}

// RenameTeam changes TEAMS.TEAMNAME, which every other table references by
// name rather than a surrogate ID — this only works because those foreign
// keys are declared ON UPDATE CASCADE, so a single UPDATE here propagates
// to SEASONS, MATCHES.TEAM_A_NAME/TEAM_B_NAME, MATCH_STATS, TEAM_MEMBERS,
// TEAM_INVITES, and TEAM_MODERATORS automatically.
func RenameTeam(c fiber.Ctx) error {
	accountID, ok := c.Locals("accountID").(int64)
	if !ok {
		return c.Status(401).SendString("Unauthorized")
	}
	data, err := auth.GetPlayerPageData(database.DB, accountID)
	if err != nil || !data.TeamName.Valid {
		return c.Status(400).JSON(fiber.Map{"error": "No team linked"})
	}

	var req struct {
		NewName string `json:"newName"`
	}
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(400).SendString("Invalid request body")
	}
	newName := strings.TrimSpace(req.NewName)
	if newName == "" {
		return c.Status(400).SendString("Team name is required")
	}
	oldName := data.TeamName.String
	if newName == oldName {
		return c.JSON(fiber.Map{"message": "Team name unchanged", "teamName": newName})
	}

	var existing string
	err = database.DB.QueryRow("SELECT TEAMNAME FROM TEAMS WHERE TEAMNAME = ?", newName).Scan(&existing)
	if err == nil {
		return c.Status(409).SendString("That team name is already taken")
	}
	if err != sql.ErrNoRows {
		log.Printf("RenameTeam lookup error: %v", err)
		return c.Status(500).SendString("Something went wrong. Please try again.")
	}

	if _, err := database.DB.Exec("UPDATE TEAMS SET TEAMNAME = ? WHERE TEAMNAME = ?", newName, oldName); err != nil {
		log.Printf("RenameTeam update error: %v", err)
		return c.Status(500).SendString("Failed to rename team")
	}
	return c.JSON(fiber.Map{"message": "Team renamed", "teamName": newName})
}
