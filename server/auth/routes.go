package auth

import "github.com/gofiber/fiber/v3"

// Auth routes are public — they establish the session, so no middleware.
func RegisterRoutes(app *fiber.App) {
	app.Get("/auth/steam", SteamLogin)
	app.Get("/auth/steam/callback", SteamCallback)
	app.Post("/auth/register", Register)
	app.Post("/auth/login", Login)
}
