package internal

import (
	"cms/db"
	"cms/models"
	"encoding/json"
	"github.com/gofiber/fiber/v2"
	"log"
	"net/http"
)

func SetupRouting(app *fiber.App) {
	app.Get("/api/shutdown", func(c *fiber.Ctx) error {
		return app.Shutdown()
	})
	app.Get("/api/create_contest", func(c *fiber.Ctx) error {
		body := c.Body()
		var contest models.BasicContest
		err := json.Unmarshal(body, &contest)
		if err != nil {
			log.Println("can't unmarshall json :" + err.Error())
			return c.SendStatus(http.StatusInternalServerError)
		}
		err = db.AddContest(contest)
		if err != nil {
			log.Println(err.Error())
			return c.SendStatus(http.StatusInternalServerError)
		}
		return c.SendStatus(http.StatusOK)
	})
	app.Get("/api/create_user", func(c *fiber.Ctx) error {
		body := c.Body()
		var user models.User
		err := json.Unmarshal(body, &user)
		if err != nil {
			log.Println("can't unmarshall json :" + err.Error())
			return c.SendStatus(http.StatusInternalServerError)
		}
		err = db.CreateUser(user)
		if err != nil {
			log.Println(err.Error())
			return c.SendStatus(http.StatusInternalServerError)
		}
		return c.SendStatus(http.StatusOK)
	})
	app.Get("/api/add_user_to_group", func(c *fiber.Ctx) error {
		body := c.Body()
		var data models.UserAndGroup
		err := json.Unmarshal(body, &data)
		if err != nil {
			log.Println("can't unmarshall json :" + err.Error())
			return c.SendStatus(http.StatusInternalServerError)
		}
		err = db.AddUserToGroup(data.GroupId, data.UserId, data.Role)
		if err != nil {
			log.Println(err.Error())
			return c.SendStatus(http.StatusInternalServerError)
		}
		return c.SendStatus(http.StatusOK)
	})
}
