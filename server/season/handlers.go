package season

import (
	"fmt"
	database "server/DB"
	"server/auth"
	"strconv"

	"github.com/gofiber/fiber/v3"
)

func Matches(c fiber.Ctx) error {
	accountID, ok := c.Locals("accountID").(int64)
	if !ok {
		return c.Status(401).SendString("Unauthorized")
	}
	data, err := auth.GetPlayerPageData(database.DB, accountID)
	if err != nil || !data.TeamName.Valid {
		return c.Status(400).JSON(fiber.Map{"error": "No team linked"})
	}
	seasonId, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "invalid season id"})
	}
	query := `
	SELECT
		m.MATCHID, m.MAP, COALESCE(m.TEAM_A_NAME, ''), m.TEAM_A_FINAL_SCORE,
		COALESCE(m.TEAM_B_NAME, ''), m.TEAM_B_FINAL_SCORE, COALESCE(m.MATCH_DATE, ''),
		s.NAME AS SEASON_NAME
	FROM MATCHES m
	JOIN SEASONS s ON s.SEASON_ID = m.SEASON_ID
	WHERE m.SEASON_ID = ?
	ORDER BY m.MATCH_DATE
`

	type SeasonMatch struct {
		ID         int    `json:"matchId"`
		Map        string `json:"map"`
		TeamA      string `json:"teamA"`
		TeamAScore int    `json:"scoreA"`
		TeamB      string `json:"teamB"`
		TeamBScore int    `json:"scoreB"`
		Date       string `json:"date"`
		SeasonName string
	}
	fmt.Printf("Season ID : %v", seasonId)
	rows, err := database.DB.Query(query, seasonId)
	if err != nil {
		fmt.Printf("Error %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "invalid season id"})
	}
	var matches []SeasonMatch
	var seasonName string
	for rows.Next() {
		var m SeasonMatch
		if err := rows.Scan(&m.ID, &m.Map, &m.TeamA, &m.TeamAScore,
			&m.TeamB, &m.TeamBScore, &m.Date, &m.SeasonName); err != nil {
			continue
		}
		seasonName = m.SeasonName
		matches = append(matches, m)
	}

	return c.JSON(fiber.Map{
		"seasonName": seasonName,
		"matches":    matches,
	})
}
