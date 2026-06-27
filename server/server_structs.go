package main

import "database/sql"

// var downloaded string = "/Users/carlosherrera/Documents/CS2DEMOS"
// var downloaded string = "/workspaces/GoCs/uploads"
type SteamResponse struct {
	Response PlayerSummariesResult `json:"response"`
}

type PlayerSummariesResult struct {
	Players []SteamPlayer `json:"players"`
}

type SteamPlayer struct {
	SteamID      string `json:"steamid"`
	PersonaName  string `json:"personaname"` // The user's display name
	ProfileURL   string `json:"profileurl"`
	Avatar       string `json:"avatar"`       // 32x32 image
	AvatarMedium string `json:"avatarmedium"` // 64x64 image
	AvatarFull   string `json:"avatarfull"`   // 184x184 image
}

type AccountRegister struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}
type LinkSteamRequest struct {
	SteamID string `json:"steamId"`
}
type PlayerPageData struct {
	AccountID  int64
	Username   string
	SteamID    sql.NullInt64 // NULL if not linked
	SteamVer   bool
	PlayerName sql.NullString // NULL if no demo parsed yet
	TeamName   sql.NullString // NULL if no team
}
type PlayerMatch struct {
	FileName string `json:"file_name"`
	Opponent string `json:"opponent"`
	Result   string `json:"result"`
	Score    string `json:"score"`
	Map      string `json:"map"`
	Kills    int    `json:"kills"`
	Assists  int    `json:"assists"`
	Deaths   int    `json:"deaths"`
}
