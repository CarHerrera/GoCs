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
		// GS.TeamTerrorists()/TeamCounterTerrorists() are fixed slots
		// (team-num 2 and 3); TeamState.ID() reflects whichever clan
		// currently occupies that slot, not a persistent per-clan identity.
		// rt.Teams[0]/[1] track "the team that started as T"/"...as CT", so
		// their recorded ID has to follow them across the side swap. This is
		// only used for team-level score bookkeeping now - player-level
		// classification uses the stable rt.PlayerTeamIdx map instead, since
		// this swap and the live entity reassignment aren't perfectly
		// synchronized at the round boundary.
		rt.Teams[1].ID, rt.Teams[0].ID = rt.Teams[0].ID, rt.Teams[1].ID
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
func setUpBombPlants(ps dem.Parser, setup *model.DemoSetup, rt *model.RoundTracker) {
	ps.RegisterEventHandler(func(e events.BombExplode) {
		if !*rt.Live {
			return
		}
		database.DB.Exec(`UPDATE ROUNDS SET BOMB_EXPLODE = ?
		WHERE ROUND_NO = ? AND MATCHID = ?`, true, rt.Rounds, rt.Matchid)
	})
	ps.RegisterEventHandler(func(bp events.BombPlanted) {
		if !*rt.Live {
			return
		}
		var planterID *int64
		if bp.Player != nil {
			id := int64(bp.Player.SteamID64)
			planterID = &id
		}
		database.DB.Exec(`UPDATE ROUNDS SET BOMB_PLANT = ?, SITE = ?, PLANTER_PLAYERID = ?
		WHERE ROUND_NO = ? AND MATCHID = ?`, true, bombsiteLabel(bp.Site), planterID, rt.Rounds, rt.Matchid)
	})
	ps.RegisterEventHandler(func(bd events.BombDefused) {
		if !*rt.Live {
			return
		}
		var defuserID *int64
		if bd.Player != nil {
			id := int64(bd.Player.SteamID64)
			defuserID = &id
		}
		database.DB.Exec(`UPDATE ROUNDS SET BOMB_DEFUSE = ?, DEFUSER_PLAYERID = ?
		WHERE ROUND_NO = ? AND MATCHID = ?`, true, defuserID, rt.Rounds, rt.Matchid)
	})

}

func bombsiteLabel(site events.Bombsite) *string {
	switch site {
	case events.BombsiteA:
		s := "A"
		return &s
	case events.BombsiteB:
		s := "B"
		return &s
	default:
		return nil
	}
}

// tradeOpportunityDistance is an approximate "close enough to have a shot at
// the trade" radius in engine units. It's a heuristic, not a precise game
// mechanic, tune accordingly
const tradeOpportunityDistance = 1000.0

// hasNearbyLivingTeammate reports whether any of the victim's teammates
// (other than the victim) were alive within tradeOpportunityDistance at the
// moment of death, i.e. whether a trade was physically possible.
func hasNearbyLivingTeammate(victim *common.Player, team *common.TeamState) bool {
	for _, teammate := range team.Members() {
		if teammate == nil || teammate.SteamID64 == victim.SteamID64 {
			continue
		}
		if !teammate.IsAlive() {
			continue
		}
		if teammate.Position().Distance(victim.Position()) <= tradeOpportunityDistance {
			return true
		}
	}
	return false
}

func setUpKillTracking(ps dem.Parser, setup *model.DemoSetup, rt *model.RoundTracker) {
	type RoundStats struct {
		k                int
		a                int
		dmg              int
		fk               bool
		fd               bool
		traded           bool
		tradeKill        bool
		died             bool
		tradeOpportunity bool
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
		// Five seconds
		tradeWindow := int(5 * ps.TickRate())
		// Might as well settle both killer and victim here
		if killer != nil && victim != nil && killer.Name != victim.Name {
			killertTeam := killer.TeamState
			victimTeam := victim.TeamState
			killerIdx := teamIdxForPlayer(rt, killer.SteamID64, killertTeam.ClanName())
			victimIdx := teamIdxForPlayer(rt, victim.SteamID64, victimTeam.ClanName())
			k := getOrAddPlayer(rt.Teams[killerIdx].PlayingPlayers, int64(killer.SteamID64), killer.Name)
			v := getOrAddPlayer(rt.Teams[victimIdx].PlayingPlayers, int64(victim.SteamID64), victim.Name)
			mkk, ok := multiKills[killer.SteamID64]
			if !ok {
				mkk = RoundStats{}
			}
			mkv, ok := multiKills[victim.SteamID64]
			if !ok {
				mkv = RoundStats{}
			}
			k.Stats.Kills++
			v.Stats.Deaths++
			mkk.k++
			mkv.died = true
			mkk.died = false
			mkv.tradeOpportunity = hasNearbyLivingTeammate(victim, victimTeam)
			if kill.IsHeadshot {
				k.Stats.HeadshotKills++
			}
			if *rt.FirstKill {
				k.Stats.EntryKills++
				v.Stats.EntryDeaths++
				mkk.fk = true
				mkv.fd = true

				elapsedSeconds := float64(tick-rt.FreezeEndTick) / ps.TickRate()
				database.DB.Exec(`UPDATE ROUNDS SET OPENING_KILL_SECONDS = ?
					WHERE MATCHID = ? AND ROUND_NO = ?`, elapsedSeconds, rt.Matchid, *rt.Rounds)
			}
			multiKills[killer.SteamID64] = mkk
			multiKills[victim.SteamID64] = mkv
			rt.Teams[killerIdx].PlayingPlayers[int64(killer.SteamID64)] = k
			rt.Teams[victimIdx].PlayingPlayers[int64(victim.SteamID64)] = v
			// check if victim recently got a kill
			if prev, ok := recentKills[victim.SteamID64]; ok {
				// Check if difference is less than 5 seconds
				if tick-prev.tick <= tradeWindow {
					origMk, ok := multiKills[prev.originalVictimID]
					if !ok {
						origMk = RoundStats{}
					}
					origMk.traded = true
					multiKills[prev.originalVictimID] = origMk
					// Find player and then adjust the counter accordingly
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
					k.Stats.TradeKills++
					rt.Teams[killerIdx].PlayingPlayers[int64(killer.SteamID64)] = k
				}
			}
			// add killer to recent kill list for this round
			recentKills[killer.SteamID64] = recentKill{
				originalVictimID: victim.SteamID64,
				tick:             tick,
			}
		}

		if assister != nil {
			idx := teamIdxForPlayer(rt, assister.SteamID64, assister.TeamState.ClanName())
			p := getOrAddPlayer(rt.Teams[idx].PlayingPlayers, int64(assister.SteamID64), assister.Name)
			p.Stats.Assists++
			rt.Teams[idx].PlayingPlayers[int64(assister.SteamID64)] = p

			mk, ok := multiKills[assister.SteamID64]
			if !ok {
				mk = RoundStats{}
			}
			mk.a++
			multiKills[assister.SteamID64] = mk
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

		for _, p := range gs.Participants().Playing() {
			idx := teamIdxForPlayer(rt, p.SteamID64, p.TeamState.ClanName())
			x := getOrAddPlayer(rt.Teams[idx].PlayingPlayers, int64(p.SteamID64), p.Name)
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
			survived := !stats.died
			_, err := database.DB.Exec(`
				UPDATE ROUND_PARTICIPANTS
					SET KILLS = ?, GOT_KILL = ?, FIRST_KILL = ?,
					    GOT_ASSIST = ?, FIRST_DEATH = ?, DAMAGE = ?,
					    GOT_TRADED = ?, TRADE_OPPORTUNITY = ?, SURVIVED = ?
					WHERE MATCHID = ? AND ROUND_NO = ? AND PLAYERID = ?
				`, stats.k, gotKill, stats.fk,
				gotAssist, stats.fd, stats.dmg,
				stats.traded, stats.tradeOpportunity, survived,
				rt.Matchid, *rt.Rounds, p.SteamID64)
			if err != nil {
				panic(err)
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

		opponents := ctAlive
		if clutchSide == common.TeamCounterTerrorists {
			opponents = tAlive
		}

		for _, player := range gs.Participants().Playing() {
			if !player.IsAlive() || player.GetTeam() != clutchSide {
				continue
			}
			idx := teamIdxForPlayer(rt, player.SteamID64, player.TeamState.ClanName())
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

			database.DB.Exec(`INSERT IGNORE INTO CLUTCHES
				(MATCHID, ROUND_NO, PLAYERID, SIDE, OPPONENTS, WON)
				VALUES (?,?,?,?,?,0)`,
				rt.Matchid, *rt.Rounds, player.SteamID64, int(clutchSide), opponents)
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
			database.DB.Exec(`UPDATE CLUTCHES SET WON = 1
				WHERE MATCHID = ? AND ROUND_NO = ? AND PLAYERID = ?`,
				rt.Matchid, *rt.Rounds, current.playerID)
		}
		// Don't clear `current` here: a player can still register a kill in
		// the brief dead-time between RoundEnd and the next RoundStart (the
		// victim's IsAlive() hasn't caught up to the kill yet at that exact
		// event), which would otherwise be misread as a fresh 1-vs-1 and
		// spuriously start a second, bogus clutch for that same round.
		// RoundStart is the only thing that should re-arm detection.
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
			idx := teamIdxForPlayer(rt, ph.Attacker.SteamID64, ph.Attacker.TeamState.ClanName())
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
