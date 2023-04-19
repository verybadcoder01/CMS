package internal

import (
	"cms/db"
	"cms/models"
	"encoding/json"
	"github.com/gofiber/fiber/v2"
	"log"
	"net/http"
	"time"
)

type Session struct {
	login  string
	expiry time.Time
}

var sessions = map[string]Session{}

func SetupRouting(app *fiber.App) {
	app.Get("/api/shutdown", func(c *fiber.Ctx) error {
		return app.Shutdown()
	})
	app.Get("/api/signin", func(c *fiber.Ctx) error {
		var req models.SimpleModerator
		err := json.Unmarshal(c.Body(), &req)
		if err != nil {
			log.Println("cant decode json: " + err.Error())
			return c.SendStatus(http.StatusInternalServerError)
		}
		err = db.CreateModerator(req)
		if err != nil {
			log.Println("user already exists: " + err.Error())
		}
		return c.SendStatus(http.StatusOK)
	})
	app.Get("/api/login", func(c *fiber.Ctx) error {
		return c.SendStatus(http.StatusOK)
	})
}
