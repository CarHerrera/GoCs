package parser

import (
	"server/model"

	"github.com/golang/geo/r3"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
)

const (
	isMoving model.PlayerAction = iota
	beginPlanting
	donePlanting
	abortedPlant
	beginDefusing
	doneDefusing
	abortedDefuse
)

type posEntry struct {
	matchID, roundNo, tick, side                              int
	steamID                                                   uint64
	hp, kills, assists, deaths, armor, money                  int
	primary, seconday, smoke, he, flash1, flash2, fire, decoy int
	hasBomb                                                   bool
	x, y, z, flashDur                                         float64
	weapon                                                    int
	action                                                    model.PlayerAction
	view                                                      float32
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

type EventEntry struct {
	matchid, roundNo, tick int
	event                  int
	steamid1, steamid2     int64
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
