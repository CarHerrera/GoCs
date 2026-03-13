package main

import (
	"fmt"

	"github.com/gofiber/fiber/v3"
)

func main() {
	app := fiber.New()
	port := ":4000"
	app.Get("/", func(c fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})

	app.Listen(port)
	fmt.Printf("Server Started at port %s", port[1:])
}
