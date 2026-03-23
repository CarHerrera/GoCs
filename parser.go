package main

import (
	events "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
	// meta "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/metadata"
	// metadata "github.com/markus-wa/demoinfocs-golang/metadata"
	// "github.com/markus-wa/demoinfocs-golang/common"
	"log"
)

func gameStart(game events.MatchStart, Map string) {

	log.Printf("Game has Started!")
}
func scoreChange(score events.ScoreUpdated) {
	// var score string
	team1 := score.TeamState
	team2 := score.TeamState.Opponent

	log.Printf("%q %v - %v %q", team1.ClanName(), team1.Score(),
		team2.Score(), team2.ClanName())

}

// func main() {
// 	file, err := os.Open("./furvitm4.dem")
// 	p := dem.NewParser(file)
// 	defer p.Close()
// 	defer file.Close()
// 	p.RegisterEventHandler(onKill)
// 	p.RegisterEventHandler(gameStart)
// 	p.RegisterEventHandler(scoreChange)
// 	pf, err := p.ParseNextFrame()
// 	for pf {
// 		// GS := p.GameState()

// 		pf, err = p.ParseNextFrame()
// 	}
// 	if err != nil {
// 		panic(err)
// 	}
// }
