package main

import (
	"github.com/golang/geo/r2"
	"github.com/golang/geo/r3"
	ex "github.com/markus-wa/demoinfocs-golang/v5/examples"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
)

type BaseDemo struct {
	FileName  string  `json:"filename,string"`
	ModDate   string  `json:"date,string"`
	FileSize  string  `json:"filesize,string"`
	Map       string  `json:"map,string"`
	TeamStats [2]Team `json:"team_stats"`
	Parsed    bool    `json:"parsed"`
	BaseStats bool    `json:"stats"`
	ID        int
}

type PlayerStats struct {
	Kills   int `json:"kills"`
	Deaths  int `json:"deaths"`
	Assists int `json:"assists"`
}
type MatchEvents struct {
	RoundPositions RoundInfo                   `json:"round_events"`
	Rounds         int                         `json:"rounds"`
	MapMeta        ex.Map                      `json:"map"`
	Teams          map[string]map[int64]string `json:"teams"`
}
type RoundInfo struct {
	// map[TICK] -> Map(ID) i.e playerid or ent id -> State/Info
	PlayerPositions map[int]map[int64]PlayerState `json:"player_positions"`
	PlayerNames     map[int64]PlayerInfo          `json:"player_info"`
	GrenadeEvents   map[int]map[int]GrenadeState  `json:"grenade_events"`
	FirePositions   map[int]map[int]FireState     `json:"fire_events"`
	// RoundEvents don't have an id
	RoundTimeline map[int]RoundEvent `json:"round_timeline"`
}

// Flashes, Kills, BombPlants and Defuses, Freezetime
type RoundEvent struct {
	Event   TrackedEvents `json:"events"`
	Player1 int64         `json:"player1,string"`
	Player2 int64         `json:"player2,string"`
}

type TrackedEvents int

const (
	UnknownEvent TrackedEvents = iota
	BombPlanted
	BombDefused
	FreezeTimeEnd
	PlayerKilled
	FireThrow
	SmokeThrow
	FlashThrow
	HeThrow
	DecoyThrow
)

type EventEntry struct {
	matchid, roundNo, tick int
	event                  int
	steamid1, steamid2     int64
}
type PlayerInfo struct {
	Name string `json:"name"`
	Side int    `json:"side"`
}
type Player struct {
	Name  string      `json:"name"`
	ID    int64       `json:"ID,string"`
	Stats PlayerStats `json:"stats"`
}
type Team struct {
	ID             int               `json:"ID"`
	ClanName       string            `json:"Clanname"`
	EndScore       int               `json:"Endscore"`
	TScore         int               `json:"TScore"`
	CTScore        int               `json:"CTScore"`
	PlayingPlayers map[string]Player `json:"Playing"`
	inited         bool
}
type posEntry struct {
	matchID, roundNo, tick, side                              int
	steamID                                                   uint64
	hp, kills, assists, deaths, armor, money                  int
	primary, seconday, smoke, he, flash1, flash2, fire, decoy int
	hasBomb                                                   bool
	x, y, z, flashDur                                         float64
	weapon                                                    int
	action                                                    PlayerAction
	view                                                      float32
}
type PlayerState struct {
	Position      r3.Vector    `json:"vector"`
	Active_Weapon int          `json:"active_weapon"`
	Primary       int          `json:"primary"`
	Secondary     int          `json:"secondary"`
	SmokeSlot     int          `json:"smoke_slot"`
	HESlot        int          `json:"he_slot"`
	FireSlot      int          `json:"fire_slot"`
	Flashslot1    int          `json:"flash_slot1"`
	FlashSlot2    int          `json:"flash_slot2"`
	DecoySlot     int          `json:"decoy_slot"`
	HP            int          `json:"hp"`
	Kills         int          `json:"kills"`
	Assists       int          `json:"assists"`
	Deaths        int          `json:"deaths"`
	Armor         int          `json:"armor"`
	Money         int          `json:"dinero"`
	Action        PlayerAction `json:"action"`
	HasBomb       bool         `json:"hasBomb"`
	BlindDuration float64      `json:"blind_dur"`
	ViewAngle     float32      `json:"view_angle,float"`
}
type PlayerAction int

const (
	isMoving PlayerAction = iota
	beginPlanting
	donePlanting
	abortedPlant
	beginDefusing
	doneDefusing
	abortedDefuse
)

type GrenadeState struct {
	Position     r3.Vector `json:"vector"`
	Grenade      int       `json:"grenade"`
	ThrownByName string    `json:"thrownBy"`
	ThrownByid   int64     `json:"thrownById,string"`
	Status       string    `json:"status"`
}
type FireState struct {
	Vertices []r2.Point `json:"vertices"`
	Status   string     `json:"status"`
}
type base_grenade struct {
	matchid, roundNo, tick int
	grenid, gren_type      int
	player                 common.Player
	pos                    r3.Vector
}
type base_event struct {
	matchid, roundNo, tick int
	event                  int
	steamid1, steamid2     int64
}

type GrenadeEntry struct {
	matchID, roundNo, tick int
	entid                  int
	steamID                uint64
	x, y, z                float64
	grenade                int
	state                  string
}
type FireEntry struct {
	matchID, roundNo, tick, entid, fireid int
	x, y                                  float64
	state                                 string
}
