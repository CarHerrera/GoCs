package middleware

import (
	database "server/DB"
	auth "server/auth"

	"github.com/gofiber/fiber/v3"
)

// RequireAuth validates the session cookie and stashes the accountID
// in Locals for downstream handlers.
func RequireAuth(c fiber.Ctx) error {
	accountID, err := auth.GetUserIDFromCookie(c)
	if err != nil {
		return c.Status(401).SendString("Unauthorized")
	}
	c.Locals("accountID", accountID)
	return c.Next()
}

// RequireTeam runs after RequireAuth. It resolves the player/team chain
// once and stashes it, so team handlers don't repeat the lookup + guard.
func RequireTeam(c fiber.Ctx) error {
	accountID := c.Locals("accountID").(int64)
	data, err := auth.GetPlayerPageData(database.DB, accountID)
	if err != nil || !data.TeamName.Valid {
		return c.Status(400).JSON(fiber.Map{"error": "No team linked"})
	}
	c.Locals("playerData", data)
	return c.Next()
}
