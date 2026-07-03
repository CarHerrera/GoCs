package season

import (
	"server/middleware"

	"github.com/gofiber/fiber/v3"
)

func RegisterRoutes(app *fiber.App) {
	g := app.Group("/api/season")
	g.Use(middleware.RequireAuth)

	g.Get("/:id/matches", Matches)
}
