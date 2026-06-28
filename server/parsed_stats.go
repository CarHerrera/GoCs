package main

import (
	"time"

	dem "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
)

type Team struct {
	ID             int              `json:"ID"`
	ClanName       string           `json:"Clanname"`
	EndScore       int              `json:"Endscore"`
	TScore         int              `json:"TScore"`
	CTScore        int              `json:"CTScore"`
	PlayingPlayers map[int64]Player `json:"Playing"`
	inited         bool
}
type Player struct {
	Name  string      `json:"name"`
	ID    int64       `json:"ID,string"`
	Stats PlayerStats `json:"stats"`
}
type PlayerStats struct {
	Kills          int `json:"kills"`
	HeadshotKills  int `json:"hs"`
	EntryKills     int `json:"entry_kills"`
	EntryDeaths    int `json:"entry_deaths"`
	Deaths         int `json:"deaths"`
	Assists        int `json:"assists"`
	Damage         int `json:"dmg"`
	UtilityDamage  int `json:"ud"`
	OneFragCount   int `json:"1k"`
	TwoFrags       int `json:"2k"`
	ThreeFrags     int `json:"3k"`
	FourFrags      int `json:"4k"`
	FiveFrags      int `json:"5k"`
	TradedDeaths   int `json:"traded_deaths"`
	TradeKills     int `json:"trade_kills"`
	ClutchesWon    int `json:"clutch_win"`
	ClutchCount    int `json:"clutch_count"`
	FlashAssists   int `json:"flash_assists"`
	TeamFlashed    int `json:"team_flashes"`
	EnemiesFlashed int `json:"enemies_flashed"`
	HEDamage       int `json:"he_damage"`
	FireDamage     int `json:"fire_damage"`
}

func setUpSideTracking(p dem.Parser, rt *RoundTracker) {
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
			rt.RoundCycle++ // track rounds per segment here
		}
	})
	p.RegisterEventHandler(func(events.RoundEndOfficial) {
		if *rt.Live {
			*rt.Rounds++ // track rounds per segment here
		}
	})
}

func setUpKillTracking(ps dem.Parser, rt *RoundTracker) {
	type RoundStats struct {
		k   int
		a   int
		dmg int
		fk  bool
		fd  bool
	}
	multiKills := make(map[uint64]RoundStats)
	ps.RegisterEventHandler(func(kill events.Kill) {
		if rt == nil || rt.Live == nil || !*rt.Live {
			return
		}
		killer := kill.Killer
		asssiter := kill.Assister
		victim := kill.Victim
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
			if mk, ok := multiKills[killer.SteamID64]; ok {
				if *rt.FirstKill {
					mk.fk = true
				}
				mk.k++
				multiKills[killer.SteamID64] = mk
			} else {
				fk := false
				if *rt.FirstKill {
					fk = true
				}
				multiKills[killer.SteamID64] = RoundStats{
					k: 1, fk: fk,
				}
			}
			rt.Teams[idx].PlayingPlayers[int64(killer.SteamID64)] = p

		}
		if asssiter != nil {
			team := asssiter.TeamState
			idx := -1
			if team.ID() == rt.Teams[0].ID {
				idx = 0
			} else {
				idx = 1
			}
			p, _ := rt.Teams[idx].PlayingPlayers[int64(asssiter.SteamID64)]
			p.Stats.Assists++
			rt.Teams[idx].PlayingPlayers[int64(asssiter.SteamID64)] = p
			if mk, ok := multiKills[asssiter.SteamID64]; ok {
				mk.a++
				multiKills[asssiter.SteamID64] = mk
			} else {
				fd := false
				multiKills[asssiter.SteamID64] = RoundStats{
					a: 1, fd: fd,
				}
			}
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
			if mk, ok := multiKills[victim.SteamID64]; ok {
				if *rt.FirstKill {
					mk.fd = true
				}
				multiKills[victim.SteamID64] = mk
			} else {
				fd := false
				if *rt.FirstKill {
					fd = true
				}
				multiKills[victim.SteamID64] = RoundStats{
					fd: fd,
				}
			}
			rt.Teams[idx].PlayingPlayers[int64(victim.SteamID64)] = p

		}

		if *rt.FirstKill {
			*rt.FirstKill = false
		}
	})
	ps.RegisterEventHandler(func(events.RoundEndOfficial) {
		if rt == nil || rt.Live == nil || !*rt.Live {
			return
		}
		gs := ps.GameState()
		DB.Exec(
			`UPDATE ROUND_PARTICIPANTS SET SURVIVED = 0 
         WHERE MATCHID = ? AND ROUND_NO = ?`,
			rt.Matchid, *rt.Rounds)
		for _, p := range gs.Participants().Playing() {
			team := p.TeamState
			idx := -1
			if team.ID() == rt.Teams[0].ID {
				idx = 0
			} else {
				idx = 1
			}
			x, _ := rt.Teams[idx].PlayingPlayers[int64(p.SteamID64)]
			if multiKills[p.SteamID64].k == 1 {
				x.Stats.OneFragCount++
			} else if multiKills[p.SteamID64].k == 2 {
				x.Stats.TwoFrags++
			} else if multiKills[p.SteamID64].k == 3 {
				x.Stats.ThreeFrags++
			} else if multiKills[p.SteamID64].k == 4 {
				x.Stats.FourFrags++
			} else if multiKills[p.SteamID64].k == 5 {
				x.Stats.FiveFrags++
			}
			gotKill := multiKills[p.SteamID64].k > 0
			DB.Exec(`
				UPDATE ROUND_PARTICIPANTS
					SET KILLS = 1, GOT_KILL = 1, FIRST_KILL = 1, SURVIVED = 0,
					GOT_ASSIST = 1, FIRST_DEATH = 1, DAMAGE = 1
					WHERE MATCHID = 5 AND ROUND_NO = 1 AND PLAYERID = 76561199472278431
				`, multiKills[p.SteamID64].k, gotKill, multiKills[p.SteamID64].fk,
				multiKills[p.SteamID64].a > 0, multiKills[p.SteamID64].fd, multiKills[p.SteamID64].dmg,
				rt.Matchid, *rt.Rounds, p.SteamID64)
			rt.Teams[idx].PlayingPlayers[int64(p.SteamID64)] = x
			if p.IsAlive() {
				_, err := DB.Exec(`
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
	})
	ps.RegisterEventHandler(func(events.RoundStart) {
		if rt == nil || rt.Live == nil || !*rt.Live {
			return
		}
		*rt.FirstKill = true

	})
	ps.RegisterEventHandler(func(ph events.PlayerHurt) {
		if rt == nil || rt.Live == nil || !*rt.Live {
			return
		}
		if ph.Attacker != nil {
			if mk, ok := multiKills[ph.Attacker.SteamID64]; ok {
				mk.dmg += ph.HealthDamageTaken
				multiKills[ph.Attacker.SteamID64] = mk
			} else {

				multiKills[ph.Attacker.SteamID64] = RoundStats{
					dmg: ph.HealthDamageTaken,
				}
			}
		}
	})
}
func setUpTradeTracking(p dem.Parser, setup *DemoSetup, rt *RoundTracker) {
	type recentKill struct {
		originalVictimID uint64
		tick             int
	}
	// key: killer's SteamID — "this person killed originalVictimID at this tick"
	recentKills := make(map[uint64]recentKill)

	p.RegisterEventHandler(func(events.RoundStart) {
		recentKills = make(map[uint64]recentKill)
	})

	p.RegisterEventHandler(func(kill events.Kill) {
		if !*rt.Live || kill.Killer == nil || kill.Victim == nil {
			return
		}
		tick := p.GameState().IngameTick()
		tradeWindow := int(5 * p.TickRate())

		// Was the victim a recent killer? If so, their earlier victim was traded
		if prev, ok := recentKills[kill.Victim.SteamID64]; ok {
			if tick-prev.tick <= tradeWindow {
				// Find the original victim and mark their death as traded
				for i := range setup.Teams {
					for name, player := range setup.Teams[i].PlayingPlayers {
						if uint64(player.ID) == prev.originalVictimID {
							player.Stats.TradedDeaths++
							DB.Exec(`UPDATE ROUND_PARTICIPANTS
							SET GOT_TRADED = 1
							WHERE MATCHID = ? AND ROUND_NO = ? AND PLAYERID = ?`, rt.Matchid, *rt.Rounds, player.ID)
							setup.Teams[i].PlayingPlayers[name] = player
							break
						}
					}
				}
				team := kill.Killer.TeamState
				idx := -1
				if team.ID() == rt.Teams[0].ID {
					idx = 0
				} else {
					idx = 1
				}
				player, _ := rt.Teams[idx].PlayingPlayers[int64(kill.Killer.SteamID64)]
				player.Stats.TradeKills++
				rt.Teams[idx].PlayingPlayers[int64(kill.Killer.SteamID64)] = player
			}
		}

		// Record this kill so it can be checked as a potential trade later
		recentKills[kill.Killer.SteamID64] = recentKill{
			originalVictimID: kill.Victim.SteamID64,
			tick:             tick,
		}
	})
}
func setUpClutchTracking(p dem.Parser, setup *DemoSetup, rt *RoundTracker) {
	type clutchState struct {
		playerID    uint64
		teamIdx     int
		winningSide common.Team // which side needs to win for it to be a clutch win
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
		// Once a clutch is active don't re-evaluate on further kills
		// e.g. player in a 1v3 killing one doesn't reset the clutch to 1v2
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

		// Check if either side now has exactly one player alive
		var clutchSide common.Team
		switch {
		case tAlive == 1 && ctAlive >= 1:
			clutchSide = common.TeamTerrorists
		case ctAlive == 1 && tAlive >= 1:
			clutchSide = common.TeamCounterTerrorists
		default:
			return
		}

		// Find the lone survivor and record the clutch
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
		// If the clutch player's side won, it counts as a clutch win
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
func setUpFlashTracking(p dem.Parser, setup *DemoSetup, rt *RoundTracker) {
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

		// Insert every flash immediately — LED_TO_KILL defaults to 0
		// and gets updated to 1 in the kill handler if it leads to a kill
		DB.Exec(`INSERT IGNORE INTO PLAYER_FLASHES
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

			// Update the existing row rather than inserting a duplicate
			DB.Exec(`UPDATE PLAYER_FLASHES SET LED_TO_KILL = 1
                WHERE MATCHID = ? AND ROUND_NO = ? AND THROWERID = ? AND VICTIMID = ? AND TICK = ?`,
				rt.Matchid, *rt.Rounds,
				flash.throwerID, kill.Victim.SteamID64,
				flash.tick,
			)
			break
		}
	})
}
func setUpGrenadeDamageTracking(p dem.Parser, setup *DemoSetup, rt *RoundTracker) {
	p.RegisterEventHandler(func(e events.PlayerHurt) {
		if !*rt.Live || e.Attacker == nil || e.Player == nil {
			return
		}
		// Skip self-damage and team damage
		if e.Attacker.SteamID64 == e.Player.SteamID64 {
			return
		}
		if e.Attacker.GetTeam() == e.Player.GetTeam() {
			return
		}
		if e.Weapon == nil {
			return
		}

		// HealthDamageTaken is capped at remaining HP so it never exceeds 100.
		// This matches how HLTV counts ADR — overkill damage is not counted.
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
func setUpDamageTracking(ps dem.Parser, rt *RoundTracker) {
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
