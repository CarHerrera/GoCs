package player

import (
	"server/middleware"

	"github.com/gofiber/fiber/v3"
)

func RegisterRoutes(app *fiber.App) {
	g := app.Group("/api/player")
	// Only RequireAuth — /me must work for a user with no team yet.
	g.Use(middleware.RequireAuth)

	g.Get("/me", Me)
	g.Get("/me/stats", MeStats)
	g.Post("/link-steam", LinkSteam)
	g.Put("/:id/role", UpdateRole)
}
