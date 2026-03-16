package main

import (
	// "fmt"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/template/html/v2"
	flog "github.com/gofiber/fiber/v3/log"
	demoinfocs "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	events "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
	"log"
)

func onKill(kill events.Kill) {
	var hs string
	if kill.IsHeadshot {
		hs = " (HS)"
	}

	var wallBang string
	if kill.PenetratedObjects > 0 {
		wallBang = " (WB)"
	}

	log.Printf("%s <%v%s%s> %s\n", kill.Killer, kill.Weapon, hs, wallBang, kill.Victim)
}

func main() {
	engine := html.New("./views", ".html")
	app := fiber.New(fiber.Config{
		// Provide a template engine
		BodyLimit: 1 * 1024 * 1024 * 1024,
		Views: engine,
	})
	port := ":4000"
	app.Get("/", func(c fiber.Ctx) error {
		return c.Render("index", fiber.Map{
			"Title": "Hello, World!",
	    }) 
	})

	app.Post("/testFile", func(c fiber.Ctx) error {
		file, err := c.FormFile("myfile")
		if err != nil {
			return err
		}
		flog.Debug("Successfuk")
		// Save the file to ./uploads/ directory
		err = c.SaveFile(file, "./uploads/"+file.Filename)
		if err != nil {
			return err
		}
		demo := demoinfocs.ParseFile("./uploads/"+file.Filename, func(p demoinfocs.Parser) error {
			p.RegisterEventHandler(onKill)

			return nil
		})
		if demo != nil {
			log.Panic("failed to parse demo: ", demo)
		}
		return c.Render("index", fiber.Map{
			"Title": "File uploaded successfully!",
	    }) 
	})
	app.Listen(port)
}
