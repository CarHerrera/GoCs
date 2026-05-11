package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"slices"
	"sync"
	"testing"
	"time"

	// Import the MariaDB-compatible driver anonymously
	_ "github.com/go-sql-driver/mysql"
	"github.com/golang/geo/r2"
	"github.com/golang/geo/r3"
	"github.com/joho/godotenv"
	dem "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
)

type SQLMatch struct {
	MATCHID      int
	DEMO_NAME    string
	SAVED_DATE   string
	PARSED_STATS int
	PARSED_2D    int
}

func TestConnect(t *testing.T) {
	_ = godotenv.Load()
	fileName := "Game2Season57.dem"
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	// log.Printf("%s %s", dbUser, dbPassword)
	dbHost := "127.0.0.1"
	dbPort := "3144" // Default MariaDB port
	dbName := os.Getenv("DB_NAME")
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", dbUser, dbPassword, dbHost, dbPort, dbName)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}

	// Important: Configure the pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	DB = db
	// parse_demo_stats(fileName, 2)
	demo := os.Getenv("DEMO_PATH") + fileName
	file, err := os.Open(demo)
	if err != nil {
		t.Error("Error opening file")
	}
	p := dem.NewParser(file)

	defer p.Close()
	// var TeamStats [2]Team
	defer file.Close()
	const batchSize = 5000
	var buffer []GrenadeEntry
	var fireBuffer []FireEntry
	batchChan := make(chan []GrenadeEntry, 100)
	fireBatch := make(chan []FireEntry, 100)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for batch := range batchChan {
			grenadeFlush(DB, batch) // Your helper function with the transaction
		}

		for batch := range fireBatch {
			fireFlush(DB, batch)
		}
	}()
	live := false
	rounds := 0
	p.RegisterEventHandler(func(e events.MatchStartedChanged) {
		if p.GameState().GamePhase() == common.GamePhaseStartGamePhase {
			live = true
			rounds = 1
		}

	})
	var playback MatchEvents
	RoundPositions := make(map[int]RoundEvents)
	FireParticles := make(map[int][]r2.Point)
	p.RegisterEventHandler(func(g events.GrenadeProjectileThrow) {
		if live != true {
			return
		}
		gs := p.GameState()
		tick := gs.IngameTick()
		player := g.Projectile.Thrower
		grenade := g.Projectile.Entity
		id := g.Projectile.Entity.ID()
		buffer = append(buffer, GrenadeEntry{
			2, rounds, tick, id, player.SteamID64, grenade.Position().X, grenade.Position().Y, grenade.Position().Z, g.Projectile.WeaponInstance.String(), "FLYING",
		})
		position := r3.Vector{X: grenade.Position().X, Y: grenade.Position().Y, Z: grenade.Position().Z}
		if len(RoundPositions[rounds].GrenadeEvents[tick]) == 0 {
			RoundPositions[rounds].GrenadeEvents[tick] = make(map[int]GrenadeState)
		}
		RoundPositions[rounds].GrenadeEvents[tick][id] = GrenadeState{
			Position: position, Grenade: g.Projectile.WeaponInstance.String(), ThrownByName: player.Name, ThrownByid: int64(player.SteamID64),
		}
		t.Logf("%v: %v threw %v with ID %v", tick, player.Name, g.Projectile.WeaponInstance.String(), g.Projectile.Entity.ID())
		g.Projectile.Entity.OnPositionUpdate(func(pos r3.Vector) {
			upTick := p.GameState().IngameTick()
			if upTick%4 == 0 {
				buffer = append(buffer, GrenadeEntry{
					2, rounds, upTick, id, player.SteamID64, pos.X, pos.Y, pos.Z, g.Projectile.WeaponInstance.String(), "FLYING",
				})
				position := r3.Vector{X: pos.X, Y: pos.Y, Z: pos.Z}
				if len(RoundPositions[rounds].PlayerPositions[upTick]) == 0 {
					RoundPositions[rounds].GrenadeEvents[upTick] = make(map[int]GrenadeState)
					RoundPositions[rounds].GrenadeEvents[upTick][id] = GrenadeState{
						Position: position, Grenade: g.Projectile.WeaponInstance.String(), ThrownByName: player.Name, ThrownByid: int64(player.SteamID64),
					}
				} else {
					RoundPositions[rounds].GrenadeEvents[upTick][id] = GrenadeState{
						Position: position, Grenade: g.Projectile.WeaponInstance.String(), ThrownByName: player.Name, ThrownByid: int64(player.SteamID64),
					}
				}
			}

		})
	})
	p.RegisterEventHandler(func(g events.GrenadeProjectileDestroy) {
		if live != true {
			return
		}
		if g.Projectile.WeaponInstance.String() == "HE Grenade" || g.Projectile.WeaponInstance.String() == "Flashbang" || g.Projectile.WeaponInstance.String() == "Smoke Grenade" {
			return
		}
		gs := p.GameState()
		tick := gs.IngameTick()
		player := g.Projectile.Thrower
		id := g.Projectile.Entity.ID()
		t.Logf("%v: %v thrown %v with ID %v has landed", tick, player.Name, g.Projectile.WeaponInstance.String(), id)
		if len(RoundPositions[rounds].GrenadeEvents[tick]) == 0 {
			RoundPositions[rounds].GrenadeEvents[tick] = make(map[int]GrenadeState)
		}
		position := g.Projectile.Entity.Position()
		RoundPositions[rounds].GrenadeEvents[tick][id] = GrenadeState{
			Position: position, Grenade: g.Projectile.WeaponInstance.String(), ThrownByName: player.Name, ThrownByid: int64(player.SteamID64),
		}
		buffer = append(buffer, GrenadeEntry{
			2, rounds, tick, id, player.SteamID64, position.X, position.Y, position.Z, g.Projectile.WeaponInstance.String(), "LANDED",
		})
	})
	p.RegisterEventHandler(func(g events.SmokeStart) {
		gs := p.GameState()
		tick := gs.IngameTick()
		id := g.GrenadeEntityID
		player := g.Thrower
		buffer = append(buffer, GrenadeEntry{
			2, rounds, tick, id, player.SteamID64, g.Position.X, g.Position.Y, g.Position.Z, g.Grenade.String(), "BLOOMED",
		})
		t.Logf("%v: %v thrown %v with ID %v and it bloomed", tick, player.Name, g.Grenade.String(), g.GrenadeEntityID)
		if len(RoundPositions[rounds].PlayerPositions[tick]) == 0 {
			RoundPositions[rounds].GrenadeEvents[tick] = make(map[int]GrenadeState)

		}
		position := r3.Vector{X: g.Position.X, Y: g.Position.Y, Z: g.Position.Z}
		RoundPositions[rounds].GrenadeEvents[tick][id] = GrenadeState{
			Position: position, Grenade: g.Grenade.String(), ThrownByName: player.Name, ThrownByid: int64(player.SteamID64),
		}
	})
	p.RegisterEventHandler(func(g events.SmokeExpired) {
		gs := p.GameState()
		tick := gs.IngameTick()
		id := g.GrenadeEntityID
		player := g.Thrower
		buffer = append(buffer, GrenadeEntry{
			2, rounds, tick, id, player.SteamID64, g.Position.X, g.Position.Y, g.Position.Z, g.Grenade.String(), "EXPIRED",
		})
		t.Logf("%v: %v thrown %v with ID %v and it FADED", tick, player.Name, g.Grenade.String(), g.GrenadeEntityID)
		if len(RoundPositions[rounds].PlayerPositions[tick]) == 0 {
			RoundPositions[rounds].GrenadeEvents[tick] = make(map[int]GrenadeState)

		}
		position := r3.Vector{X: g.Position.X, Y: g.Position.Y, Z: g.Position.Z}
		RoundPositions[rounds].GrenadeEvents[tick][id] = GrenadeState{
			Position: position, Grenade: g.Grenade.String(), ThrownByName: player.Name, ThrownByid: int64(player.SteamID64),
		}
	})

	p.RegisterEventHandler(func(g events.HeExplode) {
		gs := p.GameState()
		tick := gs.IngameTick()
		id := g.GrenadeEntityID
		player := g.Thrower
		buffer = append(buffer, GrenadeEntry{
			2, rounds, tick, id, player.SteamID64, g.Position.X, g.Position.Y, g.Position.Z, g.Grenade.String(), "EXPIRED",
		})
		t.Logf("%v: %v thrown %v with ID %v and it EXPLODED", tick, player.Name, g.Grenade.String(), g.GrenadeEntityID)
		if len(RoundPositions[rounds].PlayerPositions[tick]) == 0 {
			RoundPositions[rounds].GrenadeEvents[tick] = make(map[int]GrenadeState)

		}
		position := r3.Vector{X: g.Position.X, Y: g.Position.Y, Z: g.Position.Z}
		RoundPositions[rounds].GrenadeEvents[tick][id] = GrenadeState{
			Position: position, Grenade: g.Grenade.String(), ThrownByName: player.Name, ThrownByid: int64(player.SteamID64),
		}
	})

	p.RegisterEventHandler(func(g events.FlashExplode) {
		gs := p.GameState()
		tick := gs.IngameTick()
		id := g.GrenadeEntityID
		player := g.Thrower
		buffer = append(buffer, GrenadeEntry{
			2, rounds, tick, id, player.SteamID64, g.Position.X, g.Position.Y, g.Position.Z, g.Grenade.String(), "EXPIRED",
		})
		t.Logf("%v: %v thrown %v with ID %v and it EXPLODED", tick, player.Name, g.Grenade.String(), g.GrenadeEntityID)
		if len(RoundPositions[rounds].PlayerPositions[tick]) == 0 {
			RoundPositions[rounds].GrenadeEvents[tick] = make(map[int]GrenadeState)

		}
		position := r3.Vector{X: g.Position.X, Y: g.Position.Y, Z: g.Position.Z}
		RoundPositions[rounds].GrenadeEvents[tick][id] = GrenadeState{
			Position: position, Grenade: g.Grenade.String(), ThrownByName: player.Name, ThrownByid: int64(player.SteamID64),
		}
	})

	p.RegisterEventHandler(func(g events.InfernoStart) {
		if live == false {
			return
		}
		gs := p.GameState()
		tick := gs.IngameTick()
		id := g.Inferno.Entity.ID()
		player := g.Inferno.Thrower()

		t.Logf("%v %v GRENID:%v thrown by %v", tick, "FIRE", id, player)
		// FireParticles[id] = g.Inferno.Fires().ConvexHull3D().Vertices
		if len(RoundPositions[rounds].FirePositions[tick]) == 0 {
			RoundPositions[rounds].FirePositions[tick] = make(map[int]FireState)
		}
		RoundPositions[rounds].FirePositions[tick][id] = FireState{
			Vertices: g.Inferno.Fires().ConvexHull2D(), Status: "STARTING",
		}
		for i, fire := range g.Inferno.Fires().ConvexHull2D() {
			fireBuffer = append(fireBuffer, FireEntry{
				2, rounds, tick, id, i, fire.X, fire.Y, "STARTING",
			})
			t.Logf("%v Fire#%v at %v ID:%v START SPREAD", tick, i, fire, id)
		}

	})
	p.RegisterEventHandler(func(g events.InfernoExpired) {
		if live == false {
			return
		}
		gs := p.GameState()
		tick := gs.IngameTick()
		id := g.Inferno.Entity.ID()
		player := g.Inferno.Thrower()
		t.Logf("%v GRENID:%v thrown by %v EXPIRED", tick, id, player)
		if len(RoundPositions[rounds].FirePositions[tick]) == 0 {
			RoundPositions[rounds].FirePositions[tick] = make(map[int]FireState)
		}
		RoundPositions[rounds].FirePositions[tick][id] = FireState{
			Vertices: g.Inferno.Fires().ConvexHull2D(), Status: "ENDING",
		}
		for i, fire := range g.Inferno.Fires().ConvexHull2D() {
			t.Logf("%v Fire#%v at %v ID:%v EXPIRING", tick, i, fire, id)
			fireBuffer = append(fireBuffer, FireEntry{
				2, rounds, tick, id, i, fire.X, fire.Y, "ENDING",
			})
		}
	})

	p.RegisterEventHandler(func(f events.FrameDone) {
		if live != true {
			return
		}
		GS := p.GameState()
		tick := GS.IngameTick()
		flames := GS.Infernos()
		if GS.IngameTick()%8 != 0 {
			return
		}
		if len(flames) != 0 {
			for key, inf := range flames {
				lastState := FireParticles[key]
				if slices.Equal(lastState, inf.Fires().ConvexHull2D()) {
					return
				} else {
					if len(RoundPositions[rounds].FirePositions[tick]) == 0 {
						RoundPositions[rounds].FirePositions[tick] = make(map[int]FireState)
					}
					RoundPositions[rounds].FirePositions[tick][key] = FireState{
						Vertices: inf.Fires().ConvexHull2D(), Status: "SPREADING",
					}
					for i, fire := range inf.Fires().ConvexHull2D() {
						fireBuffer = append(fireBuffer, FireEntry{
							2, rounds, tick, key, i, fire.X, fire.Y, "SPREADING",
						})
						t.Logf("%v Fire#%v at %v ID:%v SPREADING", tick, i, fire, key)
					}
					FireParticles[key] = inf.Fires().ConvexHull2D()
				}
			}

		}
	})
	p.RegisterEventHandler(func(e events.RoundStart) {
		RoundPositions[rounds] = RoundEvents{
			PlayerPositions: make(map[int]map[int64]PlayerState),
			PlayerNames:     make(map[int64]PlayerInfo),
			GrenadeEvents:   make(map[int]map[int]GrenadeState),
			FirePositions:   make(map[int]map[int]FireState),
		}
	})
	p.RegisterEventHandler(func(e events.RoundEndOfficial) {
		if len(buffer) == 0 {
			return
		}

		fmt.Printf("Round %d ended. Flushing %d rows to DB...\n", rounds, len(buffer)+len(fireBuffer))
		rounds += 1
		// IMPORTANT: Clear the buffer for the next round
		batchToSend := make([]GrenadeEntry, len(buffer))
		copy(batchToSend, buffer)

		batchChan <- batchToSend
		buffer = buffer[:0]

		if len(fireBuffer) == 0 {
			return
		}

		batchtwo := make([]FireEntry, len(fireBuffer))
		copy(batchtwo, fireBuffer)
		fireBatch <- batchtwo
		fireBuffer = fireBuffer[:0]

	})
	err = p.ParseToEnd()
	// t.Logf("%v", RoundPositions[4].GrenadeEvents)
	if err != nil {
		t.Errorf("Error %v", err)
	}
	if len(buffer) > 0 {
		finalBatch := make([]GrenadeEntry, len(buffer))
		copy(finalBatch, buffer)
		batchChan <- finalBatch
		fmt.Printf("Flushing %d rows to DB...\n", len(buffer))
	}
	if len(fireBuffer) > 0 {
		batchtwo := make([]FireEntry, len(fireBuffer))
		copy(batchtwo, fireBuffer)
		fireBatch <- batchtwo
		fireBuffer = fireBuffer[:0]
	}
	close(batchChan)
	close(fireBatch)
	wg.Wait()
	fmt.Printf("Final Round %d ended. F", rounds)
	playback.RoundPositions = RoundPositions[1]
	// 	return c.Status(20).JSON(resp)
}
