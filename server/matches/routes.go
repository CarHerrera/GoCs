package matches

import (
	"server/middleware"

	"github.com/gofiber/fiber/v3"
)

func RegisterRoutes(app *fiber.App) {
	g := app.Group("/api/match")
	g.Use(middleware.RequireAuth)
	g.Get("/2DPlayback/:matchRound", TwoDPlayBack)
	g.Get("/advancedStats/:matchId", AdvancedStats)
}
