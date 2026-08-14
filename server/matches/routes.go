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
	g.Get("/entries/:matchId", EntryStats)
	g.Get("/utility/:matchId", UtilityStats)
	g.Get("/plantsRetakes/:matchId", PlantsAndRetakes)
	g.Get("/roundResults/:matchId", RoundResults)
	g.Get("/clutches/:matchId", ClutchStats)
}
