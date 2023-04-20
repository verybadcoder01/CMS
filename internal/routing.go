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
	"strconv"
	"time"
)

type Session struct {
	login  string
	expiry time.Time
}

const AuthCookieName = "auth_cookie"
const SessionExpiryTime = 6 * time.Hour
const CookieExpiryTime = 24 * time.Hour

var sessions = map[string]Session{}

func SetupRouting(app *fiber.App) {
	app.Get("/api/shutdown", func(c *fiber.Ctx) error {
		return app.Shutdown()
	})
	app.Get("/api/inner/register_admin", func(c *fiber.Ctx) error {
		res := CookieAuthCheck(c)
		switch res {
		case http.StatusUnauthorized:
			return c.Status(http.StatusUnauthorized).SendString("session has expired")
		case http.StatusInternalServerError:
			return c.Status(http.StatusInternalServerError).SendString("cookie has expired")
		case http.StatusBadRequest:
			return c.Status(http.StatusBadRequest).SendString("user not authorized")
		}
		var req models.SimpleModerator
		err := json.Unmarshal(c.Body(), &req)
		if err != nil {
			log.Println("cant decode json: " + err.Error())
			return c.Status(http.StatusBadRequest).SendString("invalid json body")
		}
		req.Password = HashPassword(req.Password)
		err = db.CreateModerator(req)
		if err != nil {
			log.Println("moderator already exists: " + err.Error())
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
		expTime := time.Now().Add(SessionExpiryTime)
		if models.ISDEBUG == true {
			expTime = time.Now().Add(time.Minute)
		}
		sessions[sessionToken] = Session{
			login:  req.Login,
			expiry: expTime,
		}
		cookie := new(fiber.Cookie)
		cookie.Name = AuthCookieName
		cookie.Value = sessionToken
		cookie.Expires = time.Now().Add(CookieExpiryTime)
		if models.ISDEBUG == true {
			cookie.Expires = time.Now().Add(time.Hour)
		}
		c.Cookie(cookie)
		return c.Status(http.StatusOK).SendString("login successful")
	})
	app.Get("/api/admins/home", func(c *fiber.Ctx) error {
		res := CookieAuthCheck(c)
		switch res {
		case http.StatusUnauthorized:
			return c.Status(http.StatusUnauthorized).SendString("session has expired")
		case http.StatusInternalServerError:
			return c.Status(http.StatusInternalServerError).SendString("cookie has expired")
		case http.StatusBadRequest:
			return c.Status(http.StatusBadRequest).SendString("user not authorized")
		default:
			return c.Status(http.StatusOK).SendString("welcome")
		}
	})
	app.Get("/api/users/groups/+", func(c *fiber.Ctx) error {
		group, _ := strconv.Atoi(c.Params("+"))
		contests, err := db.GetContestsInGroup(group)
		if err != nil {
			log.Println(err.Error())
			return c.Status(http.StatusInternalServerError).SendString("could not retrieve data")
		}
		// тут мы руками сделаем жсон, потому что я не хочу сериализовывать однотипный массив
		res := "{ \"contests\": ["
		var concat string
		for _, contest := range contests {
			concat += contest
			concat += ","
		}
		if len(concat) != 0 {
			res += concat[:len(concat)-1]
		}
		res += "]}"
		_, err = c.Response().BodyWriter().Write([]byte(res))
		if err != nil {
			return c.Status(http.StatusInternalServerError).SendString(err.Error())
		}
		return nil
	})
	app.Get("/api/inner/contests/+", func(c *fiber.Ctx) error {
		contest, _ := strconv.Atoi(c.Params("+"))
		info, err := db.GetContestInfo(contest)
		if err != nil {
			log.Println(err.Error())
			return c.Status(http.StatusInternalServerError).SendString(err.Error())
		}
		j, err := json.Marshal(info)
		if err != nil {
			log.Println(err.Error())
			return c.Status(http.StatusInternalServerError).SendString(err.Error())
		}
		_, err = c.Response().BodyWriter().Write(j)
		if err != nil {
			log.Println(err.Error())
			return c.Status(http.StatusInternalServerError).SendString(err.Error())
		}
		return nil
	})
	app.Get("/api/admins/+/create_contest", func(c *fiber.Ctx) error {
		var newContest models.BasicContest
		err := json.Unmarshal(c.Body(), &newContest)
		if err != nil {
			log.Println("can't unmarshall json " + err.Error())
			return c.Status(http.StatusBadRequest).SendString("can't unmarshall json")
		}
		group, _ := strconv.Atoi(c.Params("+"))
		res := CookieAuthCheck(c)
		switch res {
		case http.StatusUnauthorized:
			return c.Status(http.StatusUnauthorized).SendString("session has expired")
		case http.StatusInternalServerError:
			return c.Status(http.StatusInternalServerError).SendString("cookie has expired")
		case http.StatusBadRequest:
			return c.Status(http.StatusBadRequest).SendString("user not authorized")
		}
		// да, я повторил два действия из проверки авторизации. Что ты мне сделаешь?
		token := c.Cookies(AuthCookieName, "-1")
		session, _ := sessions[token]
		if db.IsHostInGroup(group, session.login) {
			err = db.AddContest(newContest)
			if err != nil {
				log.Println("can't create contest " + err.Error())
				return c.Status(http.StatusInternalServerError).SendString("can't create contest")
			}
			id, err := db.GetContestId(newContest.Url, newContest.StatementsUrl, newContest.Name)
			err = db.AddContestToGroup(group, id)
			if err != nil {
				log.Println("can't add contest to group " + err.Error())
				return c.Status(http.StatusInternalServerError).SendString("can't create contest")
			}
			return c.Status(http.StatusOK).SendString("successful")
		}
		return c.Status(http.StatusForbidden).SendString("you are not host")
	})
	app.Get("/api/admins/create_group", func(c *fiber.Ctx) error {
		res := CookieAuthCheck(c)
		switch res {
		case http.StatusUnauthorized:
			return c.Status(http.StatusUnauthorized).SendString("session has expired")
		case http.StatusInternalServerError:
			return c.Status(http.StatusInternalServerError).SendString("cookie has expired")
		case http.StatusBadRequest:
			return c.Status(http.StatusBadRequest).SendString("user not authorized")
		}
		token := c.Cookies(AuthCookieName, "-1")
		session, _ := sessions[token]
		var newGroup models.Group
		err := json.Unmarshal(c.Body(), &newGroup)
		if err != nil {
			log.Println("can't parse json " + err.Error())
			return c.Status(http.StatusBadRequest).SendString("can't parse json")
		}
		err = db.AddGroup(newGroup)
		if err != nil {
			log.Println(err.Error())
			return c.Status(http.StatusInternalServerError).SendString("failed to create new group")
		}
		id, _ := db.GetGroupId(newGroup.Name)
		moderatorId, err := db.GetModeratorId(session.login)
		if err != nil {
			log.Println(err.Error())
			return c.Status(http.StatusInternalServerError).SendString("failed to create new group")
		}
		err = db.AddHostToGroup(id, moderatorId)
		if err != nil {
			log.Println(err.Error())
			return c.Status(http.StatusInternalServerError).SendString("failed to create new group")
		}
		return c.Status(http.StatusOK).SendString("successful")
	})
}
