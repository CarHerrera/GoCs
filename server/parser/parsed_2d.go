package parser

import (
	"database/sql"
	"fmt"
	"server/DB"
	"server/model"
	"slices"

	"github.com/golang/geo/r2"
	"github.com/golang/geo/r3"
	dem "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
)

func setUpRoundCycle(p dem.Parser, rt *model.RoundTracker, pe *[]posEntry, ge *[]GrenadeEntry, fe *[]FireEntry, ee *[]EventEntry, ri map[int]model.RoundInfo, Fp map[int][]r2.Point) {
	p.RegisterEventHandler(func(e events.MatchStartedChanged) {
		gs := p.GameState()
		if gs.GamePhase() != common.GamePhaseStartGamePhase {
			return
		}
		if *rt.Rounds > 1 {
			return
		}
		for k := range ri {
			delete(ri, k)
		}
		for k := range Fp {
			delete(Fp, k)
		}
		ri[*rt.Rounds] = model.RoundInfo{
			PlayerPositions: make(map[int]map[int64]model.PlayerState),
			PlayerNames:     make(map[int64]model.PlayerInfo),
			GrenadeEvents:   make(map[int]map[int]model.GrenadeState),
			FirePositions:   make(map[int]map[int]model.FireState),
			RoundTimeline:   make(map[int]model.RoundEvent),
		}
		*pe = (*pe)[:0]
		*ge = (*ge)[:0]
		*fe = (*fe)[:0]
		*ee = (*ee)[:0]

	})
}
func setUpPositionTracking(p dem.Parser, rt *model.RoundTracker, ri map[int]model.RoundInfo, buff *[]posEntry, batch chan []posEntry) {
	p.RegisterEventHandler(func(events.FrameDone) {
		if *rt.Live {
			gs := p.GameState()
			tick := gs.IngameTick()
			if tick%4 != 0 {
				return
			}
			rounds := *rt.Rounds
			for _, player := range gs.Participants().Playing() {
				if player.IsAlive() {
					sqlEntry, playerState := get_pos_entry(player, tick, *rt.Rounds, rt.Matchid, isMoving)
					*buff = append(*buff, sqlEntry)
					if _, ok := ri[rounds].PlayerNames[int64(player.SteamID64)]; !ok {
						ri[rounds].PlayerNames[int64(player.SteamID64)] = model.PlayerInfo{Name: player.Name, Side: int(player.GetTeam())}
					}
					if len(ri[rounds].PlayerPositions[tick]) == 0 {
						ri[rounds].PlayerPositions[tick] = make(map[int64]model.PlayerState)
					}
					ri[rounds].PlayerPositions[tick][int64(player.SteamID64)] = playerState
				}

			}
		}
	})
	p.RegisterEventHandler(func(e events.RoundStart) {
		if !*rt.Live {
			return
		}
		ri[*rt.Rounds] = model.RoundInfo{
			PlayerPositions: make(map[int]map[int64]model.PlayerState),
			PlayerNames:     make(map[int64]model.PlayerInfo),
			GrenadeEvents:   make(map[int]map[int]model.GrenadeState),
			FirePositions:   make(map[int]map[int]model.FireState),
			RoundTimeline:   make(map[int]model.RoundEvent),
		}
		database.DB.Exec("INSERT IGNORE INTO ROUNDS (MATCHID, ROUND_NO) VALUES (?,?)", rt.Matchid, *rt.Rounds)
		for _, players := range p.GameState().Participants().Playing() {
			database.DB.Exec("INSERT IGNORE INTO ROUND_PARTICIPANTS (MATCHID, ROUND_NO, PLAYERID, SIDE) VALUES (?,?,?,?)", rt.Matchid, *rt.Rounds, players.SteamID64, int(players.GetTeam()))
		}
	})
	p.RegisterEventHandler(func(e events.RoundEndOfficial) {
		if !*rt.Live {
			return
		}
		fmt.Printf("Round %d ended. Flushing %d row of PLAYER_EVENTS to DB...\n", *rt.Rounds, len(*buff))

		if len(*buff) > 0 {
			sendBatch := make([]posEntry, len(*buff))
			copy(sendBatch, *buff)
			batch <- sendBatch
			*buff = (*buff)[:0]
		}
	})
}
func setUpFireTracking(p dem.Parser, rt *model.RoundTracker, ri map[int]model.RoundInfo, Fp map[int][]r2.Point, buff *[]FireEntry, batch chan []FireEntry) {
	p.RegisterEventHandler(func(events.FrameDone) {
		if *rt.Live {
			gs := p.GameState()
			tick := gs.IngameTick()
			flames := gs.Infernos()
			if tick%4 != 0 {
				return
			}
			if len(flames) != 0 {
				for key, inf := range flames {
					lastState := Fp[key]
					if slices.Equal(lastState, inf.Fires().Active().ConvexHull2D()) {
						return
					} else {
						if len(ri[*rt.Rounds].FirePositions[tick]) == 0 {
							ri[*rt.Rounds].FirePositions[tick] = make(map[int]model.FireState)
						}
						if len(inf.Fires().Active().ConvexHull2D()) == 0 {
							ri[*rt.Rounds].FirePositions[tick][key] = model.FireState{
								Vertices: inf.Fires().ConvexHull2D(), Status: "ENDING",
							}
							for i, fire := range inf.Fires().ConvexHull2D() {
								*buff = append(*buff, FireEntry{
									rt.Matchid, *rt.Rounds, tick, key, i, fire.X, fire.Y, "ENDING",
								})
							}
							Fp[key] = inf.Fires().Active().ConvexHull2D()
						} else {
							ri[*rt.Rounds].FirePositions[tick][key] = model.FireState{
								Vertices: inf.Fires().Active().ConvexHull2D(), Status: "SPREADING",
							}
							for i, fire := range inf.Fires().Active().ConvexHull2D() {
								*buff = append(*buff, FireEntry{
									rt.Matchid, *rt.Rounds, tick, key, i, fire.X, fire.Y, "SPREADING",
								})
							}
							Fp[key] = inf.Fires().Active().ConvexHull2D()
						}

					}
				}

			}
		}
	})
	p.RegisterEventHandler(func(g events.InfernoStart) {
		if *rt.Live == false {
			return
		}
		gs := p.GameState()
		tick := gs.IngameTick()
		id := g.Inferno.Entity.ID()

		if len(ri[*rt.Rounds].FirePositions[tick]) == 0 {
			ri[*rt.Rounds].FirePositions[tick] = make(map[int]model.FireState)
		}
		ri[*rt.Rounds].FirePositions[tick][id] = model.FireState{
			Vertices: g.Inferno.Fires().Active().ConvexHull2D(), Status: "STARTING",
		}
		for i, fire := range g.Inferno.Fires().Active().ConvexHull2D() {
			*buff = append(*buff, FireEntry{
				rt.Matchid, *rt.Rounds, tick, id, i, fire.X, fire.Y, "STARTING",
			})
		}

	})
	p.RegisterEventHandler(func(g events.InfernoExpired) {
		if *rt.Live == false {
			return
		}
		gs := p.GameState()
		tick := gs.IngameTick()
		id := g.Inferno.Entity.ID()
		if len(ri[*rt.Rounds].FirePositions[tick]) == 0 {
			ri[*rt.Rounds].FirePositions[tick] = make(map[int]model.FireState)
		}
		ri[*rt.Rounds].FirePositions[tick][id] = model.FireState{
			Vertices: g.Inferno.Fires().ConvexHull2D(), Status: "ENDING",
		}
		for i, fire := range g.Inferno.Fires().ConvexHull2D() {
			*buff = append(*buff, FireEntry{
				rt.Matchid, *rt.Rounds, tick, id, i, fire.X, fire.Y, "ENDING",
			})
		}
	})
	p.RegisterEventHandler(func(e events.RoundEndOfficial) {
		if !*rt.Live {
			return
		}
		fmt.Printf("Round %d ended. Flushing %d rows of FireEvents to DB...\n", *rt.Rounds, len(*buff))

		if len(*buff) > 0 {
			sendBatch := make([]FireEntry, len(*buff))
			copy(sendBatch, *buff)
			batch <- sendBatch
			*buff = (*buff)[:0]
		}
	})
}
func setUpEntityTracking(p dem.Parser, rt *model.RoundTracker, ri map[int]model.RoundInfo, buff *[]GrenadeEntry, batch chan []GrenadeEntry) {
	p.RegisterEventHandler(func(e events.RoundEndOfficial) {
		if !*rt.Live {
			return
		}
		fmt.Printf("Round %d ended. Flushing %d row of GRENADE_EVENTS to DB...\n", *rt.Rounds, len(*buff))

		if len(*buff) > 0 {
			sendBatch := make([]GrenadeEntry, len(*buff))
			copy(sendBatch, *buff)
			batch <- sendBatch
			*buff = (*buff)[:0]
		}
	})

	p.RegisterEventHandler(func(be events.BombPlanted) {
		if !*rt.Live {
			return
		}
		player := be.Player
		tick := p.GameState().IngameTick()
		matchInfo := base_grenade{
			tick: tick, roundNo: *rt.Rounds, matchid: rt.Matchid,
			grenid: -1, gren_type: int(common.EqBomb), player: *be.Player,
			pos: player.Position(),
		}
		sqlEntry1, grenadeState := get_grenade_entry(matchInfo, "PLANTED")
		*buff = append(*buff, sqlEntry1)
		if len(ri[*rt.Rounds].GrenadeEvents[tick]) == 0 {
			ri[*rt.Rounds].GrenadeEvents[tick] = make(map[int]model.GrenadeState)
		}
		ri[*rt.Rounds].GrenadeEvents[tick][-1] = grenadeState
	})
	p.RegisterEventHandler(func(be events.BombDefused) {
		if !*rt.Live {
			return
		}
		player := be.Player
		tick := p.GameState().IngameTick()
		rounds := *rt.Rounds
		matchid := rt.Matchid
		matchInfo := base_grenade{
			tick: tick, roundNo: rounds, matchid: matchid,
			grenid: -1, gren_type: int(common.EqBomb), player: *be.Player,
			pos: player.Position(),
		}

		sqlEntry1, grenadeState := get_grenade_entry(matchInfo, "DEFUSED")
		*buff = append(*buff, sqlEntry1)
		if len(ri[rounds].GrenadeEvents[tick]) == 0 {
			ri[rounds].GrenadeEvents[tick] = make(map[int]model.GrenadeState)
		}
		ri[rounds].GrenadeEvents[tick][-1] = grenadeState
	})
	p.RegisterEventHandler(func(be events.BombDropped) {
		if !*rt.Live {
			return
		}
		player := be.Player
		tick := p.GameState().IngameTick()
		rounds := *rt.Rounds
		matchid := rt.Matchid
		matchInfo := base_grenade{
			tick: tick, roundNo: rounds, matchid: matchid,
			grenid: -1, gren_type: int(common.EqBomb), player: *be.Player,
			pos: player.Position(),
		}

		sqlEntry1, grenadeState := get_grenade_entry(matchInfo, "DROPPED")
		*buff = append(*buff, sqlEntry1)
		if len(ri[rounds].GrenadeEvents[tick]) == 0 {
			ri[rounds].GrenadeEvents[tick] = make(map[int]model.GrenadeState)
		}
		ri[rounds].GrenadeEvents[tick][-1] = grenadeState
	})
	p.RegisterEventHandler(func(be events.BombPickup) {
		if !*rt.Live {
			return
		}
		player := be.Player
		tick := p.GameState().IngameTick()
		rounds := *rt.Rounds
		matchid := rt.Matchid
		matchInfo := base_grenade{
			tick: tick, roundNo: rounds, matchid: matchid,
			grenid: -1, gren_type: int(common.EqBomb), player: *be.Player,
			pos: player.Position(),
		}
		sqlEntry, grenadeState := get_grenade_entry(matchInfo, "GRABBED")
		*buff = append(*buff, sqlEntry)
		if len(ri[rounds].GrenadeEvents[tick]) == 0 {
			ri[rounds].GrenadeEvents[tick] = make(map[int]model.GrenadeState)
		}
		ri[rounds].GrenadeEvents[tick][-1] = grenadeState
	})
	p.RegisterEventHandler(func(g events.GrenadeProjectileThrow) {
		if !*rt.Live {
			return
		}
		tick := p.GameState().IngameTick()
		id := g.Projectile.Entity.ID()
		matchInfo := base_grenade{
			tick: tick, roundNo: *rt.Rounds, matchid: rt.Matchid,
			grenid: id, gren_type: int(g.Projectile.WeaponInstance.Type), player: *g.Projectile.Thrower,
			pos: g.Projectile.Position(),
		}
		sqlEntry, gs := get_grenade_entry(matchInfo, "FLYING")
		*buff = append(*buff, sqlEntry)
		if len(ri[*rt.Rounds].GrenadeEvents[tick]) == 0 {
			ri[*rt.Rounds].GrenadeEvents[tick] = make(map[int]model.GrenadeState)
		}
		ri[*rt.Rounds].GrenadeEvents[tick][id] = gs
		g.Projectile.Entity.OnPositionUpdate(func(pos r3.Vector) {
			upTick := p.GameState().IngameTick()
			if upTick%4 == 0 {
				matchInfo := base_grenade{
					tick: upTick, roundNo: *rt.Rounds, matchid: rt.Matchid,
					grenid: id, gren_type: int(g.Projectile.WeaponInstance.Type), player: *g.Projectile.Thrower,
					pos: pos,
				}
				sqlEntry, grenadeState := get_grenade_entry(matchInfo, "FLYING")
				*buff = append(*buff, sqlEntry)
				if len(ri[*rt.Rounds].PlayerPositions[upTick]) == 0 {
					ri[*rt.Rounds].GrenadeEvents[upTick] = make(map[int]model.GrenadeState)
				}
				ri[*rt.Rounds].GrenadeEvents[upTick][id] = grenadeState
			}
		})
	})
	p.RegisterEventHandler(func(g events.SmokeStart) {
		if *rt.Live != true {
			return
		}
		tick := p.GameState().IngameTick()
		id := g.GrenadeEntityID
		position := r3.Vector{X: g.Position.X, Y: g.Position.Y, Z: g.Position.Z}
		matchInfo := base_grenade{
			tick: tick, roundNo: *rt.Rounds, matchid: rt.Matchid,
			grenid: id, gren_type: int(g.GrenadeType), player: *g.Thrower,
			pos: position,
		}
		sqlEntry, grenadeState := get_grenade_entry(matchInfo, "BLOOMED")
		*buff = append(*buff, sqlEntry)
		if len(ri[*rt.Rounds].PlayerPositions[tick]) == 0 {
			ri[*rt.Rounds].GrenadeEvents[tick] = make(map[int]model.GrenadeState)

		}
		ri[*rt.Rounds].GrenadeEvents[tick][id] = grenadeState
	})
	p.RegisterEventHandler(func(g events.SmokeExpired) {
		if *rt.Live != true {
			return
		}
		tick := p.GameState().IngameTick()
		id := g.GrenadeEntityID
		position := r3.Vector{X: g.Position.X, Y: g.Position.Y, Z: g.Position.Z}
		matchInfo := base_grenade{
			tick: tick, roundNo: *rt.Rounds, matchid: rt.Matchid,
			grenid: id, gren_type: int(g.GrenadeType), player: *g.Thrower,
			pos: position,
		}
		sqlEntry, grenadeState := get_grenade_entry(matchInfo, "EXPIRED")
		*buff = append(*buff, sqlEntry)
		if len(ri[*rt.Rounds].PlayerPositions[tick]) == 0 {
			ri[*rt.Rounds].GrenadeEvents[tick] = make(map[int]model.GrenadeState)

		}
		ri[*rt.Rounds].GrenadeEvents[tick][id] = grenadeState
	})
	p.RegisterEventHandler(func(g events.HeExplode) {
		if *rt.Live != true {
			return
		}
		tick := p.GameState().IngameTick()
		id := g.GrenadeEntityID
		position := r3.Vector{X: g.Position.X, Y: g.Position.Y, Z: g.Position.Z}
		matchInfo := base_grenade{
			tick: tick, roundNo: *rt.Rounds, matchid: rt.Matchid,
			grenid: id, gren_type: int(g.GrenadeType), player: *g.Thrower,
			pos: position,
		}
		sqlEntry, grenadeState := get_grenade_entry(matchInfo, "EXPIRED")
		*buff = append(*buff, sqlEntry)
		if len(ri[*rt.Rounds].PlayerPositions[tick]) == 0 {
			ri[*rt.Rounds].GrenadeEvents[tick] = make(map[int]model.GrenadeState)

		}
		ri[*rt.Rounds].GrenadeEvents[tick][id] = grenadeState
	})
	p.RegisterEventHandler(func(g events.FlashExplode) {
		if *rt.Live != true {
			return
		}
		tick := p.GameState().IngameTick()
		id := g.GrenadeEntityID
		position := r3.Vector{X: g.Position.X, Y: g.Position.Y, Z: g.Position.Z}
		matchInfo := base_grenade{
			tick: tick, roundNo: *rt.Rounds, matchid: rt.Matchid,
			grenid: id, gren_type: int(g.GrenadeType), player: *g.Thrower,
			pos: position,
		}
		sqlEntry, grenadeState := get_grenade_entry(matchInfo, "EXPIRED")
		*buff = append(*buff, sqlEntry)
		if len(ri[*rt.Rounds].PlayerPositions[tick]) == 0 {
			ri[*rt.Rounds].GrenadeEvents[tick] = make(map[int]model.GrenadeState)

		}
		ri[*rt.Rounds].GrenadeEvents[tick][id] = grenadeState
	})

}

func setUpEventTracking(p dem.Parser, rt *model.RoundTracker, ri map[int]model.RoundInfo, buff *[]EventEntry, batch chan []EventEntry) {
	p.RegisterEventHandler(func(k events.Kill) {
		var (
			id1 = int64(0)
			id2 = int64(0)
		)
		if !*rt.Live {
			return
		}
		if k.Killer != nil {
			id1 = int64(k.Killer.SteamID64)
		}
		if k.Victim != nil {
			id2 = int64(k.Victim.SteamID64)
		}
		b_e := base_event{
			tick: p.GameState().IngameTick(), roundNo: *rt.Rounds, matchid: rt.Matchid, steamid1: id1, steamid2: id2,
		}
		ee, re := get_event_entry(b_e, model.PlayerKilled)
		ri[*rt.Rounds].RoundTimeline[p.GameState().IngameTick()] = re
		*buff = append(*buff, ee)
	})
	p.RegisterEventHandler(func(events.RoundFreezetimeEnd) {
		if *rt.Live {
			tick := p.GameState().IngameTick()
			be := base_event{
				tick: tick, roundNo: *rt.Rounds, matchid: rt.Matchid, steamid1: 0, steamid2: 0,
			}
			ee, re := get_event_entry(be, model.FreezeTimeEnd)
			ri[*rt.Rounds].RoundTimeline[tick] = re
			*buff = append(*buff, ee)
		}
	})
	p.RegisterEventHandler(func(be events.BombPlanted) {
		if !*rt.Live {
			return
		}
		tick := p.GameState().IngameTick()
		b_e := base_event{
			tick: tick, roundNo: *rt.Rounds, matchid: rt.Matchid, steamid1: int64(be.Player.SteamID64), steamid2: 0,
		}
		ee, re := get_event_entry(b_e, model.BombPlanted)
		ri[*rt.Rounds].RoundTimeline[tick] = re
		*buff = append(*buff, ee)
	})
	p.RegisterEventHandler(func(be events.BombDefused) {
		if !*rt.Live {
			return
		}
		tick := p.GameState().IngameTick()
		b_e := base_event{
			tick: tick, roundNo: *rt.Rounds, matchid: rt.Matchid, steamid1: int64(be.Player.SteamID64), steamid2: 0,
		}
		ee, re := get_event_entry(b_e, model.BombDefused)
		ri[*rt.Rounds].RoundTimeline[tick] = re
		*buff = append(*buff, ee)
	})
	p.RegisterEventHandler(func(g events.GrenadeProjectileThrow) {
		if *rt.Live != true {
			return
		}
		tick := p.GameState().IngameTick()
		b_e := base_event{
			tick: tick, roundNo: *rt.Rounds, matchid: rt.Matchid, steamid1: int64(g.Projectile.Thrower.SteamID64), steamid2: 0,
		}
		var re model.RoundEvent
		var ee EventEntry
		switch g.Projectile.WeaponInstance.Type {
		case common.EqSmoke:
			ee, re = get_event_entry(b_e, model.SmokeThrow)
		case common.EqHE:
			ee, re = get_event_entry(b_e, model.HeThrow)
		case common.EqFlash:
			ee, re = get_event_entry(b_e, model.FlashThrow)
		case common.EqIncendiary:
		case common.EqMolotov:
			ee, re = get_event_entry(b_e, model.FireThrow)
		case common.EqDecoy:
			ee, re = get_event_entry(b_e, model.DecoyThrow)
		}
		ri[*rt.Rounds].RoundTimeline[tick] = re
		*buff = append(*buff, ee)
	})
	p.RegisterEventHandler(func(e events.RoundEndOfficial) {
		if !*rt.Live {
			return
		}
		fmt.Printf("Round %d ended. Flushing %d row of ROUND_EVENTS to DB...\n", *rt.Rounds, len(*buff))

		if len(*buff) > 0 {
			sendBatch := make([]EventEntry, len(*buff))
			copy(sendBatch, *buff)
			batch <- sendBatch
			*buff = (*buff)[:0]
		}
	})
}
func get_pos_entry(player *common.Player, tick int, round int, matchid int, action model.PlayerAction) (posEntry, model.PlayerState) {
	pos := player.Position()
	inv := player.Weapons()
	hasBomb := false
	var (
		primary   = common.EqUnknown
		secondary = common.EqUnknown
		smoke     = common.EqUnknown
		hegren    = common.EqUnknown
		fire      = common.EqUnknown
		flash1    = common.EqUnknown
		flash2    = common.EqUnknown
		decoy     = common.EqUnknown
	)
	for _, wep := range inv {
		if wep.Type == common.EqBomb {
			hasBomb = true
		}
		switch wep.Class() {
		case common.EqClassRifle, common.EqClassHeavy, common.EqClassSMG:
			primary = wep.Type
		case common.EqClassPistols:
			secondary = wep.Type
		case common.EqClassGrenade:
			switch wep.Type {
			case common.EqSmoke:
				smoke = wep.Type
			case common.EqHE:
				hegren = wep.Type
			case common.EqFlash:
				if player.FlashbangCount() == 2 {
					flash1 = wep.Type
					flash2 = wep.Type
				} else if player.FlashbangCount() == 1 {
					flash1 = wep.Type
				}
			case common.EqIncendiary:
			case common.EqMolotov:
				fire = wep.Type
			case common.EqDecoy:
				decoy = wep.Type
			}

		}
	}

	var activeWep int
	if player.ActiveWeapon() == nil {
		activeWep = 0
	} else {
		activeWep = int(player.ActiveWeapon().Type)
	}
	pE := posEntry{
		matchid, round, tick, int(player.GetTeam()), player.SteamID64, player.Health(), player.Kills(),
		player.Assists(), player.Deaths(), player.Armor(), player.Money(), int(primary), int(secondary),
		int(smoke), int(hegren), int(flash1), int(flash2), int(fire), int(decoy), hasBomb, pos.X, pos.Y, pos.Z,
		player.FlashDurationTimeRemaining().Seconds(), activeWep, action, player.ViewDirectionX(),
	}
	pS := model.PlayerState{
		Position: player.Position(), Active_Weapon: activeWep, HP: player.Health(),
		Kills: player.Kills(), Assists: player.Assists(), Deaths: player.Deaths(),
		Primary: int(primary), Secondary: int(secondary), SmokeSlot: int(smoke), HESlot: int(hegren),
		Flashslot1: int(flash1), FlashSlot2: int(flash2), DecoySlot: int(decoy), FireSlot: int(fire),
		Armor: player.Armor(), Money: player.Money(), Action: action, HasBomb: hasBomb,
		BlindDuration: player.FlashDurationTimeRemaining().Seconds(), ViewAngle: player.ViewDirectionX(),
	}
	return pE, pS
}
func get_grenade_entry(base base_grenade, grenState string) (GrenadeEntry, model.GrenadeState) {
	player := base.player
	grenid := base.grenid
	gren_type := base.gren_type
	position := base.pos
	ge := GrenadeEntry{
		base.matchid, base.roundNo, base.tick, grenid, player.SteamID64, position.X, position.Y, position.Z, gren_type, grenState,
	}
	gs := model.GrenadeState{
		Position: position, Grenade: gren_type, ThrownByName: player.Name, ThrownByid: int64(player.SteamID64), Status: grenState,
	}
	return ge, gs
}
func get_event_entry(base base_event, event_type model.TrackedEvents) (EventEntry, model.RoundEvent) {
	ee := EventEntry{
		tick: base.tick, roundNo: base.roundNo, matchid: base.matchid,
		event: int(event_type), steamid1: int64(base.steamid1), steamid2: int64(base.steamid2),
	}
	re := model.RoundEvent{
		Event: event_type, Player1: int64(base.steamid1), Player2: int64(base.steamid2),
	}
	return ee, re
}
func flushToDB(db *sql.DB, entries []posEntry) {
	tx, err := db.Begin()
	if err != nil {
		panic(err)
	}

	Players := make(map[uint64]int)

	for _, e := range entries {
		_, err = tx.Exec("INSERT IGNORE INTO ROUNDS (MATCHID, ROUND_NO) VALUES (?,?)", e.matchID, e.roundNo)
		if err != nil {
			panic(err)
		}
		_, ok := Players[e.steamID]
		if !ok {
			Players[e.steamID] = e.side
			_, err = db.Exec("INSERT IGNORE INTO ROUND_PARTICIPANTS (MATCHID, ROUND_NO, PLAYERID, SIDE) VALUES (?,?,?,?)", e.matchID, e.roundNo, e.steamID, e.side)
			if err != nil {
				panic(err)
			}
		} else {
			continue
		}
	}

	stmt, err := tx.Prepare("INSERT INTO PLAYER_EVENTS" +
		"(MATCHID, ROUND_NO, PLAYERID, HP, ACTIVE_WEAPON, HAS_BOMB, KILLS, ASSISTS, DEATHS, ARMOR, DINERO, P_ACTION," +
		"PRIMARY_SLOT,SECONDARY_SLOT,SMOKE_SLOT,FIRE_SLOT,HE_SLOT,DECOY_SLOT,FLASH_SLOT1,FLASH_SLOT2,FLASHED_DURATION,VIEW_ANGLE,XPOS,YPOS,ZPOS,TICK) VALUES" +
		"(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)" +
		"ON DUPLICATE KEY UPDATE ACTIVE_WEAPON=(ACTIVE_WEAPON), XPOS=VALUES(XPOS), YPOS=VALUES(YPOS), ZPOS=VALUES(ZPOS)")
	if err != nil {
		panic(err)
	}
	defer stmt.Close()

	for _, e := range entries {
		if _, err := stmt.Exec(e.matchID, e.roundNo, e.steamID, e.hp, e.weapon, e.hasBomb, e.kills, e.assists, e.deaths, e.armor, e.money, e.action,
			e.primary, e.seconday, e.smoke, e.fire, e.he, e.decoy, e.flash1, e.flash2, e.flashDur, e.view, e.x, e.y, e.z, e.tick); err != nil {
			tx.Rollback()
			panic(err)
		}
	}

	if err := tx.Commit(); err != nil {
		panic(err)
	}
}

func fireFlush(db *sql.DB, entries []FireEntry) {
	tx, err := db.Begin()
	if err != nil {
		panic(err)
	}
	stmt, err := tx.Prepare("INSERT INTO FIRE_VERTICES (MATCHID,ROUND_NO,TICK,ENTITYID,FIREID,XPOS,YPOS,ENTSTATE)" +
		"VALUES (?,?,?,?,?,?,?,?) ON DUPLICATE KEY UPDATE XPOS=VALUES(XPOS), YPOS=VALUES(YPOS)")
	if err != nil {
		panic(err)
	}
	defer stmt.Close()
	for _, e := range entries {
		if _, err := stmt.Exec(e.matchID, e.roundNo, e.tick, e.entid, e.fireid, e.x, e.y, e.state); err != nil {
			tx.Rollback()
			panic(err)
		}
	}

	if err := tx.Commit(); err != nil {
		panic(err)
	}
}
func eventFlush(db *sql.DB, entries []EventEntry) {
	tx, err := db.Begin()
	if err != nil {
		panic(err)
	}
	for _, e := range entries {
		_, err = tx.Exec("INSERT IGNORE INTO ROUNDS (MATCHID, ROUND_NO) VALUES (?,?)", e.matchid, e.roundNo)
		if err != nil {
			panic(err)
		}
		break
	}
	stmt, err := tx.Prepare("INSERT IGNORE INTO ROUND_EVENTS (MATCHID,ROUND_NO,TICK,EVENT_TYPE,PLAYER1ID,PLAYER2ID)" +
		"VALUES (?,?,?,?,?,?)")
	if err != nil {
		panic(err)
	}
	defer stmt.Close()
	for _, e := range entries {
		if _, err := stmt.Exec(e.matchid, e.roundNo, e.tick, e.event, e.steamid1, e.steamid2); err != nil {
			tx.Rollback()
			panic(err)
		}
	}

	if err := tx.Commit(); err != nil {
		panic(err)
	}
}
func grenadeFlush(db *sql.DB, entries []GrenadeEntry) {
	tx, err := db.Begin()
	if err != nil {
		panic(err)
	}

	stmt, err := tx.Prepare("INSERT INTO GRENADE_EVENTS (MATCHID,ROUND_NO,PLAYERID,ENTITYID,TICK,XPOS,YPOS,ZPOS,GRENADE,ENTSTATE) VALUES (?,?,?,?,?,?,?,?,?,?)" +
		"ON DUPLICATE KEY UPDATE GRENADE=(GRENADE), XPOS=VALUES(XPOS), YPOS=VALUES(YPOS), ZPOS=VALUES(ZPOS)")
	if err != nil {
		panic(err)
	}
	defer stmt.Close()

	for _, e := range entries {
		if _, err := stmt.Exec(e.matchID, e.roundNo, e.steamID, e.entid, e.tick, e.x, e.y, e.z, e.grenade, e.state); err != nil {
			tx.Rollback()
			panic(err)
		}
	}

	if err := tx.Commit(); err != nil {
		panic(err)
	}
}
