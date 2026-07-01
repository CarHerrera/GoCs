package parser

import (
	database "server/DB"
	"server/model"
	"time"

	dem "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
)

func setUpSideTracking(p dem.Parser, rt *model.RoundTracker) {
	p.RegisterEventHandler(func(events.TeamSideSwitch) {
		if !*rt.Live {
			return
		}
		rt.LRTH = false
		temp := rt.Teams[1].ID
		rt.Teams[1].ID = rt.Teams[0].ID
		rt.Teams[0].ID = temp
		rt.Catch = true
	})
	p.RegisterEventHandler(func(events.AnnouncementLastRoundHalf) {
		rt.LRTH = true
	})
	p.RegisterEventHandler(func(events.RoundEnd) {
		if rt.LRTH {
			rt.Catch = false
		}
		if *rt.Live {
			rt.RoundCycle++
		}
	})
	p.RegisterEventHandler(func(events.RoundEndOfficial) {
		if *rt.Live {
			*rt.Rounds++
		}
	})
}

func setUpKillTracking(ps dem.Parser, setup *model.DemoSetup, rt *model.RoundTracker) {
	type RoundStats struct {
		k         int
		a         int
		dmg       int
		fk        bool
		fd        bool
		traded    bool
		tradeKill bool
	}
	type recentKill struct {
		originalVictimID uint64
		tick             int
	}

	multiKills := make(map[uint64]RoundStats)
	recentKills := make(map[uint64]recentKill)

	ps.RegisterEventHandler(func(events.RoundStart) {
		if rt == nil || rt.Live == nil || !*rt.Live {
			return
		}
		*rt.FirstKill = true
		multiKills = make(map[uint64]RoundStats)
		recentKills = make(map[uint64]recentKill)
	})

	ps.RegisterEventHandler(func(kill events.Kill) {
		if rt == nil || rt.Live == nil || !*rt.Live {
			return
		}
		killer := kill.Killer
		assister := kill.Assister
		victim := kill.Victim
		tick := ps.GameState().IngameTick()
		tradeWindow := int(5 * ps.TickRate())

		if killer != nil && victim != nil && killer.Name != victim.Name {
			team := killer.TeamState
			idx := -1
			if team.ID() == rt.Teams[0].ID {
				idx = 0
			} else {
				idx = 1
			}
			p, _ := rt.Teams[idx].PlayingPlayers[int64(killer.SteamID64)]
			p.Stats.Kills++
			if kill.IsHeadshot {
				p.Stats.HeadshotKills++
			}
			if *rt.FirstKill {
				p.Stats.EntryKills++
			}

			mk, ok := multiKills[killer.SteamID64]
			if !ok {
				mk = RoundStats{}
			}
			mk.k++
			if *rt.FirstKill {
				mk.fk = true
			}
			multiKills[killer.SteamID64] = mk
			rt.Teams[idx].PlayingPlayers[int64(killer.SteamID64)] = p

			if prev, ok := recentKills[victim.SteamID64]; ok {
				if tick-prev.tick <= tradeWindow {
					origMk, ok := multiKills[prev.originalVictimID]
					if !ok {
						origMk = RoundStats{}
					}
					origMk.traded = true
					multiKills[prev.originalVictimID] = origMk

					for i := range setup.Teams {
						for name, player := range setup.Teams[i].PlayingPlayers {
							if uint64(player.ID) == prev.originalVictimID {
								player.Stats.TradedDeaths++
								setup.Teams[i].PlayingPlayers[name] = player
								break
							}
						}
					}

					mk2 := multiKills[killer.SteamID64]
					mk2.tradeKill = true
					multiKills[killer.SteamID64] = mk2
					p.Stats.TradeKills++
					rt.Teams[idx].PlayingPlayers[int64(killer.SteamID64)] = p
				}
			}

			recentKills[killer.SteamID64] = recentKill{
				originalVictimID: victim.SteamID64,
				tick:             tick,
			}
		}

		if assister != nil {
			team := assister.TeamState
			idx := -1
			if team.ID() == rt.Teams[0].ID {
				idx = 0
			} else {
				idx = 1
			}
			p, _ := rt.Teams[idx].PlayingPlayers[int64(assister.SteamID64)]
			p.Stats.Assists++
			rt.Teams[idx].PlayingPlayers[int64(assister.SteamID64)] = p

			mk, ok := multiKills[assister.SteamID64]
			if !ok {
				mk = RoundStats{}
			}
			mk.a++
			multiKills[assister.SteamID64] = mk
		}

		if victim != nil {
			team := victim.TeamState
			idx := -1
			if team.ID() == rt.Teams[0].ID {
				idx = 0
			} else {
				idx = 1
			}
			p, _ := rt.Teams[idx].PlayingPlayers[int64(victim.SteamID64)]
			p.Stats.Deaths++
			if *rt.FirstKill {
				p.Stats.EntryDeaths++
			}

			mk, ok := multiKills[victim.SteamID64]
			if !ok {
				mk = RoundStats{}
			}
			if *rt.FirstKill {
				mk.fd = true
			}
			multiKills[victim.SteamID64] = mk
			rt.Teams[idx].PlayingPlayers[int64(victim.SteamID64)] = p
		}

		if *rt.FirstKill {
			*rt.FirstKill = false
		}
	})

	ps.RegisterEventHandler(func(ph events.PlayerHurt) {
		if rt == nil || rt.Live == nil || !*rt.Live {
			return
		}
		if ph.Attacker != nil {
			mk, ok := multiKills[ph.Attacker.SteamID64]
			if !ok {
				mk = RoundStats{}
			}
			mk.dmg += ph.HealthDamageTaken
			multiKills[ph.Attacker.SteamID64] = mk
		}
	})

	ps.RegisterEventHandler(func(events.RoundEndOfficial) {
		if rt == nil || rt.Live == nil || !*rt.Live {
			return
		}
		gs := ps.GameState()

		_, err := database.DB.Exec(
			`UPDATE ROUND_PARTICIPANTS SET SURVIVED = 0
			 WHERE MATCHID = ? AND ROUND_NO = ?`,
			rt.Matchid, *rt.Rounds)
		if err != nil {
			panic(err)
		}

		for _, p := range gs.Participants().Playing() {
			team := p.TeamState
			idx := -1
			if team.ID() == rt.Teams[0].ID {
				idx = 0
			} else {
				idx = 1
			}
			x, _ := rt.Teams[idx].PlayingPlayers[int64(p.SteamID64)]
			stats := multiKills[p.SteamID64]

			switch stats.k {
			case 1:
				x.Stats.OneFragCount++
			case 2:
				x.Stats.TwoFrags++
			case 3:
				x.Stats.ThreeFrags++
			case 4:
				x.Stats.FourFrags++
			case 5:
				x.Stats.FiveFrags++
			}
			rt.Teams[idx].PlayingPlayers[int64(p.SteamID64)] = x

			gotKill := stats.k > 0
			gotAssist := stats.a > 0

			_, err := database.DB.Exec(`
				UPDATE ROUND_PARTICIPANTS
					SET KILLS = ?, GOT_KILL = ?, FIRST_KILL = ?,
					    GOT_ASSIST = ?, FIRST_DEATH = ?, DAMAGE = ?,
					    GOT_TRADED = ?
					WHERE MATCHID = ? AND ROUND_NO = ? AND PLAYERID = ?
				`, stats.k, gotKill, stats.fk,
				gotAssist, stats.fd, stats.dmg,
				stats.traded,
				rt.Matchid, *rt.Rounds, p.SteamID64)
			if err != nil {
				panic(err)
			}

			if p.IsAlive() {
				_, err := database.DB.Exec(`
					UPDATE ROUND_PARTICIPANTS
						SET SURVIVED = 1
						WHERE MATCHID = ? AND ROUND_NO = ? AND PLAYERID = ?
					`, rt.Matchid, *rt.Rounds, p.SteamID64)
				if err != nil {
					panic(err)
				}
			}
		}

		multiKills = make(map[uint64]RoundStats)
		recentKills = make(map[uint64]recentKill)
	})
}
func setUpClutchTracking(p dem.Parser, setup *model.DemoSetup, rt *model.RoundTracker) {
	type clutchState struct {
		playerID    uint64
		teamIdx     int
		winningSide common.Team
		active      bool
	}
	var current clutchState

	p.RegisterEventHandler(func(events.RoundStart) {
		current = clutchState{}
	})

	p.RegisterEventHandler(func(kill events.Kill) {
		if !*rt.Live || kill.Victim == nil {
			return
		}
		if current.active {
			return
		}
		gs := p.GameState()
		var tAlive, ctAlive int
		for _, player := range gs.Participants().Playing() {
			if !player.IsAlive() {
				continue
			}
			if player.GetTeam() == common.TeamTerrorists {
				tAlive++
			} else {
				ctAlive++
			}
		}

		var clutchSide common.Team
		switch {
		case tAlive == 1 && ctAlive >= 1:
			clutchSide = common.TeamTerrorists
		case ctAlive == 1 && tAlive >= 1:
			clutchSide = common.TeamCounterTerrorists
		default:
			return
		}

		for _, player := range gs.Participants().Playing() {
			if !player.IsAlive() || player.GetTeam() != clutchSide {
				continue
			}
			idx := 0
			if player.TeamState.ID() != rt.Teams[0].ID {
				idx = 1
			}
			pl, ok := setup.Teams[idx].PlayingPlayers[int64(player.SteamID64)]
			if !ok {
				break
			}
			pl.Stats.ClutchCount++
			setup.Teams[idx].PlayingPlayers[int64(player.SteamID64)] = pl

			current = clutchState{
				playerID:    player.SteamID64,
				teamIdx:     idx,
				winningSide: clutchSide,
				active:      true,
			}
			break
		}
	})

	p.RegisterEventHandler(func(e events.RoundEnd) {
		if !*rt.Live || !current.active {
			return
		}
		if e.Winner == current.winningSide {
			pl, ok := setup.Teams[current.teamIdx].PlayingPlayers[int64(current.playerID)]
			if ok {
				pl.Stats.ClutchesWon++
				setup.Teams[current.teamIdx].PlayingPlayers[int64(current.playerID)] = pl
			}
		}
		current = clutchState{}
	})
}
func setUpFlashTracking(p dem.Parser, setup *model.DemoSetup, rt *model.RoundTracker) {
	type flashRecord struct {
		throwerID uint64
		tick      int
		duration  time.Duration
	}
	recentFlashes := make(map[uint64]flashRecord)

	p.RegisterEventHandler(func(events.RoundStart) {
		recentFlashes = make(map[uint64]flashRecord)
	})

	p.RegisterEventHandler(func(e events.PlayerFlashed) {
		if !*rt.Live || e.Attacker == nil || e.Player == nil {
			return
		}
		isTeamFlash := e.Attacker.GetTeam() == e.Player.GetTeam()

		for i := range setup.Teams {
			pl, ok := setup.Teams[i].PlayingPlayers[int64(e.Attacker.SteamID64)]
			if !ok {
				continue
			}
			if isTeamFlash {
				pl.Stats.TeamFlashed++
			} else {
				pl.Stats.EnemiesFlashed++
				recentFlashes[e.Player.SteamID64] = flashRecord{
					throwerID: e.Attacker.SteamID64,
					tick:      p.GameState().IngameTick(),
					duration:  e.FlashDuration(),
				}
			}
			setup.Teams[i].PlayingPlayers[int64(e.Attacker.SteamID64)] = pl
			break
		}

		database.DB.Exec(`INSERT IGNORE INTO PLAYER_FLASHES
            (MATCHID, ROUND_NO, THROWERID, VICTIMID, TICK, BLIND_DUR, IS_TEAM_FLASH, LED_TO_KILL)
            VALUES (?,?,?,?,?,?,?,0)`,
			rt.Matchid, *rt.Rounds,
			e.Attacker.SteamID64, e.Player.SteamID64,
			p.GameState().IngameTick(),
			e.FlashDuration().Seconds(),
			isTeamFlash,
		)
	})

	p.RegisterEventHandler(func(kill events.Kill) {
		if !*rt.Live || kill.Victim == nil {
			return
		}
		flash, ok := recentFlashes[kill.Victim.SteamID64]
		if !ok {
			return
		}
		if p.GameState().IngameTick()-flash.tick > int(4*p.TickRate()) {
			return
		}
		for i := range setup.Teams {
			pl, ok := setup.Teams[i].PlayingPlayers[int64(flash.throwerID)]
			if !ok {
				continue
			}
			pl.Stats.FlashAssists++
			setup.Teams[i].PlayingPlayers[int64(flash.throwerID)] = pl

			database.DB.Exec(`UPDATE PLAYER_FLASHES SET LED_TO_KILL = 1
                WHERE MATCHID = ? AND ROUND_NO = ? AND THROWERID = ? AND VICTIMID = ? AND TICK = ?`,
				rt.Matchid, *rt.Rounds,
				flash.throwerID, kill.Victim.SteamID64,
				flash.tick,
			)
			break
		}
	})
}
func setUpGrenadeDamageTracking(p dem.Parser, setup *model.DemoSetup, rt *model.RoundTracker) {
	p.RegisterEventHandler(func(e events.PlayerHurt) {
		if !*rt.Live || e.Attacker == nil || e.Player == nil {
			return
		}
		if e.Attacker.SteamID64 == e.Player.SteamID64 {
			return
		}
		if e.Attacker.GetTeam() == e.Player.GetTeam() {
			return
		}
		if e.Weapon == nil {
			return
		}

		damage := e.HealthDamageTaken

		for i := range setup.Teams {
			pl, ok := setup.Teams[i].PlayingPlayers[int64(e.Attacker.SteamID64)]
			if !ok {
				continue
			}
			switch e.Weapon.Type {
			case common.EqHE:
				pl.Stats.HEDamage += damage
			case common.EqMolotov, common.EqIncendiary:
				pl.Stats.FireDamage += damage
			}
			setup.Teams[i].PlayingPlayers[int64(e.Attacker.SteamID64)] = pl
			break
		}
	})
}
func setUpDamageTracking(ps dem.Parser, rt *model.RoundTracker) {
	ps.RegisterEventHandler(func(ph events.PlayerHurt) {
		if rt == nil || rt.Live == nil || !*rt.Live {
			return
		}
		if ph.Attacker != nil {
			idx := -1
			if ph.Attacker.TeamState.ID() == rt.Teams[0].ID {
				idx = 0
			} else {
				idx = 1
			}
			p, ok := rt.Teams[idx].PlayingPlayers[int64(ph.Attacker.SteamID64)]
			if !ok {
				return
			}
			p.Stats.Damage += ph.HealthDamageTaken
			if ph.Weapon != nil && ph.Weapon.Class() == common.EqClassGrenade {
				p.Stats.UtilityDamage += ph.HealthDamageTaken
			}
			rt.Teams[idx].PlayingPlayers[int64(ph.Attacker.SteamID64)] = p

		}
	})
}
