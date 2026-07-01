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
		Teams:     &setup.Teams,
		Live:      &setup.Live,
		Catch:     true,
		Matchid:   setup.MatchId,
		Rounds:    rounds,
		FirstKill: &setup.FirstKill,
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
			if teamName == "" {
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
			}
			database.DB.Exec("INSERT IGNORE INTO TEAMS (TEAMNAME) VALUES (?)", teamName)
			for _, player := range side.state.Members() {
				setup.Teams[side.idx].PlayingPlayers[int64(player.SteamID64)] = model.Player{
					Name: player.Name, ID: int64(player.SteamID64), Stats: model.PlayerStats{},
				}
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
	case avgEquipValue < 1000:
		return BuyTypeEco
	case avgEquipValue < 3500:
		return BuyTypeForce
	default:
		return BuyTypeFull
	}
}

func setupRoundInfo(p dem.Parser, rt *model.RoundTracker) {
	p.RegisterEventHandler(func(events.RoundFreezetimeEnd) {
		if !*rt.Live {
			return
		}

		gs := p.GameState()
		database.DB.Exec("INSERT IGNORE INTO ROUNDS (ROUND_NO,MATCHID) VALUES (?,?)", *rt.Rounds, rt.Matchid)
		var tTotal, ctTotal, tCount, ctCount int
		tside := gs.TeamTerrorists().ClanName()
		ctside := gs.TeamCounterTerrorists().ClanName()
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
			tBuy, ctBuy, tside, ctside, rt.Matchid, *rt.Rounds,
		)
	})
	p.RegisterEventHandler(func(bp events.BombPlanted) {
		if !*rt.Live {
			return
		}
		database.DB.Exec(`UPDATE ROUNDS SET BOMB_PLANT = ?
		WHERE ROUND_NO = ? AND MATCHID = ?`, true, rt.Rounds, rt.Matchid)
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
