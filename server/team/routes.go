package team

import (
	"server/middleware"

	"github.com/gofiber/fiber/v3"
)

func RegisterRoutes(app *fiber.App) {
	g := app.Group("/api/team")
	g.Use(middleware.RequireAuth)
	g.Get("/summary", Summary)
	g.Get("/advanced", Advanced)
	g.Get("/PlayerStats", PlayerStats)
	g.Get("/info", Info)
	g.Get("/Logo", Logo)
	g.Get("/demos", Demos)
	g.Post("/upload-demo", UploadDemo)

}
