package main

import (
	"os"

	ex "github.com/markus-wa/demoinfocs-golang/v5/examples"
	dem "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/msg"
)

type DemoSetup struct {
	MatchId   int
	GameMap   string
	Teams     [2]Team
	Live      bool
	FirstKill bool
}
type MatchEvents struct {
	RoundPositions RoundInfo                   `json:"round_events"`
	Rounds         int                         `json:"rounds"`
	MapMeta        ex.Map                      `json:"map"`
	Teams          map[string]map[int64]string `json:"teams"`
}
type RoundTracker struct {
	Teams      *[2]Team
	Live       *bool
	FirstKill  *bool
	LRTH       bool
	Catch      bool
	Matchid    int
	Rounds     *int
	RoundCycle int
}

func newRoundTracker(setup *DemoSetup, rounds *int) *RoundTracker {
	return &RoundTracker{
		Teams:     &setup.Teams,
		Live:      &setup.Live,
		Catch:     true,
		Matchid:   setup.MatchId,
		Rounds:    rounds,
		FirstKill: &setup.FirstKill,
	}
}
func setupDemoFile(fileName string) (*os.File, dem.Parser, error) {
	file, err := os.Open(getDemoPath() + fileName)

	if err != nil {
		return nil, nil, err
	}
	p := dem.NewParserWithConfig(file, dem.ParserConfig{
		MsgQueueBufferSize:        0,
		IgnorePacketEntitiesPanic: true,
	})

	return file, p, nil
}

func setupMap(p dem.Parser, setup *DemoSetup) {
	p.RegisterNetMessageHandler(func(msg *msg.CSVCMsg_ServerInfo) {
		setup.GameMap = *msg.MapName
	})
}
func ensurePlayerExists(PlayerID int64, name string) {
	DB.Exec("INSERT IGNORE INTO PLAYERS (PLAYERID,PLAYERNAME) VALUES (?,?)", PlayerID, name)
}
func setupTeams(p dem.Parser, setup *DemoSetup, rt *RoundTracker) {
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
			setup.Teams[side.idx] = Team{
				ID:             side.state.ID(),
				EndScore:       -1,
				CTScore:        0,
				TScore:         0,
				ClanName:       teamName,
				PlayingPlayers: make(map[int64]Player),
				inited:         true,
			}
			DB.Exec("INSERT IGNORE INTO TEAMS (TEAMNAME) VALUES (?)", teamName)
			for _, player := range side.state.Members() {
				setup.Teams[side.idx].PlayingPlayers[int64(player.SteamID64)] = Player{
					Name: player.Name, ID: int64(player.SteamID64), Stats: PlayerStats{},
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
		// Check to make sure it isn't null
		if rt.Teams[0].inited && rt.Catch {
			// team1 (non opp) will always have the score incremented
			//

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
			//
			//
		}
	})
}
func classifyBuy(roundNo int, avgEquipValue int) int {
	const (
		BuyTypePistol = 1 // round 1 or 13, special case regardless of money
		BuyTypeEco    = 2 // avg equipment value < 1000 — default pistol only
		BuyTypeForce  = 3 // avg 1000–3500 — upgraded pistols, SMGs, some armor
		BuyTypeFull   = 4 // avg > 3500 — rifles + armor + utility
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

func setupRoundInfo(p dem.Parser, rt *RoundTracker) {
	p.RegisterEventHandler(func(events.RoundFreezetimeEnd) {
		if !*rt.Live {
			return
		}

		gs := p.GameState()
		DB.Exec("INSERT IGNORE INTO ROUNDS (ROUND_NO,MATCHID) VALUES (?,?)", *rt.Rounds, rt.Matchid)
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
			DB.Exec("INSERT IGNORE INTO ROUND_PARTICIPANTS (MATCHID,ROUND_NO,PLAYERID,SIDE) VALUES (?,?,?,?)", rt.Matchid, *rt.Rounds, int64(player.SteamID64), player.Team)
		}

		if tCount == 0 || ctCount == 0 {
			return
		}

		tAvg := tTotal / tCount
		ctAvg := ctTotal / ctCount

		tBuy := classifyBuy(*rt.Rounds, tAvg)
		ctBuy := classifyBuy(*rt.Rounds, ctAvg)

		DB.Exec(
			`UPDATE ROUNDS SET BUY_TYPE_T = ?, BUY_TYPE_CT = ?, CT_TEAM = ?, T_TEAM = ?
			WHERE MATCHID = ? AND ROUND_NO = ?`,
			tBuy, ctBuy, tside, ctside, rt.Matchid, *rt.Rounds,
		)
	})
	p.RegisterEventHandler(func(bp events.BombPlanted) {
		if !*rt.Live {
			return
		}
		DB.Exec(`UPDATE ROUNDS SET BOMB_PLANT = ?
		WHERE ROUND_NO = ? AND MATCHID = ?`, true, rt.Rounds, rt.Matchid)
	})
	p.RegisterEventHandler(func(e events.RoundEnd) {
		if !*rt.Live {
			return
		}
		DB.Exec(`UPDATE ROUNDS SET WINNING_SIDE = ?, WIN_REASON =?
		WHERE ROUND_NO = ? AND MATCHID = ?`,
			e.Winner, e.Reason, rt.Rounds, rt.Matchid)

	})
}
