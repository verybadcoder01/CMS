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

var SessionExpiryTime = 6 * time.Hour
var CookieExpiryTime = 24 * time.Hour

var sessions = map[string]Session{}

func SetupRouting(app *fiber.App) {
	app.Get("/", func(c *fiber.Ctx) error {
		return c.Redirect("/api/users/groups")
	})
	// требует, чтобы ты был залогинен как дефолтный модератор
	app.Post("/api/shutdown", func(c *fiber.Ctx) error {
		res := CookieAuthCheck(c)
		switch res {
		case http.StatusUnauthorized:
			return c.Status(http.StatusUnauthorized).SendString("session has expired")
		case http.StatusForbidden:
			return c.Status(http.StatusForbidden).SendString("cookie has expired or never existed")
		case http.StatusBadRequest:
			return c.Status(http.StatusBadRequest).SendString("user not authorized")
		}
		token := c.Cookies(AuthCookieName, "-1")
		session, _ := sessions[token]
		id, _ := db.GetModeratorId(session.login)
		if id != 1 {
			return c.Status(http.StatusForbidden).SendString("must be number 1 moderator to do this")
		}
		return app.Shutdown()
	})
	app.Post("/api/inner/register_admin", func(c *fiber.Ctx) error {
		res := CookieAuthCheck(c)
		switch res {
		case http.StatusUnauthorized:
			return c.Status(http.StatusUnauthorized).SendString("session has expired")
		case http.StatusForbidden:
			return c.Status(http.StatusForbidden).SendString("cookie has expired or never existed")
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
	app.Post("/api/admins/login", func(c *fiber.Ctx) error {
		var req models.SimpleModerator
		err := json.Unmarshal(c.Body(), &req)
		if err != nil {
			log.Println("can't decode json: " + err.Error())
			return c.Status(http.StatusBadRequest).SendString("invalid json body")
		}
		expected, err := db.GetPasswordHash(req.Login)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("user with login %v not found", req.Login)
			return c.Status(http.StatusBadRequest).SendString("user not found")
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
		case http.StatusForbidden:
			return c.Status(http.StatusForbidden).SendString("cookie has expired or never existed")
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
	app.Post("/api/admins/create_contest", func(c *fiber.Ctx) error {
		var newContest models.BasicContest
		err := json.Unmarshal(c.Body(), &newContest)
		if err != nil {
			log.Println("can't unmarshall json " + err.Error())
			return c.Status(http.StatusBadRequest).SendString("can't unmarshall json")
		}
		group := c.GetReqHeaders()["Group"]
		res := CookieAuthCheck(c)
		switch res {
		case http.StatusUnauthorized:
			return c.Status(http.StatusUnauthorized).SendString("session has expired")
		case http.StatusForbidden:
			return c.Status(http.StatusForbidden).SendString("cookie has expired or never existed")
		case http.StatusBadRequest:
			return c.Status(http.StatusBadRequest).SendString("user not authorized")
		}
		// да, я повторил два действия из проверки авторизации. Что ты мне сделаешь?
		token := c.Cookies(AuthCookieName, "-1")
		session, _ := sessions[token]
		id, err := db.GetModeratorId(session.login)
		if err != nil {
			log.Println(err.Error())
			return c.Status(http.StatusInternalServerError).SendString("unknown error")
		}
		groupId, err := db.GetGroupId(group)
		if err != nil {
			log.Println(err.Error())
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return c.Status(http.StatusBadRequest).SendString("no such group")
			} else {
				return c.Status(http.StatusInternalServerError).SendString("couldn't create contest")
			}
		}
		if db.IsHostInGroup(groupId, id) {
			err = db.AddContest(newContest)
			if err != nil {
				log.Println("can't create contest " + err.Error())
				return c.Status(http.StatusInternalServerError).SendString("can't create contest")
			}
			id, err := db.GetContestId(newContest.Name)
			err = db.AddContestToGroup(groupId, id)
			if err != nil {
				log.Println("can't add contest to group " + err.Error())
				return c.Status(http.StatusInternalServerError).SendString("can't create contest")
			}
			return c.Status(http.StatusOK).SendString("successful")
		}
		return c.Status(http.StatusForbidden).SendString("you are not host")
	})
	app.Post("/api/admins/create_group", func(c *fiber.Ctx) error {
		res := CookieAuthCheck(c)
		switch res {
		case http.StatusUnauthorized:
			return c.Status(http.StatusUnauthorized).SendString("session has expired")
		case http.StatusForbidden:
			return c.Status(http.StatusForbidden).SendString("cookie has expired or never existed")
		case http.StatusBadRequest:
			return c.Status(http.StatusBadRequest).SendString("user not authorized")
		}
		token := c.Cookies(AuthCookieName, "-1")
		session, _ := sessions[token]
		var newGroup models.BasicGroup
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
		err = db.AddHostToGroup(id, 1) // дефолтный админ
		if err != nil {
			log.Println(err.Error())
			return c.Status(http.StatusInternalServerError).SendString("failed to create new group")
		}
		return c.Status(http.StatusOK).SendString("successful")
	})
	app.Post("/api/admins/logout", func(c *fiber.Ctx) error {
		res := CookieAuthCheck(c)
		switch res {
		case http.StatusUnauthorized:
			return c.Status(http.StatusUnauthorized).SendString("session has expired")
		case http.StatusForbidden:
			return c.Status(http.StatusForbidden).SendString("cookie has expired or never existed")
		case http.StatusBadRequest:
			return c.Status(http.StatusBadRequest).SendString("user not authorized")
		}
		token := c.Cookies(AuthCookieName, "-1")
		delete(sessions, token)
		c.ClearCookie(AuthCookieName)
		return c.Status(http.StatusOK).SendString("logout successful")
	})
	app.Get("/api/users/groups", func(c *fiber.Ctx) error {
		groups, err := db.GetGroups()
		if err != nil {
			log.Println(err.Error())
			return c.Status(http.StatusInternalServerError).SendString("couldn't retrieve groups")
		}
		res, _ := json.Marshal(groups)
		_, err = c.Response().BodyWriter().Write(res)
		if err != nil {
			log.Println(err.Error())
			return c.Status(http.StatusInternalServerError).SendString("couldn't retrieve groups")
		}
		return nil
	})
	app.Post("/api/admins/give_host", func(c *fiber.Ctx) error {
		res := CookieAuthCheck(c)
		switch res {
		case http.StatusUnauthorized:
			return c.Status(http.StatusUnauthorized).SendString("session has expired")
		case http.StatusForbidden:
			return c.Status(http.StatusForbidden).SendString("cookie has expired or never existed")
		case http.StatusBadRequest:
			return c.Status(http.StatusBadRequest).SendString("user not authorized")
		}
		token := c.Cookies(AuthCookieName, "-1")
		session, _ := sessions[token]
		id, err := db.GetModeratorId(session.login)
		if err != nil {
			log.Println(err.Error())
			return c.Status(http.StatusInternalServerError).SendString("unable to get your id")
		}
		var req models.GroupAndHost
		err = json.Unmarshal(c.Body(), &req)
		if err != nil {
			log.Println("can't parse json " + err.Error())
			return c.Status(http.StatusInternalServerError).SendString("unable to give host")
		}
		if db.IsHostInGroup(req.GroupId, id) {
			id, err := db.GetModeratorId(req.ModeratorId)
			if errors.Is(err, gorm.ErrRecordNotFound) {
				log.Println(err.Error())
				return c.Status(http.StatusBadRequest).SendString("no such moderator")
			} else if err != nil {
				log.Println(err.Error())
				return c.Status(http.StatusInternalServerError).SendString("unable to give host")
			}
			err = db.AddHostToGroup(req.GroupId, id)
			if err != nil {
				log.Println(err.Error())
				return c.Status(http.StatusInternalServerError).SendString("unable to give host")
			}
		} else {
			return c.Status(http.StatusForbidden).SendString("you are not authorized as group host")
		}
		return c.Status(http.StatusOK).SendString("success")
	})
	app.Post("/api/admins/edit_contest", func(c *fiber.Ctx) error {
		res := CookieAuthCheck(c)
		switch res {
		case http.StatusUnauthorized:
			return c.Status(http.StatusUnauthorized).SendString("session has expired")
		case http.StatusForbidden:
			return c.Status(http.StatusForbidden).SendString("cookie has expired or never existed")
		case http.StatusBadRequest:
			return c.Status(http.StatusBadRequest).SendString("user not authorized")
		}
		var contest models.BasicContest
		err := json.Unmarshal(c.Body(), &contest)
		if err != nil {
			log.Println("can't parse json " + err.Error())
			return c.Status(http.StatusBadRequest).SendString("invalid request body")
		}
		token := c.Cookies(AuthCookieName, "-1")
		session, _ := sessions[token]
		modId, err := db.GetModeratorId(session.login)
		if err != nil {
			log.Println(err.Error())
			return c.Status(http.StatusInternalServerError).SendString("unknown error")
		}
		contestName := c.GetReqHeaders()["Contest"]
		id, err := db.GetContestId(contestName)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Println("no such contest " + err.Error())
			return c.Status(http.StatusBadRequest).SendString("no such contest")
		} else if err != nil {
			log.Println(err.Error())
			return c.Status(http.StatusInternalServerError).SendString("unable to edit contest")
		}
		group, err := db.GetGroupByContest(id)
		if err != nil {
			log.Println(err.Error())
			return c.Status(http.StatusInternalServerError).SendString("unknown error")
		}
		if db.IsHostInGroup(group, modId) {
			err = db.EditContest(id, contest)
			if err != nil {
				log.Println(err.Error())
				return c.Status(http.StatusInternalServerError).SendString("unable to edit contest")
			}
		}
		return c.Status(http.StatusOK).SendString("successful")
	})
	app.Post("/api/admins/remove_host", func(c *fiber.Ctx) error {
		res := CookieAuthCheck(c)
		switch res {
		case http.StatusUnauthorized:
			return c.Status(http.StatusUnauthorized).SendString("session has expired")
		case http.StatusForbidden:
			return c.Status(http.StatusForbidden).SendString("cookie has expired or never existed")
		case http.StatusBadRequest:
			return c.Status(http.StatusBadRequest).SendString("user not authorized")
		}
		token := c.Cookies(AuthCookieName, "-1")
		session, _ := sessions[token]
		id, err := db.GetModeratorId(session.login)
		if err != nil {
			log.Println(err.Error())
			return c.Status(http.StatusInternalServerError).SendString("unable to get your id")
		}
		var req models.GroupAndHost
		err = json.Unmarshal(c.Body(), &req)
		if err != nil {
			log.Println("can't parse json " + err.Error())
			return c.Status(http.StatusInternalServerError).SendString("unable to remove host")
		}
		reqModId, err := db.GetModeratorId(req.ModeratorId)
		if db.IsHostInGroup(req.GroupId, id) && db.IsHostInGroup(req.GroupId, reqModId) && reqModId != 1 {
			err = db.RemoveModeratorInGroup(req.GroupId, reqModId)
			if err != nil {
				log.Println(err.Error())
				return c.Status(http.StatusInternalServerError).SendString("could not remove host")
			}
		} else {
			return c.Status(http.StatusForbidden).SendString("you lack permission to do that")
		}
		return c.Status(http.StatusOK).SendString("success")
	})
}
