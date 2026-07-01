package model

import (
	"github.com/golang/geo/r2"
	"github.com/golang/geo/r3"
	ex "github.com/markus-wa/demoinfocs-golang/v5/examples"
)

type DemoSetup struct {
	MatchId   int
	GameMap   string
	Teams     [2]Team
	Live      bool
	FirstKill bool
}
type Team struct {
	ID             int              `json:"ID"`
	ClanName       string           `json:"Clanname"`
	EndScore       int              `json:"Endscore"`
	TScore         int              `json:"TScore"`
	CTScore        int              `json:"CTScore"`
	PlayingPlayers map[int64]Player `json:"Playing"`
	Inited         bool
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

type MatchEvents struct {
	RoundPositions RoundInfo                   `json:"round_events"`
	Rounds         int                         `json:"rounds"`
	MapMeta        ex.Map                      `json:"map"`
	Teams          map[string]map[int64]string `json:"teams"`
}
type SQLMatch struct {
	MATCHID      int
	DEMO_NAME    string
	SAVED_DATE   string
	PARSED_STATS int
	PARSED_2D    int
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
type BaseDemo struct {
	FileName  string  `json:"filename,string"`
	ModDate   string  `json:"date,string"`
	SavedDate string  `json:"savedate,string"`
	FileSize  string  `json:"filesize,string"`
	Map       string  `json:"map,string"`
	TeamStats [2]Team `json:"team_stats"`
	Parsed    bool    `json:"parsed"`
	BaseStats bool    `json:"stats"`
	ID        int
}

type RoundInfo struct {
	PlayerPositions map[int]map[int64]PlayerState `json:"player_positions"`
	PlayerNames     map[int64]PlayerInfo          `json:"player_info"`
	GrenadeEvents   map[int]map[int]GrenadeState  `json:"grenade_events"`
	FirePositions   map[int]map[int]FireState     `json:"fire_events"`
	RoundTimeline   map[int]RoundEvent            `json:"round_timeline"`
}

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

type PlayerInfo struct {
	Name string `json:"name"`
	Side int    `json:"side"`
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
