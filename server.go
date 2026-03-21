package main

import (
	"os"
	"fmt"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/template/html/v2"
	"github.com/gofiber/fiber/v3/middleware/static"
	flog "github.com/gofiber/fiber/v3/log"
	dem "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	events "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
	"log"
	"strings"
	"html/template"
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
var downloaded string = "/Users/carlosherrera/Documents/CS2DEMOS"
func main() {
	engine := html.New("./views", ".html")
	engine.AddFunc(
        // add unescape function
        "unescape", func(s string) template.HTML {
            return template.HTML(s)
        },
    )
	app := fiber.New(fiber.Config{
		// Provide a template engine
		BodyLimit: 1 * 1024 * 1024 * 1024,
		Views: engine,
	})
	
	app.Use("/static", static.New("./static"))
	port := ":4000"
	app.Get("/", func(c fiber.Ctx) error {
		return c.Render("index", fiber.Map{
			"Title": "Hello, World!",
	    }) 
	})
	app.Get("/demoList", func(c fiber.Ctx) error {
		entries, err := os.ReadDir(downloaded)
		if err != nil {
			log.Fatal(err)
		}
		var demoRow strings.Builder
		for _, e := range entries {
			info, _ := e.Info()
			link := "/advancedStats/"+ info.Name()
			demoRow.WriteString(fmt.Sprintf("<tr><td>%s</td><td>%s</td><td><a href=\"%s\">Stats</a></td></tr>", e.Name(), info.ModTime(), link))
		}
		return c.Render("demoList", fiber.Map{
			"DemoContent": demoRow.String(),
	    }) 
	})
	app.Get("/advancedStats/:demoName", func(c fiber.Ctx) error {
		path := downloaded+ "/" + c.Params("demoName")
		file, _ := os.Open(path)
		p := dem.NewParser(file)
		head := p.ParseHeader()
		Map := head.MapName()
		defer p.Close()
		defer file.Close()
		return c.Render("stats", fiber.Map{
			"FileName": Map,
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
		demo := dem.ParseFile("./uploads/"+file.Filename, func(p dem.Parser) error {
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
