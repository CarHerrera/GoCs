package matches

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	database "server/DB"
	"server/model"
	"server/parser"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/golang/geo/r2"
	"github.com/golang/geo/r3"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
)

// Side values mirror common.Team so they line up with ROUND_PARTICIPANTS.SIDE / ROUNDS.WINNING_SIDE.
const (
	sideT  = int(common.TeamTerrorists)
	sideCT = int(common.TeamCounterTerrorists)
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
	}

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
		// Fall through to the SQL path below instead of returning
		// demo_stats.TeamStats directly — that shape omits KAST/rating and
		// doesn't match what the frontend expects, which is why a fresh
		// parse used to require a manual reload to show correct stats.
		row.Map = demo_stats.Map
	}

	log.Printf("Grabbing from SQL")
	query := `
		SELECT
			m.TEAM_A_NAME, m.TEAM_B_NAME,
			m.TEAM_A_T_SCORE, m.TEAM_A_CT_SCORE, m.TEAM_A_FINAL_SCORE,
			m.TEAM_B_T_SCORE, m.TEAM_B_CT_SCORE, m.TEAM_B_FINAL_SCORE,
			COALESCE(ms.TEAMNAME, "Unknown Team"), p.PLAYERNAME, p.PLAYERID,
			ms.TOTAL_KILLS, ms.HEADSHOTS, ms.ENTRY_KILLS, ms.ENTRY_DEATHS,
			ms.TOTAL_DEATHS, ms.TOTAL_ASSISTS,
			ms.TOTAL_DAMAGE, ms.UTILITY_DAMAGE,
			ms.ONE_KILL_COUNT, ms.TWO_KILL_COUNT, ms.THREE_KILL_COUNT,
			ms.FOUR_KILL_COUNT, ms.FIVE_KILL_COUNT,
			ms.TRADED_DEATHS, ms.TRADED_KILLS,
			ms.CLUTCHES_WON, ms.CLUTCHES_COUNT,
			ms.FLASH_ASSISTS, ms.TEAM_FLASHES, ms.ENEMIES_FLASHED,
			ms.HE_DAMAGE, ms.FIRE_DAMAGE, ms.SIDE_INDEX,
			(
				SELECT COUNT(*) FROM ROUND_PARTICIPANTS rp
				WHERE rp.MATCHID = m.MATCHID AND rp.PLAYERID = p.PLAYERID
				AND (rp.GOT_KILL = 1 OR rp.GOT_ASSIST = 1 OR rp.GOT_TRADED = 1 OR rp.SURVIVED = 1)
			) AS KAST_ROUNDS
		FROM MATCHES m
		JOIN MATCH_STATS ms ON m.MATCHID = ms.MATCHID
		JOIN PLAYERS p ON ms.PLAYERID = p.PLAYERID
		WHERE m.MATCHID = ?
		ORDER BY ms.SIDE_INDEX, ms.TOTAL_KILLS DESC`

	rows, err := database.DB.Query(query, row.ID)
	if err != nil {
		log.Printf("SQL ERROR (advanced stats query): %v", err)
		return c.Status(500).JSON(fiber.Map{
			"message": "Failed to query advanced stats",
			"success": "false",
		})
	}
	defer rows.Close()

	var stats [2]model.Team
	var teamAName, teamBName string
	var A_T, A_CT, A_FINAL, B_T, B_CT, B_FINAL int

	for rows.Next() {
		var teamName, playerName string
		var playerID int64
		var sideIndex sql.NullInt64
		var kills, hs, entryKills, entryDeaths, deaths, assists int
		var damage, utilityDamage int
		var k1, k2, k3, k4, k5 int
		var tradedDeaths, tradeKills int
		var clutchesWon, clutchCount int
		var flashAssists, teamFlashed, enemiesFlashed int
		var heDamage, fireDamage int
		var kastRounds int

		if err := rows.Scan(
			&teamAName, &teamBName,
			&A_T, &A_CT, &A_FINAL,
			&B_T, &B_CT, &B_FINAL,
			&teamName, &playerName, &playerID,
			&kills, &hs, &entryKills, &entryDeaths,
			&deaths, &assists,
			&damage, &utilityDamage,
			&k1, &k2, &k3, &k4, &k5,
			&tradedDeaths, &tradeKills,
			&clutchesWon, &clutchCount,
			&flashAssists, &teamFlashed, &enemiesFlashed,
			&heDamage, &fireDamage, &sideIndex,
			&kastRounds,
		); err != nil {
			log.Printf("Row scan error for match %v: %v", row.ID, err)
			return c.Status(500).JSON(fiber.Map{
				"message": "Failed to read advanced stats row",
				"success": "false",
			})
		}

		totalRounds := A_FINAL + B_FINAL
		// SIDE_INDEX is the authoritative bucket — a player's TEAMNAME can
		// legitimately differ from teamAName/teamBName (roster override, or
		// still-unassigned). Fall back to name matching only for matches
		// parsed before SIDE_INDEX existed.
		slot := 1
		if sideIndex.Valid {
			slot = int(sideIndex.Int64)
		} else if teamName == teamAName {
			slot = 0
		}

		if !stats[slot].Inited {
			if slot == 0 {
				stats[slot].ClanName = teamAName
			} else {
				stats[slot].ClanName = teamBName
			}
			stats[slot].ID = slot + 1
			stats[slot].Inited = true
			stats[slot].PlayingPlayers = make(map[int64]model.Player)
			if slot == 0 {
				stats[slot].EndScore, stats[slot].TScore, stats[slot].CTScore = A_FINAL, A_T, A_CT
			} else {
				stats[slot].EndScore, stats[slot].TScore, stats[slot].CTScore = B_FINAL, B_T, B_CT
			}
		}

		stats[slot].PlayingPlayers[playerID] = buildPlayer(playerName, playerID, buildPlayerInput{
			kills, hs, entryKills, entryDeaths, deaths, assists,
			damage, utilityDamage, k1, k2, k3, k4, k5,
			tradedDeaths, tradeKills, clutchesWon, clutchCount,
			flashAssists, teamFlashed, enemiesFlashed,
			heDamage, fireDamage, kastRounds, totalRounds,
		})
	}

	if err := rows.Err(); err != nil {
		log.Printf("Row iteration error for match %v: %v", row.ID, err)
		return c.Status(500).JSON(fiber.Map{
			"message": "Error while reading advanced stats",
			"success": "false",
		})
	}
	log.Print(fiber.Map{
		"stats":   stats,
		"mapName": row.Map,
	})
	return c.Status(200).JSON(fiber.Map{
		"stats":   stats,
		"mapName": row.Map,
	})
}

type buildPlayerInput struct {
	Kills, HS, EntryKills, EntryDeaths, Deaths, Assists int
	Damage, UtilityDamage                               int
	K1, K2, K3, K4, K5                                  int
	TradedDeaths, TradeKills                            int
	ClutchesWon, ClutchCount                            int
	FlashAssists, TeamFlashed, EnemiesFlashed           int
	HEDamage, FireDamage                                int
	KastRounds, TotalRounds                             int
}

func buildPlayer(name string, id int64, in buildPlayerInput) model.Player {
	adr, kastPct, rating := 0.0, 0.0, 0.0

	if in.TotalRounds > 0 {
		adr = float64(in.Damage) / float64(in.TotalRounds)
		kastPct = float64(in.KastRounds) / float64(in.TotalRounds) * 100

		kpr := float64(in.Kills) / float64(in.TotalRounds)
		apr := float64(in.Assists) / float64(in.TotalRounds)
		dpr := float64(in.Deaths) / float64(in.TotalRounds)

		// HLTV Rating 2.0-style formula — matches team/handlers.go's Summary
		// and PlayerStats so match-level and team-level ratings agree.
		impact := 2.13*kpr + 0.42*apr - 0.41
		rating = 0.0073*kastPct + 0.3591*kpr - 0.5329*dpr + 0.2372*impact + 0.0032*adr + 0.1587
	}

	return model.Player{
		Name: name,
		ID:   id,
		Stats: model.PlayerStats{
			Kills:          in.Kills,
			HeadshotKills:  in.HS,
			EntryKills:     in.EntryKills,
			EntryDeaths:    in.EntryDeaths,
			Deaths:         in.Deaths,
			Assists:        in.Assists,
			Damage:         in.Damage,
			UtilityDamage:  in.UtilityDamage,
			OneFragCount:   in.K1,
			TwoFrags:       in.K2,
			ThreeFrags:     in.K3,
			FourFrags:      in.K4,
			FiveFrags:      in.K5,
			TradedDeaths:   in.TradedDeaths,
			TradeKills:     in.TradeKills,
			ClutchesWon:    in.ClutchesWon,
			ClutchCount:    in.ClutchCount,
			FlashAssists:   in.FlashAssists,
			TeamFlashed:    in.TeamFlashed,
			EnemiesFlashed: in.EnemiesFlashed,
			HEDamage:       in.HEDamage,
			FireDamage:     in.FireDamage,
			ADR:            round2(adr),
			KAST:           round2(kastPct),
			Rating:         round2(rating),
		},
	}
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

type EntryPlayerStats struct {
	PlayerID          int64  `json:"playerId,string"`
	Name              string `json:"name"`
	Team              string `json:"team"`
	EntryKillsT       int    `json:"entryKillsT"`
	EntryDeathsT      int    `json:"entryDeathsT"`
	EntryKillsCT      int    `json:"entryKillsCT"`
	EntryDeathsCT     int    `json:"entryDeathsCT"`
	TradedEntryDeaths int    `json:"tradedEntryDeaths"`
	// Trade fields cover every death in the match, not just entry deaths.
	TradeOpportunities       int `json:"tradeOpportunities"`
	TradesConverted          int `json:"tradesConverted"`
	MissedTradeOpportunities int `json:"missedTradeOpportunities"`
	UntradableDeaths         int `json:"untradableDeaths"`
}

// TeamEntryStats is the per-team sum of EntryPlayerStats across its roster.
type TeamEntryStats struct {
	Team                     string `json:"team"`
	EntryKillsT              int    `json:"entryKillsT"`
	EntryDeathsT             int    `json:"entryDeathsT"`
	EntryKillsCT             int    `json:"entryKillsCT"`
	EntryDeathsCT            int    `json:"entryDeathsCT"`
	TradedEntryDeaths        int    `json:"tradedEntryDeaths"`
	TradeOpportunities       int    `json:"tradeOpportunities"`
	TradesConverted          int    `json:"tradesConverted"`
	MissedTradeOpportunities int    `json:"missedTradeOpportunities"`
	UntradableDeaths         int    `json:"untradableDeaths"`
}

// OpeningDuel is the round's first kill: who got it, on which side, against
// whom, and how far into the round it happened.
type OpeningDuel struct {
	RoundNo        int     `json:"roundNo"`
	ElapsedSeconds float64 `json:"elapsedSeconds,omitempty"`
	KillerName     string  `json:"killerName,omitempty"`
	KillerTeam     string  `json:"killerTeam,omitempty"`
	KillerSide     string  `json:"killerSide,omitempty"`
	VictimName     string  `json:"victimName,omitempty"`
	VictimTeam     string  `json:"victimTeam,omitempty"`
	VictimSide     string  `json:"victimSide,omitempty"`
	Traded         bool    `json:"traded"`
	TBuyType       string  `json:"tBuyType,omitempty"`
	CTBuyType      string  `json:"ctBuyType,omitempty"`
	WinningTeam    string  `json:"winningTeam,omitempty"`
}

func sideLabel(side sql.NullInt64) string {
	if !side.Valid {
		return ""
	}
	switch int(side.Int64) {
	case sideT:
		return "T"
	case sideCT:
		return "CT"
	default:
		return ""
	}
}

// buyTypeLabel mirrors the codes classifyBuy assigns in parser_setup.go
// (1=Pistol, 2=Eco, 3=Force, 4=Full).
func buyTypeLabel(code sql.NullInt64) string {
	if !code.Valid {
		return ""
	}
	switch code.Int64 {
	case 1:
		return "Pistol"
	case 2:
		return "Eco"
	case 3:
		return "Force"
	case 4:
		return "Full"
	default:
		return ""
	}
}

// EntryStats breaks opening-duel kills/deaths down by side (which
// MATCH_STATS.ENTRY_KILLS/ENTRY_DEATHS don't distinguish), rolls per-player
// numbers up into team totals, tracks trade opportunities across every death
// in the match (not just entries), and lists each round's opening duel.
func EntryStats(c fiber.Ctx) error {
	_, ok := c.Locals("accountID").(int64)
	if !ok {
		return c.Status(401).SendString("Unauthorized")
	}

	matchId, err := strconv.ParseInt(c.Params("matchId"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid match id"})
	}

	var teamAName, teamBName string
	if err := database.DB.QueryRow("SELECT COALESCE(TEAM_A_NAME, ''), COALESCE(TEAM_B_NAME, '') FROM MATCHES WHERE MATCHID = ?", matchId).Scan(&teamAName, &teamBName); err != nil {
		log.Printf("SQL ERROR (entry stats match lookup): %v", err)
		return c.Status(500).JSON(fiber.Map{
			"message": "Failed to look up match",
			"success": "false",
		})
	}

	query := `
		SELECT p.PLAYERID, p.PLAYERNAME, COALESCE(ms.TEAMNAME, ''), ms.SIDE_INDEX,
			rp.SIDE, rp.FIRST_KILL, rp.FIRST_DEATH,
			rp.GOT_TRADED, rp.TRADE_OPPORTUNITY, rp.SURVIVED
		FROM ROUND_PARTICIPANTS rp
		JOIN PLAYERS p ON p.PLAYERID = rp.PLAYERID
		JOIN MATCH_STATS ms ON ms.MATCHID = rp.MATCHID AND ms.PLAYERID = rp.PLAYERID
		WHERE rp.MATCHID = ?`

	rows, err := database.DB.Query(query, matchId)
	if err != nil {
		log.Printf("SQL ERROR (entry stats query): %v", err)
		return c.Status(500).JSON(fiber.Map{
			"message": "Failed to query entry stats",
			"success": "false",
		})
	}
	defer rows.Close()

	players := make(map[int64]*EntryPlayerStats)
	// playerSlot is the authoritative team-A(0)/team-B(1) bucket for rollup —
	// a player's TEAMNAME can legitimately differ from teamAName/teamBName
	// (roster override, or still-unassigned), so it can't be used to match.
	playerSlot := make(map[int64]int)
	order := make([]int64, 0)

	for rows.Next() {
		var playerID int64
		var playerName, teamName string
		var sideIndex sql.NullInt64
		var side int
		var firstKill, firstDeath, gotTraded, tradeOpportunity, survived bool

		if err := rows.Scan(&playerID, &playerName, &teamName, &sideIndex, &side,
			&firstKill, &firstDeath, &gotTraded, &tradeOpportunity, &survived); err != nil {
			log.Printf("Row scan error for match %v: %v", matchId, err)
			return c.Status(500).JSON(fiber.Map{
				"message": "Failed to read entry stats row",
				"success": "false",
			})
		}

		p, ok := players[playerID]
		if !ok {
			p = &EntryPlayerStats{PlayerID: playerID, Name: playerName, Team: teamName}
			players[playerID] = p
			order = append(order, playerID)
			if sideIndex.Valid {
				playerSlot[playerID] = int(sideIndex.Int64)
			} else if teamName != teamAName {
				playerSlot[playerID] = 1
			}
		}

		switch {
		case firstKill && side == sideT:
			p.EntryKillsT++
		case firstKill && side == sideCT:
			p.EntryKillsCT++
		case firstDeath && side == sideT:
			p.EntryDeathsT++
			if gotTraded {
				p.TradedEntryDeaths++
			}
		case firstDeath && side == sideCT:
			p.EntryDeathsCT++
			if gotTraded {
				p.TradedEntryDeaths++
			}
		}

		if !survived {
			switch {
			case tradeOpportunity && gotTraded:
				p.TradeOpportunities++
				p.TradesConverted++
			case tradeOpportunity:
				p.TradeOpportunities++
				p.MissedTradeOpportunities++
			default:
				p.UntradableDeaths++
			}
		}
	}

	if err := rows.Err(); err != nil {
		log.Printf("Row iteration error for match %v: %v", matchId, err)
		return c.Status(500).JSON(fiber.Map{
			"message": "Error while reading entry stats",
			"success": "false",
		})
	}

	playerResult := make([]*EntryPlayerStats, 0, len(order))
	// Always report team A before team B so the frontend can rely on a
	// stable left/right order instead of database row order.
	slotTeams := [2]*TeamEntryStats{{Team: teamAName}, {Team: teamBName}}
	for _, id := range order {
		p := players[id]
		playerResult = append(playerResult, p)

		t := slotTeams[playerSlot[id]]
		t.EntryKillsT += p.EntryKillsT
		t.EntryDeathsT += p.EntryDeathsT
		t.EntryKillsCT += p.EntryKillsCT
		t.EntryDeathsCT += p.EntryDeathsCT
		t.TradedEntryDeaths += p.TradedEntryDeaths
		t.TradeOpportunities += p.TradeOpportunities
		t.TradesConverted += p.TradesConverted
		t.MissedTradeOpportunities += p.MissedTradeOpportunities
		t.UntradableDeaths += p.UntradableDeaths
	}

	teamResult := []*TeamEntryStats{slotTeams[0], slotTeams[1]}

	// INNER JOIN on krp (FIRST_KILL) so rounds with no opening kill at all
	// (nobody fought — e.g. a full double-save) are omitted rather than
	// showing up with empty killer/victim columns.
	roundQuery := `
		SELECT r.ROUND_NO, r.OPENING_KILL_SECONDS, r.T_TEAM, r.CT_TEAM,
			r.BUY_TYPE_T, r.BUY_TYPE_CT, r.WINNING_SIDE,
			kp.PLAYERNAME, krp.SIDE,
			vp.PLAYERNAME, vrp.SIDE, vrp.GOT_TRADED
		FROM ROUNDS r
		JOIN ROUND_PARTICIPANTS krp ON krp.MATCHID = r.MATCHID AND krp.ROUND_NO = r.ROUND_NO AND krp.FIRST_KILL = 1
		JOIN PLAYERS kp ON kp.PLAYERID = krp.PLAYERID
		LEFT JOIN ROUND_PARTICIPANTS vrp ON vrp.MATCHID = r.MATCHID AND vrp.ROUND_NO = r.ROUND_NO AND vrp.FIRST_DEATH = 1
		LEFT JOIN PLAYERS vp ON vp.PLAYERID = vrp.PLAYERID
		WHERE r.MATCHID = ?
		ORDER BY r.ROUND_NO`

	roundRows, err := database.DB.Query(roundQuery, matchId)
	if err != nil {
		log.Printf("SQL ERROR (opening duel query): %v", err)
		return c.Status(500).JSON(fiber.Map{
			"message": "Failed to query opening duels",
			"success": "false",
		})
	}
	defer roundRows.Close()

	rounds := make([]*OpeningDuel, 0)
	for roundRows.Next() {
		var roundNo int
		var elapsedSeconds sql.NullFloat64
		var tTeam, ctTeam, killerName, victimName sql.NullString
		var tBuyType, ctBuyType sql.NullInt64
		var winningSide sql.NullInt64
		var killerSide, victimSide sql.NullInt64
		var traded sql.NullBool

		if err := roundRows.Scan(&roundNo, &elapsedSeconds, &tTeam, &ctTeam,
			&tBuyType, &ctBuyType, &winningSide,
			&killerName, &killerSide, &victimName, &victimSide, &traded); err != nil {
			log.Printf("Row scan error for match %v: %v", matchId, err)
			return c.Status(500).JSON(fiber.Map{
				"message": "Failed to read opening duel row",
				"success": "false",
			})
		}

		duel := &OpeningDuel{
			RoundNo:        roundNo,
			ElapsedSeconds: elapsedSeconds.Float64,
			KillerName:     killerName.String,
			KillerSide:     sideLabel(killerSide),
			VictimName:     victimName.String,
			VictimSide:     sideLabel(victimSide),
			Traded:         traded.Bool,
			TBuyType:       buyTypeLabel(tBuyType),
			CTBuyType:      buyTypeLabel(ctBuyType),
		}
		if killerSide.Valid && int(killerSide.Int64) == sideT {
			duel.KillerTeam = tTeam.String
		} else if killerSide.Valid && int(killerSide.Int64) == sideCT {
			duel.KillerTeam = ctTeam.String
		}
		if victimSide.Valid && int(victimSide.Int64) == sideT {
			duel.VictimTeam = tTeam.String
		} else if victimSide.Valid && int(victimSide.Int64) == sideCT {
			duel.VictimTeam = ctTeam.String
		}
		if winningSide.Valid && int(winningSide.Int64) == sideT {
			duel.WinningTeam = tTeam.String
		} else if winningSide.Valid && int(winningSide.Int64) == sideCT {
			duel.WinningTeam = ctTeam.String
		}

		rounds = append(rounds, duel)
	}
	if err := roundRows.Err(); err != nil {
		log.Printf("Row iteration error for match %v: %v", matchId, err)
		return c.Status(500).JSON(fiber.Map{
			"message": "Error while reading opening duels",
			"success": "false",
		})
	}

	return c.Status(200).JSON(fiber.Map{
		"players": playerResult,
		"teams":   teamResult,
		"rounds":  rounds,
	})
}

type UtilityPlayerStats struct {
	PlayerID       int64          `json:"playerId,string"`
	Name           string         `json:"name"`
	Team           string         `json:"team"`
	UtilityDamage  int            `json:"utilityDamage"`
	HEDamage       int            `json:"heDamage"`
	FireDamage     int            `json:"fireDamage"`
	FlashAssists   int            `json:"flashAssists"`
	TeamFlashed    int            `json:"teamFlashed"`
	EnemiesFlashed int            `json:"enemiesFlashed"`
	FlashesThrown  int            `json:"flashesThrown"`
	AvgBlindTime   float64        `json:"avgBlindTime"`
	FlashesToKill  int            `json:"flashesToKill"`
	GrenadesThrown map[string]int `json:"grenadesThrown"`
}

// TeamUtilityStats is the per-team sum of UtilityPlayerStats across its
// roster. AvgBlindTime is not a sum of player averages (that would weight
// players unevenly) — it's computed directly from the underlying
// PLAYER_FLASHES rows for the whole team.
type TeamUtilityStats struct {
	Team           string         `json:"team"`
	UtilityDamage  int            `json:"utilityDamage"`
	HEDamage       int            `json:"heDamage"`
	FireDamage     int            `json:"fireDamage"`
	FlashAssists   int            `json:"flashAssists"`
	TeamFlashed    int            `json:"teamFlashed"`
	EnemiesFlashed int            `json:"enemiesFlashed"`
	FlashesThrown  int            `json:"flashesThrown"`
	AvgBlindTime   float64        `json:"avgBlindTime"`
	FlashesToKill  int            `json:"flashesToKill"`
	GrenadesThrown map[string]int `json:"grenadesThrown"`
}

var grenadeTypeNames = map[int]string{
	int(common.EqDecoy):      "decoy",
	int(common.EqMolotov):    "molotov",
	int(common.EqIncendiary): "incendiary",
	int(common.EqFlash):      "flashbang",
	int(common.EqSmoke):      "smoke",
	int(common.EqHE):         "hegrenade",
}

// UtilityStats layers grenade-throw counts and flash effectiveness (from
// GRENADE_EVENTS / PLAYER_FLASHES) on top of the utility damage numbers
// already aggregated in MATCH_STATS.
func UtilityStats(c fiber.Ctx) error {
	_, ok := c.Locals("accountID").(int64)
	if !ok {
		return c.Status(401).SendString("Unauthorized")
	}

	matchId, err := strconv.ParseInt(c.Params("matchId"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid match id"})
	}

	var teamAName, teamBName string
	if err := database.DB.QueryRow("SELECT COALESCE(TEAM_A_NAME, ''), COALESCE(TEAM_B_NAME, '') FROM MATCHES WHERE MATCHID = ?", matchId).Scan(&teamAName, &teamBName); err != nil {
		log.Printf("SQL ERROR (utility stats match lookup): %v", err)
		return c.Status(500).JSON(fiber.Map{
			"message": "Failed to look up match",
			"success": "false",
		})
	}

	players := make(map[int64]*UtilityPlayerStats)
	// playerSlot is the authoritative team-A(0)/team-B(1) bucket for rollup —
	// a player's TEAMNAME can legitimately differ from teamAName/teamBName
	// (roster override, or still-unassigned), so it can't be used to match.
	playerSlot := make(map[int64]int)
	order := make([]int64, 0)

	baseQuery := `
		SELECT p.PLAYERID, p.PLAYERNAME, COALESCE(ms.TEAMNAME, ''), ms.SIDE_INDEX,
			ms.UTILITY_DAMAGE, ms.HE_DAMAGE, ms.FIRE_DAMAGE,
			ms.FLASH_ASSISTS, ms.TEAM_FLASHES, ms.ENEMIES_FLASHED
		FROM MATCH_STATS ms
		JOIN PLAYERS p ON p.PLAYERID = ms.PLAYERID
		WHERE ms.MATCHID = ?`

	rows, err := database.DB.Query(baseQuery, matchId)
	if err != nil {
		log.Printf("SQL ERROR (utility stats query): %v", err)
		return c.Status(500).JSON(fiber.Map{
			"message": "Failed to query utility stats",
			"success": "false",
		})
	}
	defer rows.Close()

	for rows.Next() {
		var playerID int64
		var playerName, teamName string
		var sideIndex sql.NullInt64
		var utilityDamage, heDamage, fireDamage int
		var flashAssists, teamFlashed, enemiesFlashed int

		if err := rows.Scan(&playerID, &playerName, &teamName, &sideIndex,
			&utilityDamage, &heDamage, &fireDamage,
			&flashAssists, &teamFlashed, &enemiesFlashed); err != nil {
			log.Printf("Row scan error for match %v: %v", matchId, err)
			return c.Status(500).JSON(fiber.Map{
				"message": "Failed to read utility stats row",
				"success": "false",
			})
		}

		players[playerID] = &UtilityPlayerStats{
			PlayerID:       playerID,
			Name:           playerName,
			Team:           teamName,
			UtilityDamage:  utilityDamage,
			HEDamage:       heDamage,
			FireDamage:     fireDamage,
			FlashAssists:   flashAssists,
			TeamFlashed:    teamFlashed,
			EnemiesFlashed: enemiesFlashed,
			GrenadesThrown: make(map[string]int),
		}
		order = append(order, playerID)
		if sideIndex.Valid {
			playerSlot[playerID] = int(sideIndex.Int64)
		} else if teamName != teamAName {
			playerSlot[playerID] = 1
		}
	}
	if err := rows.Err(); err != nil {
		log.Printf("Row iteration error for match %v: %v", matchId, err)
		return c.Status(500).JSON(fiber.Map{
			"message": "Error while reading utility stats",
			"success": "false",
		})
	}

	grenadeQuery := `
		SELECT PLAYERID, GRENADE, COUNT(DISTINCT CONCAT(ROUND_NO, '-', ENTITYID))
		FROM GRENADE_EVENTS
		WHERE MATCHID = ? AND GRENADE IN (?, ?, ?, ?, ?, ?)
		GROUP BY PLAYERID, GRENADE`

	grenRows, err := database.DB.Query(grenadeQuery, matchId,
		int(common.EqDecoy), int(common.EqMolotov), int(common.EqIncendiary),
		int(common.EqFlash), int(common.EqSmoke), int(common.EqHE))
	if err != nil {
		log.Printf("SQL ERROR (grenade throw query): %v", err)
		return c.Status(500).JSON(fiber.Map{
			"message": "Failed to query grenade throws",
			"success": "false",
		})
	}
	defer grenRows.Close()

	for grenRows.Next() {
		var playerID int64
		var grenadeType, count int

		if err := grenRows.Scan(&playerID, &grenadeType, &count); err != nil {
			log.Printf("Row scan error for match %v: %v", matchId, err)
			return c.Status(500).JSON(fiber.Map{
				"message": "Failed to read grenade throw row",
				"success": "false",
			})
		}

		p, ok := players[playerID]
		if !ok {
			continue
		}
		name := grenadeTypeNames[grenadeType]
		p.GrenadesThrown[name] += count
		if grenadeType == int(common.EqFlash) {
			p.FlashesThrown += count
		}
	}
	if err := grenRows.Err(); err != nil {
		log.Printf("Row iteration error for match %v: %v", matchId, err)
		return c.Status(500).JSON(fiber.Map{
			"message": "Error while reading grenade throws",
			"success": "false",
		})
	}

	flashQuery := `
		SELECT THROWERID, AVG(BLIND_DUR), SUM(LED_TO_KILL)
		FROM PLAYER_FLASHES
		WHERE MATCHID = ? AND IS_TEAM_FLASH = 0
		GROUP BY THROWERID`

	flashRows, err := database.DB.Query(flashQuery, matchId)
	if err != nil {
		log.Printf("SQL ERROR (flash effectiveness query): %v", err)
		return c.Status(500).JSON(fiber.Map{
			"message": "Failed to query flash effectiveness",
			"success": "false",
		})
	}
	defer flashRows.Close()

	for flashRows.Next() {
		var throwerID int64
		var avgBlind float64
		var flashesToKill int

		if err := flashRows.Scan(&throwerID, &avgBlind, &flashesToKill); err != nil {
			log.Printf("Row scan error for match %v: %v", matchId, err)
			return c.Status(500).JSON(fiber.Map{
				"message": "Failed to read flash effectiveness row",
				"success": "false",
			})
		}

		if p, ok := players[throwerID]; ok {
			p.AvgBlindTime = round2(avgBlind)
			p.FlashesToKill = flashesToKill
		}
	}
	if err := flashRows.Err(); err != nil {
		log.Printf("Row iteration error for match %v: %v", matchId, err)
		return c.Status(500).JSON(fiber.Map{
			"message": "Error while reading flash effectiveness",
			"success": "false",
		})
	}

	result := make([]*UtilityPlayerStats, 0, len(order))
	slotTeams := [2]*TeamUtilityStats{
		{Team: teamAName, GrenadesThrown: make(map[string]int)},
		{Team: teamBName, GrenadesThrown: make(map[string]int)},
	}
	for _, id := range order {
		p := players[id]
		result = append(result, p)

		t := slotTeams[playerSlot[id]]
		t.UtilityDamage += p.UtilityDamage
		t.HEDamage += p.HEDamage
		t.FireDamage += p.FireDamage
		t.FlashAssists += p.FlashAssists
		t.TeamFlashed += p.TeamFlashed
		t.EnemiesFlashed += p.EnemiesFlashed
		t.FlashesThrown += p.FlashesThrown
		for grenade, count := range p.GrenadesThrown {
			t.GrenadesThrown[grenade] += count
		}
	}

	// AvgBlindTime is computed directly from PLAYER_FLASHES per side rather
	// than averaging the per-player averages, so it isn't skewed by players
	// who threw very few flashes. Grouped by SIDE_INDEX, not TEAMNAME, for
	// the same reason as the player rollup above.
	teamFlashQuery := `
		SELECT ms.SIDE_INDEX, AVG(pf.BLIND_DUR), SUM(pf.LED_TO_KILL)
		FROM PLAYER_FLASHES pf
		JOIN MATCH_STATS ms ON ms.MATCHID = pf.MATCHID AND ms.PLAYERID = pf.THROWERID
		WHERE pf.MATCHID = ? AND pf.IS_TEAM_FLASH = 0 AND ms.SIDE_INDEX IS NOT NULL
		GROUP BY ms.SIDE_INDEX`

	teamFlashRows, err := database.DB.Query(teamFlashQuery, matchId)
	if err != nil {
		log.Printf("SQL ERROR (team flash effectiveness query): %v", err)
		return c.Status(500).JSON(fiber.Map{
			"message": "Failed to query team flash effectiveness",
			"success": "false",
		})
	}
	defer teamFlashRows.Close()

	for teamFlashRows.Next() {
		var sideIndex int
		var avgBlind float64
		var flashesToKill int

		if err := teamFlashRows.Scan(&sideIndex, &avgBlind, &flashesToKill); err != nil {
			log.Printf("Row scan error for match %v: %v", matchId, err)
			return c.Status(500).JSON(fiber.Map{
				"message": "Failed to read team flash effectiveness row",
				"success": "false",
			})
		}

		if sideIndex == 0 || sideIndex == 1 {
			slotTeams[sideIndex].AvgBlindTime = round2(avgBlind)
			slotTeams[sideIndex].FlashesToKill = flashesToKill
		}
	}
	if err := teamFlashRows.Err(); err != nil {
		log.Printf("Row iteration error for match %v: %v", matchId, err)
		return c.Status(500).JSON(fiber.Map{
			"message": "Error while reading team flash effectiveness",
			"success": "false",
		})
	}

	teamResult := []*TeamUtilityStats{slotTeams[0], slotTeams[1]}

	return c.Status(200).JSON(fiber.Map{"players": result, "teams": teamResult})
}

type TeamBombStats struct {
	Team             string  `json:"team"`
	Plants           int     `json:"plants"`
	PlantRoundWins   int     `json:"plantRoundWins"`
	PlantWinPct      float64 `json:"plantWinPct"`
	Defuses          int     `json:"defuses"`
	BombExplodes     int     `json:"bombExplodes"`
	RetakeAttempts   int     `json:"retakeAttempts"`
	RetakeSuccesses  int     `json:"retakeSuccesses"`
	RetakeSuccessPct float64 `json:"retakeSuccessPct"`
	// Saves counts rounds where the team had a live opportunity to contest
	// (retake as CT, or plant as T) but chose to preserve weapons and
	// survive instead of engaging.
	Saves int `json:"saves"`
	// Holds counts rounds played as CT where the bomb was never planted and
	// the team won — i.e. the site was never lost in the first place, no
	// retake needed. CTRoundsPlayed is the denominator for HoldPct.
	Holds          int     `json:"holds"`
	HoldPct        float64 `json:"holdPct"`
	CTRoundsPlayed int     `json:"ctRoundsPlayed"`
}

// TeamSiteStats is one team's plant/retake record at one bombsite. Plants
// splits into PlantWins (the team held after planting there) and
// PlantLosses (the bomb was planted there but the other team retook it) —
// PlantWins + PlantLosses always equals Plants. RetakeAttempts/Successes are
// this team's record defending that site as CT.
type TeamSiteStats struct {
	Team             string  `json:"team"`
	Site             string  `json:"site"`
	Plants           int     `json:"plants"`
	PlantWins        int     `json:"plantWins"`
	PlantLosses      int     `json:"plantLosses"`
	PlantWinPct      float64 `json:"plantWinPct"`
	RetakeAttempts   int     `json:"retakeAttempts"`
	RetakeSuccesses  int     `json:"retakeSuccesses"`
	RetakeSuccessPct float64 `json:"retakeSuccessPct"`
}

func teamSiteKey(team, site string) string {
	return team + "|" + site
}

// RoundBombEvent is the per-round detail behind the TeamBombStats
// aggregates: where the bomb was planted, who planted/defused it, and how
// the round was decided.
type RoundBombEvent struct {
	RoundNo       int    `json:"roundNo"`
	Site          string `json:"site,omitempty"`
	PlantingTeam  string `json:"plantingTeam,omitempty"`
	Planter       string `json:"planter,omitempty"`
	DefendingTeam string `json:"defendingTeam,omitempty"`
	Defuser       string `json:"defuser,omitempty"`
	BombPlanted   bool   `json:"bombPlanted"`
	BombDefused   bool   `json:"bombDefused"`
	BombExploded  bool   `json:"bombExploded"`
	WinningTeam   string `json:"winningTeam,omitempty"`
	// Outcome is one of: retake_success, save, retake_failed, t_save,
	// t_eliminated, t_win_no_plant.
	Outcome string `json:"outcome"`
}

// PlantsAndRetakes aggregates round-level bomb outcomes from ROUNDS. T_TEAM/
// CT_TEAM record which clan held which side that round, so plants/retakes
// can be attributed correctly even across side switches. "Saves" are
// inferred from survivors: a round the team lost (bomb went off / time ran
// out) where at least one of their players was still alive means they chose
// not to contest it rather than dying trying.
func PlantsAndRetakes(c fiber.Ctx) error {
	_, ok := c.Locals("accountID").(int64)
	if !ok {
		return c.Status(401).SendString("Unauthorized")
	}

	matchId, err := strconv.ParseInt(c.Params("matchId"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid match id"})
	}

	var teamAName, teamBName string
	if err := database.DB.QueryRow("SELECT COALESCE(TEAM_A_NAME, ''), COALESCE(TEAM_B_NAME, '') FROM MATCHES WHERE MATCHID = ?", matchId).Scan(&teamAName, &teamBName); err != nil {
		log.Printf("SQL ERROR (plants/retakes match lookup): %v", err)
		return c.Status(500).JSON(fiber.Map{
			"message": "Failed to look up match",
			"success": "false",
		})
	}

	teams := map[string]*TeamBombStats{
		teamAName: {Team: teamAName},
		teamBName: {Team: teamBName},
	}
	teamSites := map[string]*TeamSiteStats{
		teamSiteKey(teamAName, "A"): {Team: teamAName, Site: "A"},
		teamSiteKey(teamAName, "B"): {Team: teamAName, Site: "B"},
		teamSiteKey(teamBName, "A"): {Team: teamBName, Site: "A"},
		teamSiteKey(teamBName, "B"): {Team: teamBName, Site: "B"},
	}

	survivorQuery := `
		SELECT ROUND_NO, SIDE, COUNT(*)
		FROM ROUND_PARTICIPANTS
		WHERE MATCHID = ? AND SURVIVED = 1
		GROUP BY ROUND_NO, SIDE`

	survivorRows, err := database.DB.Query(survivorQuery, matchId)
	if err != nil {
		log.Printf("SQL ERROR (survivor query): %v", err)
		return c.Status(500).JSON(fiber.Map{
			"message": "Failed to query round survivors",
			"success": "false",
		})
	}
	defer survivorRows.Close()

	survivors := make(map[int]map[int]int)
	for survivorRows.Next() {
		var roundNo, side, count int
		if err := survivorRows.Scan(&roundNo, &side, &count); err != nil {
			log.Printf("Row scan error for match %v: %v", matchId, err)
			return c.Status(500).JSON(fiber.Map{
				"message": "Failed to read round survivors row",
				"success": "false",
			})
		}
		if survivors[roundNo] == nil {
			survivors[roundNo] = make(map[int]int)
		}
		survivors[roundNo][side] = count
	}
	if err := survivorRows.Err(); err != nil {
		log.Printf("Row iteration error for match %v: %v", matchId, err)
		return c.Status(500).JSON(fiber.Map{
			"message": "Error while reading round survivors",
			"success": "false",
		})
	}

	query := `
		SELECT r.ROUND_NO, r.T_TEAM, r.CT_TEAM, r.SITE,
			r.BOMB_PLANT, r.BOMB_DEFUSE, r.BOMB_EXPLODE, r.WINNING_SIDE,
			pl.PLAYERNAME, df.PLAYERNAME
		FROM ROUNDS r
		LEFT JOIN PLAYERS pl ON pl.PLAYERID = r.PLANTER_PLAYERID
		LEFT JOIN PLAYERS df ON df.PLAYERID = r.DEFUSER_PLAYERID
		WHERE r.MATCHID = ?
		ORDER BY r.ROUND_NO`

	rows, err := database.DB.Query(query, matchId)
	if err != nil {
		log.Printf("SQL ERROR (plants/retakes query): %v", err)
		return c.Status(500).JSON(fiber.Map{
			"message": "Failed to query plants/retakes",
			"success": "false",
		})
	}
	defer rows.Close()

	rounds := make([]*RoundBombEvent, 0)

	for rows.Next() {
		var roundNo int
		var tTeam, ctTeam, site sql.NullString
		var bombPlant, bombDefuse, bombExplode bool
		var winningSide sql.NullInt64
		var planterName, defuserName sql.NullString

		if err := rows.Scan(&roundNo, &tTeam, &ctTeam, &site,
			&bombPlant, &bombDefuse, &bombExplode, &winningSide,
			&planterName, &defuserName); err != nil {
			log.Printf("Row scan error for match %v: %v", matchId, err)
			return c.Status(500).JSON(fiber.Map{
				"message": "Failed to read plants/retakes row",
				"success": "false",
			})
		}

		t := teams[tTeam.String]
		ct := teams[ctTeam.String]

		if ct != nil {
			ct.CTRoundsPlayed++
		}

		var winningTeam string
		if winningSide.Valid {
			if int(winningSide.Int64) == sideT {
				winningTeam = tTeam.String
			} else if int(winningSide.Int64) == sideCT {
				winningTeam = ctTeam.String
			}
		}

		tSurvivors := survivors[roundNo][sideT]
		ctSurvivors := survivors[roundNo][sideCT]

		event := &RoundBombEvent{
			RoundNo:       roundNo,
			Site:          site.String,
			DefendingTeam: ctTeam.String,
			Defuser:       defuserName.String,
			BombPlanted:   bombPlant,
			BombDefused:   bombDefuse,
			BombExploded:  bombExplode,
			WinningTeam:   winningTeam,
		}

		if bombPlant {
			event.PlantingTeam = tTeam.String
			event.Planter = planterName.String

			if t != nil {
				t.Plants++
			}
			if ct != nil {
				ct.RetakeAttempts++
			}
			if bombDefuse && ct != nil {
				ct.Defuses++
			}
			if bombExplode && t != nil {
				t.BombExplodes++
			}

			var plantSite, retakeSite *TeamSiteStats
			if site.Valid {
				plantSite = teamSites[teamSiteKey(tTeam.String, site.String)]
				retakeSite = teamSites[teamSiteKey(ctTeam.String, site.String)]
			}
			if plantSite != nil {
				plantSite.Plants++
			}
			if retakeSite != nil {
				retakeSite.RetakeAttempts++
			}

			switch {
			case winningSide.Valid && int(winningSide.Int64) == sideCT:
				event.Outcome = "retake_success"
				if ct != nil {
					ct.RetakeSuccesses++
				}
				if plantSite != nil {
					plantSite.PlantLosses++
				}
				if retakeSite != nil {
					retakeSite.RetakeSuccesses++
				}
			case winningSide.Valid && int(winningSide.Int64) == sideT:
				if t != nil {
					t.PlantRoundWins++
				}
				if plantSite != nil {
					plantSite.PlantWins++
				}
				if ctSurvivors > 0 {
					event.Outcome = "save"
					if ct != nil {
						ct.Saves++
					}
				} else {
					event.Outcome = "retake_failed"
				}
			}
		} else {
			switch {
			case winningSide.Valid && int(winningSide.Int64) == sideCT:
				if ct != nil {
					ct.Holds++
				}
				if tSurvivors > 0 {
					event.Outcome = "t_save"
					if t != nil {
						t.Saves++
					}
				} else {
					event.Outcome = "t_eliminated"
				}
			case winningSide.Valid && int(winningSide.Int64) == sideT:
				event.Outcome = "t_win_no_plant"
			}
		}

		// Only surface rounds with an actual plant in the round-by-round
		// list — team/site aggregates above still account for every round
		// (holds/saves need the no-plant rounds too).
		if bombPlant {
			rounds = append(rounds, event)
		}
	}

	if err := rows.Err(); err != nil {
		log.Printf("Row iteration error for match %v: %v", matchId, err)
		return c.Status(500).JSON(fiber.Map{
			"message": "Error while reading plants/retakes",
			"success": "false",
		})
	}

	result := make([]*TeamBombStats, 0, 2)
	for _, name := range []string{teamAName, teamBName} {
		t := teams[name]
		if t.Plants > 0 {
			t.PlantWinPct = round2(float64(t.PlantRoundWins) / float64(t.Plants) * 100)
		}
		if t.RetakeAttempts > 0 {
			t.RetakeSuccessPct = round2(float64(t.RetakeSuccesses) / float64(t.RetakeAttempts) * 100)
		}
		if t.CTRoundsPlayed > 0 {
			t.HoldPct = round2(float64(t.Holds) / float64(t.CTRoundsPlayed) * 100)
		}
		result = append(result, t)
	}

	teamSiteResult := make([]*TeamSiteStats, 0, 4)
	for _, teamName := range []string{teamAName, teamBName} {
		for _, site := range []string{"A", "B"} {
			s := teamSites[teamSiteKey(teamName, site)]
			if s.Plants > 0 {
				s.PlantWinPct = round2(float64(s.PlantWins) / float64(s.Plants) * 100)
			}
			if s.RetakeAttempts > 0 {
				s.RetakeSuccessPct = round2(float64(s.RetakeSuccesses) / float64(s.RetakeAttempts) * 100)
			}
			teamSiteResult = append(teamSiteResult, s)
		}
	}

	return c.Status(200).JSON(fiber.Map{"teams": result, "rounds": rounds, "teamSites": teamSiteResult})
}

type RoundResult struct {
	RoundNo     int    `json:"roundNo"`
	WinningSide string `json:"winningSide,omitempty"`
}

// RoundResults is a lightweight per-round win/loss summary (by side, not
// team) meant for UI like the 2D playback round selector, which just needs
// to color-code rounds without pulling in the full plants/retakes payload.
func RoundResults(c fiber.Ctx) error {
	_, ok := c.Locals("accountID").(int64)
	if !ok {
		return c.Status(401).SendString("Unauthorized")
	}

	matchId, err := strconv.ParseInt(c.Params("matchId"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid match id"})
	}

	rows, err := database.DB.Query("SELECT ROUND_NO, WINNING_SIDE FROM ROUNDS WHERE MATCHID = ? ORDER BY ROUND_NO", matchId)
	if err != nil {
		log.Printf("SQL ERROR (round results query): %v", err)
		return c.Status(500).JSON(fiber.Map{
			"message": "Failed to query round results",
			"success": "false",
		})
	}
	defer rows.Close()

	results := make([]RoundResult, 0)
	for rows.Next() {
		var roundNo int
		var winningSide sql.NullInt64

		if err := rows.Scan(&roundNo, &winningSide); err != nil {
			log.Printf("Row scan error for match %v: %v", matchId, err)
			return c.Status(500).JSON(fiber.Map{
				"message": "Failed to read round results row",
				"success": "false",
			})
		}

		results = append(results, RoundResult{RoundNo: roundNo, WinningSide: sideLabel(winningSide)})
	}
	if err := rows.Err(); err != nil {
		log.Printf("Row iteration error for match %v: %v", matchId, err)
		return c.Status(500).JSON(fiber.Map{
			"message": "Error while reading round results",
			"success": "false",
		})
	}

	return c.Status(200).JSON(fiber.Map{"rounds": results})
}

type PlayerClutchStats struct {
	PlayerID int64   `json:"playerId,string"`
	Name     string  `json:"name"`
	Team     string  `json:"team"`
	Attempts int     `json:"attempts"`
	Wins     int     `json:"wins"`
	WinPct   float64 `json:"winPct"`
}

// TeamClutchSituation is one team's record in a given "1vN" situation,
// where N is the number of opponents alive when the clutch began.
type TeamClutchSituation struct {
	Team      string  `json:"team"`
	Opponents int     `json:"opponents"`
	Attempts  int     `json:"attempts"`
	Wins      int     `json:"wins"`
	WinPct    float64 `json:"winPct"`
}

// ClutchRound is a single clutch situation that occurred in the match.
type ClutchRound struct {
	RoundNo    int    `json:"roundNo"`
	PlayerName string `json:"playerName"`
	Team       string `json:"team"`
	Side       string `json:"side"`
	Opponents  int    `json:"opponents"`
	Won        bool   `json:"won"`
}

func clutchSituationKey(team string, opponents int) string {
	return fmt.Sprintf("%s|%d", team, opponents)
}

// ClutchStats tracks 1vN clutch situations from the CLUTCHES table: overall
// per-player record, each team's win rate broken down by how many opponents
// were alive (1v1..1v5), and the round-by-round list of clutches that
// actually happened (rounds with no clutch are simply absent).
func ClutchStats(c fiber.Ctx) error {
	_, ok := c.Locals("accountID").(int64)
	if !ok {
		return c.Status(401).SendString("Unauthorized")
	}

	matchId, err := strconv.ParseInt(c.Params("matchId"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid match id"})
	}

	var teamAName, teamBName string
	if err := database.DB.QueryRow("SELECT COALESCE(TEAM_A_NAME, ''), COALESCE(TEAM_B_NAME, '') FROM MATCHES WHERE MATCHID = ?", matchId).Scan(&teamAName, &teamBName); err != nil {
		log.Printf("SQL ERROR (clutch stats match lookup): %v", err)
		return c.Status(500).JSON(fiber.Map{
			"message": "Failed to look up match",
			"success": "false",
		})
	}

	query := `
		SELECT cl.ROUND_NO, p.PLAYERID, p.PLAYERNAME, COALESCE(ms.TEAMNAME, ''), ms.SIDE_INDEX, cl.SIDE, cl.OPPONENTS, cl.WON
		FROM CLUTCHES cl
		JOIN PLAYERS p ON p.PLAYERID = cl.PLAYERID
		JOIN MATCH_STATS ms ON ms.MATCHID = cl.MATCHID AND ms.PLAYERID = cl.PLAYERID
		WHERE cl.MATCHID = ?
		ORDER BY cl.ROUND_NO`

	rows, err := database.DB.Query(query, matchId)
	if err != nil {
		log.Printf("SQL ERROR (clutch stats query): %v", err)
		return c.Status(500).JSON(fiber.Map{
			"message": "Failed to query clutch stats",
			"success": "false",
		})
	}
	defer rows.Close()

	players := make(map[int64]*PlayerClutchStats)
	order := make([]int64, 0)
	situations := make(map[string]*TeamClutchSituation)
	rounds := make([]*ClutchRound, 0)

	for rows.Next() {
		var roundNo int
		var playerID int64
		var playerName, teamName string
		var sideIndex sql.NullInt64
		var side, opponents int
		var won bool

		if err := rows.Scan(&roundNo, &playerID, &playerName, &teamName, &sideIndex, &side, &opponents, &won); err != nil {
			log.Printf("Row scan error for match %v: %v", matchId, err)
			return c.Status(500).JSON(fiber.Map{
				"message": "Failed to read clutch stats row",
				"success": "false",
			})
		}

		p, ok := players[playerID]
		if !ok {
			p = &PlayerClutchStats{PlayerID: playerID, Name: playerName, Team: teamName}
			players[playerID] = p
			order = append(order, playerID)
		}
		p.Attempts++
		if won {
			p.Wins++
		}

		// situations is keyed by teamAName/teamBName (not the player's own
		// TEAMNAME, which can legitimately differ — roster override or
		// still-unassigned) so it lines up with the situationResult lookup
		// below.
		teamLabel := teamAName
		if (sideIndex.Valid && sideIndex.Int64 == 1) || (!sideIndex.Valid && teamName != teamAName) {
			teamLabel = teamBName
		}
		key := clutchSituationKey(teamLabel, opponents)
		s, ok := situations[key]
		if !ok {
			s = &TeamClutchSituation{Team: teamLabel, Opponents: opponents}
			situations[key] = s
		}
		s.Attempts++
		if won {
			s.Wins++
		}

		var sideStr string
		if side == sideT {
			sideStr = "T"
		} else if side == sideCT {
			sideStr = "CT"
		}

		rounds = append(rounds, &ClutchRound{
			RoundNo:    roundNo,
			PlayerName: playerName,
			Team:       teamName,
			Side:       sideStr,
			Opponents:  opponents,
			Won:        won,
		})
	}
	if err := rows.Err(); err != nil {
		log.Printf("Row iteration error for match %v: %v", matchId, err)
		return c.Status(500).JSON(fiber.Map{
			"message": "Error while reading clutch stats",
			"success": "false",
		})
	}

	playerResult := make([]*PlayerClutchStats, 0, len(order))
	for _, id := range order {
		p := players[id]
		if p.Attempts > 0 {
			p.WinPct = round2(float64(p.Wins) / float64(p.Attempts) * 100)
		}
		playerResult = append(playerResult, p)
	}

	// Always report all five situations per team, zero-filled, so the
	// frontend has a stable 1v1..1v5 axis to chart even when a situation
	// never came up.
	situationResult := make([]*TeamClutchSituation, 0, 10)
	for _, team := range []string{teamAName, teamBName} {
		for opponents := 1; opponents <= 5; opponents++ {
			s, ok := situations[clutchSituationKey(team, opponents)]
			if !ok {
				s = &TeamClutchSituation{Team: team, Opponents: opponents}
			}
			if s.Attempts > 0 {
				s.WinPct = round2(float64(s.Wins) / float64(s.Attempts) * 100)
			}
			situationResult = append(situationResult, s)
		}
	}

	return c.Status(200).JSON(fiber.Map{
		"players":    playerResult,
		"situations": situationResult,
		"rounds":     rounds,
	})
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
				var playername string
				var team sql.NullString
				var playerid int64
				if err := team_response.Scan(&playername, &playerid, &team); err != nil || !team.Valid {
					// Player has no resolved TEAMNAME yet (still-unassigned
					// solo player on a default-named side) - leave them out
					// of the team grouping rather than bucketing them under
					// an empty-string team key.
					continue
				}
				if _, ok := me.Teams[team.String]; !ok {
					me.Teams[team.String] = make(map[int64]string)
				}
				me.Teams[team.String][int64(playerid)] = playername
			}
			mapMeta, err := parser.ResolveMapMetadata(gamemap)
			if err != nil {
				log.Printf("Map metadata lookup failed for match %v (%v): %v", matchid, gamemap, err)
				return c.Status(500).JSON(fiber.Map{
					"message": fmt.Sprintf("No radar overview data available for map %q yet", gamemap),
					"success": "false",
				})
			}
			me.MapMeta = mapMeta

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
