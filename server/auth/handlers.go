package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"database/sql"

	database "server/DB"
	"server/model"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func SteamLogin(c fiber.Ctx) error {
	params := url.Values{
		"openid.ns":         {"http://specs.openid.net/auth/2.0"},
		"openid.mode":       {"checkid_setup"},
		"openid.return_to":  {"http://localhost:4000/auth/steam/callback"},
		"openid.realm":      {"http://localhost:4000"},
		"openid.identity":   {"http://specs.openid.net/auth/2.0/identifier_select"},
		"openid.claimed_id": {"http://specs.openid.net/auth/2.0/identifier_select"},
	}
	steamLoginURL := "https://steamcommunity.com/openid/login?" + params.Encode()
	return c.Redirect().Status(303).To(steamLoginURL)
}

func SteamCallback(c fiber.Ctx) error {
	params, err := url.ParseQuery(string(c.Request().URI().QueryString()))
	if err != nil {
		log.Println("Failed to parse query params:", err)
		return c.Status(400).SendString("Bad request")
	}
	steamAPIKey := os.Getenv("STEAM_API")
	params.Set("openid.mode", "check_authentication")

	resp, err := http.PostForm("https://steamcommunity.com/openid/login", params)
	if err != nil {
		log.Println("Failed to contact Steam for verification:", err)
		return c.Status(500).SendString("Could not contact Steam")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("Failed to read Steam response:", err)
		return c.Status(500).SendString("Could not read Steam response")
	}

	log.Println("Steam verification response:", string(body))

	if !strings.Contains(string(body), "is_valid:true") {
		log.Println("Steam said login is invalid")
		return c.Status(401).SendString("Invalid login")
	}

	claimedID := params.Get("openid.claimed_id")
	steamID := path.Base(claimedID)
	log.Println("Verified Steam ID:", steamID)
	url := fmt.Sprintf(
		"https://api.steampowered.com/ISteamUser/GetPlayerSummaries/v0002/?key=%s&steamids=%s",
		steamAPIKey,
		steamID,
	)

	client := &http.Client{Timeout: 10 * time.Second}

	resp, err = client.Get(url)
	if err != nil {
		return c.Status(500).SendString("Failed to get steam profile")
	}

	defer resp.Body.Close()
	var result model.SteamResponse

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return c.Status(500).SendString("Failed to decode steam profile")
	}
	var accountID int64
	if err := database.DB.QueryRow("SELECT ACCOUNTID FROM USER_ACCOUNTS WHERE STEAMID = ?", steamID).Scan(&accountID); err != nil {
		if err == sql.ErrNoRows {
			if len(result.Response.Players) == 0 {
				return c.Status(500).SendString("no players found for steamID: " + steamID)
			}
			steamPlayer := result.Response.Players[0]
			res, insertErr := database.DB.Exec(
				"INSERT INTO USER_ACCOUNTS (USERNAME, STEAMID, STEAM_VER) VALUES (?, ?, 1)",
				steamPlayer.PersonaName, steamID,
			)
			if insertErr != nil {
				log.Printf("Failed to create Steam account: %v", insertErr)
				return c.Status(500).SendString("Failed to create account")
			}
			newID, insertErr := res.LastInsertId()
			if insertErr != nil {
				return c.Status(500).SendString("Failed to retrieve new account ID")
			}
			token, tokenErr := GenerateJWT(newID)
			if tokenErr != nil {
				return c.Status(500).SendString("Failed to generate token")
			}
			c.Cookie(&fiber.Cookie{
				Name:     "auth_token",
				Value:    token,
				HTTPOnly: true,
				Expires:  time.Now().Add(24 * time.Hour),
				SameSite: "Lax",
			})
			return c.Redirect().Status(303).To("http://localhost:5173/accountHome")
		}
		return c.Status(500).SendString("SQL ERROR")
	} else {
		token, err := GenerateJWT(accountID)
		if err != nil {
			log.Println("Failed to generate JWT:", err)
			return c.Status(500).SendString("Failed to generate token")
		}
		if len(result.Response.Players) == 0 {
			return c.Status(500).SendString("no players found for steamID:" + steamID)
		}
		player := result.Response.Players[0]
		if player.Avatar != "" || player.AvatarFull != "" {
			if _, err := database.DB.Exec(
				"UPDATE PLAYERS SET AVATAR_URL = ?, AVATAR_URL_FULL = ? WHERE PLAYERID = ?",
				player.Avatar, player.AvatarFull, steamID,
			); err != nil {
				log.Printf("avatar upsert failed for steamID %s: %v", steamID, err)
			}
		}
		log.Printf("RESPONSE: %v", result)
		c.Cookie(&fiber.Cookie{
			Name:     "auth_token",
			Value:    token,
			HTTPOnly: true,
			Expires:  time.Now().Add(24 * time.Hour),
			SameSite: "Lax",
		})
		return c.Redirect().Status(303).To("http://localhost:5173/accountHome")
	}
}

func Register(c fiber.Ctx) error {
	var result model.AccountRegister
	if err := c.Bind().Body(&result); err != nil {
		return c.Status(400).SendString("Failed to decode steam profile")
	}
	var email string
	err := database.DB.QueryRow("SELECT EMAIL FROM USER_ACCOUNTS WHERE EMAIL = ?", result.Email).Scan(&email)
	if err == nil {
		log.Printf("DB error checking email: %v val:%v", err, result)
		return c.Status(409).SendString("Email already in use. Please choose another.")

	}
	if err != sql.ErrNoRows {
		return c.Status(500).SendString("Something went wrong. Please try again.")
	}
	hash, err := HashPassword(result.Password)
	if err != nil {
		log.Printf("Hash error: %v", err)
		return c.Status(500).SendString("Something went wrong. Please try again.")
	}

	res, err := database.DB.Exec(
		"INSERT INTO USER_ACCOUNTS (EMAIL, USER_PASSWORD, USERNAME) VALUES (?, ?, ?)",
		result.Email, hash, result.Username,
	)
	if err != nil {
		log.Printf("Insert error: %v", err)
		return c.Status(500).SendString("Something went wrong while creating the account.")
	}

	userID, err := res.LastInsertId()
	if err != nil {
		log.Printf("LastInsertId error: %v", err)
		return c.Status(500).SendString("Something went wrong. Please try again.")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(7 * 24 * time.Hour).Unix(),
	})
	tokenString, err := token.SignedString([]byte(os.Getenv("SECURE_TOKEN")))
	if err != nil {
		log.Printf("JWT error: %v", err)
		return c.Status(500).SendString("Something went wrong. Please try again.")
	}

	c.Cookie(&fiber.Cookie{
		Name:     "auth_token",
		Value:    tokenString,
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Lax",
		MaxAge:   7 * 24 * 60 * 60,
	})

	return c.Status(200).JSON(fiber.Map{
		"message":  "Account created successfully",
		"username": result.Username,
	})
}

func Login(c fiber.Ctx) error {
	var req model.AccountRegister
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(400).SendString("Invalid request body")
	}

	var accountID int64
	var storedHash string
	var linkedSteamID sql.NullInt64
	err := database.DB.QueryRow(
		"SELECT ACCOUNTID, USER_PASSWORD, STEAMID FROM USER_ACCOUNTS WHERE EMAIL = ?",
		req.Email,
	).Scan(&accountID, &storedHash, &linkedSteamID)

	if err == sql.ErrNoRows {
		return c.Status(401).SendString("Invalid email or password")
	}
	if err != nil {
		log.Printf("Login DB error: %v", err)
		return c.Status(500).SendString("Something went wrong. Please try again.")
	}

	if !CheckPassword(req.Password, storedHash) {
		return c.Status(401).SendString("Invalid email or password")
	}

	if linkedSteamID.Valid {
		var avatarURL sql.NullString
		if dbErr := database.DB.QueryRow(
			"SELECT AVATAR_URL FROM PLAYERS WHERE PLAYERID = ?", linkedSteamID.Int64,
		).Scan(&avatarURL); dbErr == nil && !avatarURL.Valid {
			steamAPIKey := os.Getenv("STEAM_API")
			steamURL := fmt.Sprintf(
				"https://api.steampowered.com/ISteamUser/GetPlayerSummaries/v0002/?key=%s&steamids=%d",
				steamAPIKey, linkedSteamID.Int64,
			)
			client := &http.Client{Timeout: 10 * time.Second}
			if resp, steamErr := client.Get(steamURL); steamErr == nil {
				defer resp.Body.Close()
				var steamResult model.SteamResponse
				if jsonErr := json.NewDecoder(resp.Body).Decode(&steamResult); jsonErr == nil && len(steamResult.Response.Players) > 0 {
					p := steamResult.Response.Players[0]
					if _, execErr := database.DB.Exec(
						"UPDATE PLAYERS SET AVATAR_URL = ?, AVATAR_URL_FULL = ? WHERE PLAYERID = ?",
						p.Avatar, p.AvatarFull, linkedSteamID.Int64,
					); execErr != nil {
						log.Printf("avatar backfill failed for steamID %d: %v", linkedSteamID.Int64, execErr)
					}
				}
			}
		}
	}

	tokenString, err := GenerateJWT(accountID)
	if err != nil {
		log.Printf("JWT error: %v", err)
		return c.Status(500).SendString("Something went wrong. Please try again.")
	}

	c.Cookie(&fiber.Cookie{
		Name:     "auth_token",
		Value:    tokenString,
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Lax",
		MaxAge:   7 * 24 * 60 * 60,
	})

	return c.Redirect().Status(303).To("http://localhost:5173/accountHome")

}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)

	return string(bytes), err
}
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
func GenerateJWT(id int64) (string, error) {
	key := []byte(os.Getenv("SECURE_TOKEN"))
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss":     "carlos-goback-server",
		"user_id": id,
		"exp":     time.Now().Add(7 * 24 * time.Hour).Unix(),
	})
	return t.SignedString(key)
}
func GetTeamRoundStats(db *sql.DB, teamName string) (*model.TeamRoundStats, error) {
	query := `
		SELECT
			COUNT(*) AS TOTAL_ROUNDS,
			SUM(CASE WHEN r.WINNING_SIDE = tr.SIDE THEN 1 ELSE 0 END) AS ROUNDS_WON,
			SUM(CASE WHEN r.WINNING_SIDE != tr.SIDE THEN 1 ELSE 0 END) AS ROUNDS_LOST,
			SUM(CASE WHEN tr.SIDE = 2 THEN 1 ELSE 0 END) AS T_ROUNDS,
			SUM(CASE WHEN tr.SIDE = 3 THEN 1 ELSE 0 END) AS CT_ROUNDS,
			SUM(CASE WHEN tr.SIDE = 2 AND r.WINNING_SIDE = 2 THEN 1 ELSE 0 END) AS T_WINS,
			SUM(CASE WHEN r.WINNING_SIDE = 3 AND tr.SIDE = 3 THEN 1 ELSE 0 END) AS CT_WINS,
			SUM(CASE WHEN (tr.SIDE = 2 AND r.BUY_TYPE_T = 1) OR (tr.SIDE = 3 AND r.BUY_TYPE_CT = 1) THEN 1 ELSE 0 END) AS PISTOL_ROUNDS,
			SUM(CASE WHEN (tr.SIDE = 2 AND r.BUY_TYPE_T = 2) OR (tr.SIDE = 3 AND r.BUY_TYPE_CT = 2) THEN 1 ELSE 0 END) AS ECO_ROUNDS,
			SUM(CASE WHEN (tr.SIDE = 2 AND r.BUY_TYPE_T = 3) OR (tr.SIDE = 3 AND r.BUY_TYPE_CT = 3) THEN 1 ELSE 0 END) AS FORCE_ROUNDS,
			SUM(CASE WHEN (tr.SIDE = 2 AND r.BUY_TYPE_T = 4) OR (tr.SIDE = 3 AND r.BUY_TYPE_CT = 4) THEN 1 ELSE 0 END) AS FULL_BUY_ROUNDS,
			SUM(CASE WHEN r.WINNING_SIDE = tr.SIDE AND ((tr.SIDE = 2 AND r.BUY_TYPE_T = 1) OR (tr.SIDE = 3 AND r.BUY_TYPE_CT = 1)) THEN 1 ELSE 0 END) AS PISTOL_WINS,
			SUM(CASE WHEN r.WINNING_SIDE = tr.SIDE AND ((tr.SIDE = 2 AND r.BUY_TYPE_T = 2) OR (tr.SIDE = 3 AND r.BUY_TYPE_CT = 2)) THEN 1 ELSE 0 END) AS ECO_WINS,
			SUM(CASE WHEN r.WINNING_SIDE = tr.SIDE AND ((tr.SIDE = 2 AND r.BUY_TYPE_T = 3) OR (tr.SIDE = 3 AND r.BUY_TYPE_CT = 3)) THEN 1 ELSE 0 END) AS FORCE_WINS,
			SUM(CASE WHEN r.WINNING_SIDE = tr.SIDE AND ((tr.SIDE = 2 AND r.BUY_TYPE_T = 4) OR (tr.SIDE = 3 AND r.BUY_TYPE_CT = 4)) THEN 1 ELSE 0 END) AS FULL_BUY_WINS,
			SUM(CASE
				WHEN tr.SIDE = 2 AND r.BUY_TYPE_T = 4 AND r.BUY_TYPE_CT IN (2, 3) THEN 1
				WHEN tr.SIDE = 3 AND r.BUY_TYPE_CT = 4 AND r.BUY_TYPE_T IN (2, 3) THEN 1
				ELSE 0
			END) AS ANTI_ECO_ROUNDS,
			SUM(CASE
				WHEN tr.SIDE = 2 AND r.BUY_TYPE_T = 4 AND r.BUY_TYPE_CT IN (2, 3) AND r.WINNING_SIDE = tr.SIDE THEN 1
				WHEN tr.SIDE = 3 AND r.BUY_TYPE_CT = 4 AND r.BUY_TYPE_T IN (2, 3) AND r.WINNING_SIDE = tr.SIDE THEN 1
				ELSE 0
			END) AS ANTI_ECO_WINS,
			SUM(CASE WHEN tr.SIDE = 2 AND r.BOMB_PLANT = 1 THEN 1 ELSE 0 END) AS T_PLANTS,
			SUM(CASE WHEN tr.SIDE = 2 AND r.BOMB_PLANT = 1 AND r.WINNING_SIDE = 2 THEN 1 ELSE 0 END) AS T_PLANT_WINS,
			ROUND(
				SUM(CASE WHEN tr.SIDE = 2 AND r.BOMB_PLANT = 1 AND r.WINNING_SIDE = 2 THEN 1 ELSE 0 END)
				/ NULLIF(SUM(CASE WHEN tr.SIDE = 2 AND r.BOMB_PLANT = 1 THEN 1 ELSE 0 END), 0) * 100,
			1) AS T_PLANT_WIN_PCT,
			SUM(CASE WHEN tr.SIDE = 3 AND r.BOMB_PLANT = 1 THEN 1 ELSE 0 END) AS RETAKE_OPPORTUNITIES,
			SUM(CASE WHEN tr.SIDE = 3 AND r.BOMB_PLANT = 1 AND r.WINNING_SIDE = 3 THEN 1 ELSE 0 END) AS RETAKE_WINS,
			ROUND(
				SUM(CASE WHEN tr.SIDE = 3 AND r.BOMB_PLANT = 1 AND r.WINNING_SIDE = 3 THEN 1 ELSE 0 END)
				/ NULLIF(SUM(CASE WHEN tr.SIDE = 3 AND r.BOMB_PLANT = 1 THEN 1 ELSE 0 END), 0) * 100,
			1) AS RETAKE_SUCCESS_PCT
		FROM (
			SELECT DISTINCT rp.MATCHID, rp.ROUND_NO, rp.SIDE
			FROM ROUND_PARTICIPANTS rp
			JOIN MATCH_STATS ms ON ms.PLAYERID = rp.PLAYERID AND ms.MATCHID = rp.MATCHID
			WHERE ms.TEAMNAME = ?
		) tr
		JOIN ROUNDS r ON r.MATCHID = tr.MATCHID AND r.ROUND_NO = tr.ROUND_NO
	`
	s := &model.TeamRoundStats{}
	err := db.QueryRow(query, teamName).Scan(
		&s.TotalRounds, &s.RoundsWon, &s.RoundsLost,
		&s.TRounds, &s.CTRounds, &s.TWins, &s.CTWins,
		&s.PistolRounds, &s.EcoRounds, &s.ForceRounds, &s.FullBuyRounds,
		&s.PistolWins, &s.EcoWins, &s.ForceWins, &s.FullBuyWins,
		&s.AntiEcoRounds, &s.AntiEcoWins,
		&s.TPlants, &s.TPlantWins, &s.TPlantWinPct,
		&s.RetakeOpportunities, &s.RetakeWins, &s.RetakeSuccessPct,
	)
	if err != nil {
		return nil, err
	}
	return s, nil
}
func GetPlayerPageData(db *sql.DB, accountID int64) (model.PlayerPageData, error) {
	data := model.PlayerPageData{}
	err := db.QueryRow(`
        SELECT
            u.ACCOUNTID,
            u.USERNAME,
            u.STEAMID,
            u.STEAM_VER,
            p.PLAYERNAME,
            ms.TEAMNAME
        FROM USER_ACCOUNTS u
        LEFT JOIN PLAYERS p ON p.PLAYERID = u.STEAMID
        LEFT JOIN MATCH_STATS ms ON ms.PLAYERID = p.PLAYERID
            AND ms.MATCHID = (
                SELECT MATCHID FROM MATCH_STATS
                WHERE PLAYERID = p.PLAYERID
                ORDER BY MATCHID DESC LIMIT 1
            )
        WHERE u.ACCOUNTID = ?
    `, accountID).Scan(
		&data.AccountID,
		&data.Username,
		&data.SteamID,
		&data.SteamVer,
		&data.PlayerName,
		&data.TeamName,
	)
	if err != nil {
		return model.PlayerPageData{}, fmt.Errorf("getPlayerPageData: %w", err)
	}
	return data, nil
}
func GetUserIDFromCookie(c fiber.Ctx) (int64, error) {
	tokenString := c.Cookies("auth_token")
	if tokenString == "" {
		return 0, fmt.Errorf("no token")
	}

	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(os.Getenv("SECURE_TOKEN")), nil
	})
	if err != nil || !token.Valid {
		return 0, fmt.Errorf("invalid token: %w", err)
	}

	claims := token.Claims.(jwt.MapClaims)
	userID := int64(claims["user_id"].(float64))
	return userID, nil
}

func GetDemoPath() string {
	return os.Getenv("DEMO_PATH")
}
