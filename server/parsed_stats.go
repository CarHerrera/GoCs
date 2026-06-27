package main

import (
	dem "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
)

func setupSideTracking(p dem.Parser, rt *RoundTracker) {
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

func setupKillTracking(ps dem.Parser, rt *RoundTracker) {

	ps.RegisterEventHandler(func(kill events.Kill) {
		if rt == nil || rt.Live == nil || !*rt.Live {
			return
		}
		killer := kill.Killer
		asssiter := kill.Assister
		victim := kill.Victim
		if killer != nil && killer.Name != victim.Name {
			team := killer.TeamState
			if team.ID() == rt.Teams[0].ID {
				p, _ := rt.Teams[0].PlayingPlayers[int64(killer.SteamID64)]
				p.Stats.Kills++
				rt.Teams[0].PlayingPlayers[int64(killer.SteamID64)] = p
			} else {
				p, _ := rt.Teams[1].PlayingPlayers[int64(killer.SteamID64)]
				p.Stats.Kills++
				rt.Teams[1].PlayingPlayers[int64(killer.SteamID64)] = p
			}
		}
		if asssiter != nil {
			team := asssiter.TeamState
			if team.ID() == rt.Teams[0].ID {
				p, _ := rt.Teams[0].PlayingPlayers[int64(asssiter.SteamID64)]
				p.Stats.Assists++
				rt.Teams[0].PlayingPlayers[int64(asssiter.SteamID64)] = p
			} else {
				p, _ := rt.Teams[1].PlayingPlayers[int64(asssiter.SteamID64)]
				p.Stats.Assists++
				rt.Teams[1].PlayingPlayers[int64(asssiter.SteamID64)] = p
			}
		}
		if victim != nil {
			team := victim.TeamState
			if team.ID() == rt.Teams[0].ID {
				p, _ := rt.Teams[0].PlayingPlayers[int64(victim.SteamID64)]
				p.Stats.Deaths++
				rt.Teams[0].PlayingPlayers[int64(victim.SteamID64)] = p
			} else {
				p, _ := rt.Teams[1].PlayingPlayers[int64(victim.SteamID64)]
				p.Stats.Deaths++
				rt.Teams[1].PlayingPlayers[int64(victim.SteamID64)] = p
			}
		}
	})
}
