package parser

import (
	"os"
	database "server/DB"
	"server/auth"
	"server/model"

	dem "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/msg"
)

func newRoundTracker(setup *model.DemoSetup, rounds *int) *model.RoundTracker {
	return &model.RoundTracker{
		Teams:         &setup.Teams,
		Live:          &setup.Live,
		Catch:         true,
		Matchid:       setup.MatchId,
		Rounds:        rounds,
		FirstKill:     &setup.FirstKill,
		PlayerTeamIdx: make(map[int64]int),
	}
}
func setupDemoFile(fileName string) (*os.File, dem.Parser, error) {
	file, err := os.Open(auth.GetDemoPath() + fileName)

	if err != nil {
		return nil, nil, err
	}
	p := dem.NewParserWithConfig(file, dem.ParserConfig{
		MsgQueueBufferSize:        0,
		IgnorePacketEntitiesPanic: true,
	})

	return file, p, nil
}

func setupMap(p dem.Parser, setup *model.DemoSetup) {
	p.RegisterNetMessageHandler(func(msg *msg.CSVCMsg_ServerInfo) {
		setup.GameMap = *msg.MapName
	})
}
func ensurePlayerExists(PlayerID int64, name string) {
	database.DB.Exec("INSERT IGNORE INTO PLAYERS (PLAYERID,PLAYERNAME) VALUES (?,?)", PlayerID, name)
}

// getOrAddPlayer returns the tracked player for id, registering it (both in
// PLAYERS and in the roster map) if this is the first time it's been seen.
// Without this, a player who joins after the initial roster capture (e.g. a
// mid-match substitute) would be written back with a zero-value ID, which
// later violates the MATCH_STATS -> PLAYERS foreign key.
func getOrAddPlayer(players map[int64]model.Player, id int64, name string) model.Player {
	if p, ok := players[id]; ok {
		return p
	}
	ensurePlayerExists(id, name)
	p := model.Player{Name: name, ID: id}
	players[id] = p
	return p
}

// teamIdxForPlayer returns the roster index (0 or 1) a player was pinned to
// the first time they were classified, and reuses it from then on instead
// of re-deriving it from live TeamState every time. Roster membership never
// changes mid-match, but around the last round of a half a player's live
// TeamState (ClanName/ID) can already reflect the *next* half's side
// assignment before the round is officially over and before TeamSideSwitch
// fires - re-checking it on every event misclassifies players in that
// window. currentClanName is only consulted the first time a given
// steamID64 is seen (kickoff roster, or a genuine mid-match joiner).
func teamIdxForPlayer(rt *model.RoundTracker, steamID64 uint64, currentClanName string) int {
	id := int64(steamID64)
	if idx, ok := rt.PlayerTeamIdx[id]; ok {
		return idx
	}
	idx := 1
	if currentClanName == rt.Teams[0].ClanName {
		idx = 0
	}
	rt.PlayerTeamIdx[id] = idx
	return idx
}
func setupTeams(p dem.Parser, setup *model.DemoSetup, rt *model.RoundTracker) {
	GS := p.GameState()
	p.RegisterEventHandler(func(e events.MatchStartedChanged) error {
		if GS.GamePhase() != common.GamePhaseStartGamePhase {
			return nil
		}
		prev := rt.RoundCycle
		rt.RoundCycle = 0
		if prev > 1 {
			return nil
		}
		*rt.Rounds = 1
		setup.Live = true
		sides := []struct {
			state    *common.TeamState
			idx      int
			fallback string
		}{
			{GS.TeamTerrorists(), 0, "Team 1"},
			{GS.TeamCounterTerrorists(), 1, "Team 2"},
		}

		for _, side := range sides {
			teamName := side.state.ClanName()
			isDefault := teamName == ""
			if isDefault {
				teamName = side.fallback
			}
			setup.Teams[side.idx] = model.Team{
				ID:             side.state.ID(),
				EndScore:       -1,
				CTScore:        0,
				TScore:         0,
				ClanName:       teamName,
				PlayingPlayers: make(map[int64]model.Player),
				Inited:         true,
				IsDefaultName:  isDefault,
			}
			// MATCHES.TEAM_A_NAME/TEAM_B_NAME are written from ClanName
			// regardless of IsDefaultName (see ParseDemoStats) and FK-reference
			// TEAMS, so a row must exist here even for the "Team 1"/"Team 2"
			// placeholder — it's just a display label for the match overview.
			// IsDefaultName is what keeps a placeholder from ever being written
			// to MATCH_STATS.TEAMNAME as if it were a real roster team.
			database.DB.Exec("INSERT IGNORE INTO TEAMS (TEAMNAME) VALUES (?)", teamName)
			for _, player := range side.state.Members() {
				setup.Teams[side.idx].PlayingPlayers[int64(player.SteamID64)] = model.Player{
					Name: player.Name, ID: int64(player.SteamID64), Stats: model.PlayerStats{},
				}
				rt.PlayerTeamIdx[int64(player.SteamID64)] = side.idx
				ensurePlayerExists(int64(player.SteamID64), player.Name)
			}
		}

		return nil
	})
	p.RegisterEventHandler(func(score events.ScoreUpdated) {
		if !*rt.Live {
			return
		}
		team1 := score.TeamState
		if rt.Teams[0].Inited && rt.Catch {
			if rt.Teams[0].ID == team1.ID() {
				rt.Teams[0].EndScore = score.NewScore
				if team1.Team() == common.TeamCounterTerrorists {
					rt.Teams[0].CTScore += 1
				} else {
					rt.Teams[0].TScore += 1
				}
			} else {
				rt.Teams[1].EndScore = score.NewScore
				if team1.Team() == common.TeamCounterTerrorists {
					rt.Teams[1].CTScore++
				} else {
					rt.Teams[1].TScore++
				}
			}
		}
	})
}
func classifyBuy(roundNo int, avgEquipValue int) int {
	const (
		BuyTypePistol = 1
		BuyTypeEco    = 2
		BuyTypeForce  = 3
		BuyTypeFull   = 4
	)
	if roundNo == 1 || roundNo == 13 {
		return BuyTypePistol
	}
	switch {
	case avgEquipValue < 1500:
		return BuyTypeEco
	case avgEquipValue < 4000:
		return BuyTypeForce
	default:
		return BuyTypeFull
	}
}

// teamNameForID resolves a live TeamState's numeric ID back to the
// fallback-substituted ClanName captured in setupTeams, so ROUNDS.T_TEAM/
// CT_TEAM always agree with MATCHES.TEAM_A_NAME/TEAM_B_NAME — including for
// sides with no clan tag, which ClanName() reports as "" here but setupTeams
// already resolved to "Team 1"/"Team 2".
func teamNameForID(rt *model.RoundTracker, id int) string {
	for _, t := range rt.Teams {
		if t.ID == id {
			return t.ClanName
		}
	}
	return ""
}

func setupRoundInfo(p dem.Parser, rt *model.RoundTracker) {
	p.RegisterEventHandler(func(events.RoundFreezetimeEnd) {
		if !*rt.Live {
			return
		}

		gs := p.GameState()
		rt.FreezeEndTick = gs.IngameTick()
		database.DB.Exec("INSERT IGNORE INTO ROUNDS (ROUND_NO,MATCHID) VALUES (?,?)", *rt.Rounds, rt.Matchid)
		var tTotal, ctTotal, tCount, ctCount int
		tside := teamNameForID(rt, gs.TeamTerrorists().ID())
		ctside := teamNameForID(rt, gs.TeamCounterTerrorists().ID())
		for _, player := range gs.Participants().Playing() {
			switch player.GetTeam() {
			case common.TeamTerrorists:
				tTotal += player.EquipmentValueCurrent()
				tCount++
			case common.TeamCounterTerrorists:
				ctTotal += player.EquipmentValueCurrent()
				ctCount++
			}
			database.DB.Exec("INSERT IGNORE INTO ROUND_PARTICIPANTS (MATCHID,ROUND_NO,PLAYERID,SIDE) VALUES (?,?,?,?)", rt.Matchid, *rt.Rounds, int64(player.SteamID64), player.Team)
		}

		if tCount == 0 || ctCount == 0 {
			return
		}

		tAvg := tTotal / tCount
		ctAvg := ctTotal / ctCount

		tBuy := classifyBuy(*rt.Rounds, tAvg)
		ctBuy := classifyBuy(*rt.Rounds, ctAvg)

		database.DB.Exec(
			`UPDATE ROUNDS SET BUY_TYPE_T = ?, BUY_TYPE_CT = ?, CT_TEAM = ?, T_TEAM = ?
			WHERE MATCHID = ? AND ROUND_NO = ?`,
			tBuy, ctBuy, ctside, tside, rt.Matchid, *rt.Rounds,
		)
	})
	p.RegisterEventHandler(func(e events.RoundEnd) {
		if !*rt.Live {
			return
		}
		database.DB.Exec(`UPDATE ROUNDS SET WINNING_SIDE = ?, WIN_REASON =?
		WHERE ROUND_NO = ? AND MATCHID = ?`,
			e.Winner, e.Reason, rt.Rounds, rt.Matchid)

	})
}
