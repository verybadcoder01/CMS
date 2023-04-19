package internal

import (
	"cms/db"
	"cms/models"
	"encoding/json"
	"errors"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"log"
	"net/http"
	"time"
)

type Session struct {
	login  string
	expiry time.Time
}

const AUTH_COOKIE_NAME = "auth_cookie"
const ExpiryTime = 60 * time.Second

var sessions = map[string]Session{}

func SetupRouting(app *fiber.App) {
	app.Get("/api/shutdown", func(c *fiber.Ctx) error {
		return app.Shutdown()
	})
	app.Get("/api/admins/signin", func(c *fiber.Ctx) error {
		var req models.SimpleModerator
		err := json.Unmarshal(c.Body(), &req)
		if err != nil {
			log.Println("cant decode json: " + err.Error())
			return c.Status(http.StatusBadRequest).SendString("invalid json body")
		}
		req.Password = HashPassword(req.Password)
		err = db.CreateModerator(req)
		if err != nil {
			log.Println("user already exists: " + err.Error())
			return c.Status(http.StatusBadRequest).SendString("user with this login already exists")
		}
		return c.SendStatus(http.StatusOK)
	})
	app.Get("/api/admins/login", func(c *fiber.Ctx) error {
		var req models.SimpleModerator
		err := json.Unmarshal(c.Body(), &req)
		if err != nil {
			log.Println("can't decode json: " + err.Error())
			return c.Status(http.StatusBadRequest).SendString("invalid json body")
		}
		expected, err := db.GetPasswordHash(req.Login)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("user with login %v not found", req.Login)
			return c.Status(http.StatusBadRequest).SendString("user not found)")
		}
		if CheckPasswordHash(expected, HashPassword(req.Password)) {
			return c.Status(http.StatusBadRequest).SendString("wrong password")
		}
		sessionToken := uuid.NewString()
		expTime := time.Now().Add(ExpiryTime)
		sessions[sessionToken] = Session{
			login:  req.Login,
			expiry: expTime,
		}
		cookie := new(fiber.Cookie)
		cookie.Name = AUTH_COOKIE_NAME
		cookie.Value = sessionToken
		cookie.Expires = expTime
		c.Cookie(cookie)
		return c.Status(http.StatusOK).SendString("login successful")
	})
	app.Get("/api/admins/access", func(c *fiber.Ctx) error {
		token := c.Cookies(AUTH_COOKIE_NAME, "-1")
		if token == "-1" {
			log.Println("cookie not found on access route")
			return c.Status(http.StatusInternalServerError).SendString("idk some session shit")
		}
		session, exists := sessions[token]
		if !exists {
			log.Printf("user not authorized on access route, session %v", token)
			return c.Status(http.StatusBadRequest).SendString("session not founc")
		}
		if session.isExpired() {
			log.Printf("session %v has expired", session)
			delete(sessions, token)
			return c.Status(http.StatusUnauthorized).SendString("session has expired")
		}
		return c.Status(http.StatusOK).SendString("welcome")
	})
}
