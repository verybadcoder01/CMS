package internal

import (
	"cms/config"
	"cms/db"
	"cms/models"
	"encoding/json"
	"errors"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
	"unicode/utf8"
)

type Session struct {
	login  string
	expiry time.Time
}

var SessionExpiryTime = 6 * time.Hour

var sessions sync.Map

func login(c *fiber.Ctx) error {
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
	if !CheckPasswordHash(req.Password, expected) {
		return c.Status(http.StatusForbidden).SendString("wrong password")
	}
	sessionToken := uuid.NewString()
	expTime := time.Now().Add(SessionExpiryTime)
	if config.ISDEBUG == true {
		expTime = time.Now().Add(time.Minute)
	}
	session := Session{login: req.Login, expiry: expTime}
	sessions.Store(sessionToken, session)
	var cookie models.SessionInfo
	cookie.Session = sessionToken
	res, err := json.Marshal(cookie)
	if err != nil {
		log.Println(err.Error())
		return c.Status(http.StatusInternalServerError).SendString("can't marshall response")
	}
	_, err = c.Response().BodyWriter().Write(res)
	if err != nil {
		log.Println(err.Error())
		return c.Status(http.StatusInternalServerError).SendString(err.Error())
	}
	log.Printf("send session %v", cookie)
	return nil
}

func logout(c *fiber.Ctx) error {
	res := SessionAuthCheck(c)
	switch res {
	case http.StatusUnauthorized:
		return c.Status(http.StatusUnauthorized).SendString("session has expired")
	case http.StatusForbidden:
		return c.Status(http.StatusForbidden).SendString("user not authorized")
	case http.StatusBadRequest:
		return c.Status(http.StatusBadRequest).SendString("header doesn't have session key")
	}
	token := c.GetReqHeaders()["Session"]
	sessions.Delete(token)
	return c.Status(http.StatusOK).SendString("logout successful")
}

func SetupRouting(app *fiber.App) {
	app.Get("/", func(c *fiber.Ctx) error {
		return c.Redirect("/api/users/groups")
	})
	// требует, чтобы ты был залогинен как дефолтный модератор
	app.Post("/api/shutdown", func(c *fiber.Ctx) error {
		res := SessionAuthCheck(c)
		switch res {
		case http.StatusUnauthorized:
			return c.Status(http.StatusUnauthorized).SendString("session has expired")
		case http.StatusForbidden:
			return c.Status(http.StatusForbidden).SendString("user not authorized")
		case http.StatusBadRequest:
			return c.Status(http.StatusBadRequest).SendString("header doesn't have session key")
		}
		token := c.GetReqHeaders()["Session"]
		session, _ := sessions.Load(token)
		id, _ := db.GetModeratorId(session.(Session).login)
		if id != 1 {
			return c.Status(http.StatusForbidden).SendString("must be number 1 moderator to do this")
		}
		return app.Shutdown()
	})
	app.Post("/api/inner/register_admin", func(c *fiber.Ctx) error {
		res := SessionAuthCheck(c)
		switch res {
		case http.StatusUnauthorized:
			return c.Status(http.StatusUnauthorized).SendString("session has expired")
		case http.StatusForbidden:
			return c.Status(http.StatusForbidden).SendString("user not authorized")
		case http.StatusBadRequest:
			return c.Status(http.StatusBadRequest).SendString("header doesn't have session key")
		}
		var req models.SimpleModerator
		err := json.Unmarshal(c.Body(), &req)
		if err != nil {
			log.Println("cant decode json: " + err.Error())
			return c.Status(http.StatusBadRequest).SendString("invalid json body")
		}
		if utf8.RuneCountInString(req.Password)*4 > 72 {
			log.Println("password too large, rejecting")
			return c.Status(http.StatusBadRequest).SendString("password too big >72 bytes")
		}
		req.Password = HashPassword(req.Password)
		isOk := db.CreateModerator(req)
		if isOk == false {
			log.Println("moderator with this login already exists " + req.Login)
			return c.Status(http.StatusBadRequest).SendString("moderator with this login already exists")
		}
		return c.SendStatus(http.StatusOK)
	})
	app.Post("/api/admins/login", login)
	app.Get("/api/admins/home", func(c *fiber.Ctx) error {
		res := SessionAuthCheck(c)
		switch res {
		case http.StatusUnauthorized:
			return c.Status(http.StatusUnauthorized).SendString("session has expired")
		case http.StatusForbidden:
			return c.Status(http.StatusForbidden).SendString("user not authorized")
		case http.StatusBadRequest:
			return c.Status(http.StatusBadRequest).SendString("header doesn't have session key")
		default:
			var t models.TimeJson
			session, _ := c.GetReqHeaders()["Session"]
			info, exists := sessions.Load(session)
			if exists {
				t.UntilExpires = info.(Session).untilExpiration()
			}
			b, err := json.Marshal(t)
			if err != nil {
				log.Println(err.Error())
				return c.Status(http.StatusInternalServerError).SendString("can't marshall response")
			}
			_, err = c.Response().BodyWriter().Write(b)
			if err != nil {
				log.Println(err.Error())
				return c.Status(http.StatusInternalServerError).SendString("can't marshall response")
			}
			return nil
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
			return c.Status(http.StatusBadRequest).SendString("invalid json body")
		}
		group, _ := url.QueryUnescape(c.GetReqHeaders()["Group"])
		res := SessionAuthCheck(c)
		switch res {
		case http.StatusUnauthorized:
			return c.Status(http.StatusUnauthorized).SendString("session has expired")
		case http.StatusForbidden:
			return c.Status(http.StatusForbidden).SendString("user not authorized")
		case http.StatusBadRequest:
			return c.Status(http.StatusBadRequest).SendString("header doesn't have session key")
		}
		token := c.GetReqHeaders()["Session"]
		session, _ := sessions.Load(token)
		id, err := db.GetModeratorId(session.(Session).login)
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
			if errors.Is(err, gorm.ErrInvalidData) {
				log.Println("this contest already exists " + newContest.Name)
				return c.Status(http.StatusBadRequest).SendString("this contest already exists")
			}
			if err != nil {
				log.Println("can't create contest " + err.Error())
				return c.Status(http.StatusInternalServerError).SendString("can't create contest")
			}
			id, err := db.GetContestId(newContest)
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
		res := SessionAuthCheck(c)
		switch res {
		case http.StatusUnauthorized:
			return c.Status(http.StatusUnauthorized).SendString("session has expired")
		case http.StatusForbidden:
			return c.Status(http.StatusForbidden).SendString("user not authorized")
		case http.StatusBadRequest:
			return c.Status(http.StatusBadRequest).SendString("header doesn't have session key")
		}
		token := c.GetReqHeaders()["Session"]
		session, _ := sessions.Load(token)
		var newGroup models.BasicGroup
		err := json.Unmarshal(c.Body(), &newGroup)
		if err != nil {
			log.Println("can't parse json " + err.Error())
			return c.Status(http.StatusBadRequest).SendString("invalid json body")
		}
		err = db.AddGroup(newGroup)
		if err != nil {
			log.Println(err.Error())
			return c.Status(http.StatusBadRequest).SendString("name not unique")
		}
		id, _ := db.GetGroupId(newGroup.Name)
		moderatorId, err := db.GetModeratorId(session.(Session).login)
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
	app.Post("/api/admins/logout", logout)
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
		res := SessionAuthCheck(c)
		switch res {
		case http.StatusUnauthorized:
			return c.Status(http.StatusUnauthorized).SendString("session has expired")
		case http.StatusForbidden:
			return c.Status(http.StatusForbidden).SendString("user not authorized")
		case http.StatusBadRequest:
			return c.Status(http.StatusBadRequest).SendString("header doesn't have session key")
		}
		token := c.GetReqHeaders()["Session"]
		session, _ := sessions.Load(token)
		id, err := db.GetModeratorId(session.(Session).login)
		if err != nil {
			log.Println(err.Error())
			return c.Status(http.StatusInternalServerError).SendString("unable to get your id")
		}
		var req models.GroupAndHost
		err = json.Unmarshal(c.Body(), &req)
		if err != nil {
			log.Println("can't parse json " + err.Error())
			return c.Status(http.StatusBadRequest).SendString("invalid json body")
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
		res := SessionAuthCheck(c)
		switch res {
		case http.StatusUnauthorized:
			return c.Status(http.StatusUnauthorized).SendString("session has expired")
		case http.StatusForbidden:
			return c.Status(http.StatusForbidden).SendString("user not authorized")
		case http.StatusBadRequest:
			return c.Status(http.StatusBadRequest).SendString("header doesn't have session key")
		}
		var contest models.BasicContest
		err := json.Unmarshal(c.Body(), &contest)
		if err != nil {
			log.Println("can't parse json " + err.Error())
			return c.Status(http.StatusBadRequest).SendString("invalid json body")
		}
		token := c.GetReqHeaders()["Session"]
		session, _ := sessions.Load(token)
		modId, err := db.GetModeratorId(session.(Session).login)
		if err != nil {
			log.Println(err.Error())
			return c.Status(http.StatusInternalServerError).SendString("unknown error")
		}
		id, err := strconv.Atoi(c.GetReqHeaders()["Contest"])
		if err != nil {
			log.Println(err.Error())
			return c.Status(http.StatusBadRequest).SendString("no such contest")
		}
		group, err := db.GetGroupByContest(id)
		if err != nil {
			log.Println(err.Error())
			return c.Status(http.StatusInternalServerError).SendString("unknown error")
		}
		if db.IsHostInGroup(group, modId) {
			prevId, err := db.GetContestId(contest)
			if err == nil && prevId != 0 {
				log.Println("trying to change contest into existing " + contest.Name)
				return c.Status(http.StatusBadRequest).SendString("identical contest exists")
			}
			err = db.EditContest(id, contest)
			if err != nil {
				log.Println(err.Error())
				return c.Status(http.StatusInternalServerError).SendString("unable to edit contest")
			}
		} else {
			return c.Status(http.StatusForbidden).SendString("you are not host in group")
		}
		return c.Status(http.StatusOK).SendString("successful")
	})
	app.Post("/api/admins/remove_host", func(c *fiber.Ctx) error {
		res := SessionAuthCheck(c)
		switch res {
		case http.StatusUnauthorized:
			return c.Status(http.StatusUnauthorized).SendString("session has expired")
		case http.StatusForbidden:
			return c.Status(http.StatusForbidden).SendString("user not authorized")
		case http.StatusBadRequest:
			return c.Status(http.StatusBadRequest).SendString("header doesn't have session key")
		}
		token := c.GetReqHeaders()["Session"]
		session, _ := sessions.Load(token)
		id, err := db.GetModeratorId(session.(Session).login)
		if err != nil {
			log.Println(err.Error())
			return c.Status(http.StatusInternalServerError).SendString("unable to get your id")
		}
		var req models.GroupAndHost
		err = json.Unmarshal(c.Body(), &req)
		if err != nil {
			log.Println("can't parse json " + err.Error())
			return c.Status(http.StatusBadRequest).SendString("invalid json body")
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
	app.Post("/api/admins/edit_group", func(c *fiber.Ctx) error {
		res := SessionAuthCheck(c)
		switch res {
		case http.StatusUnauthorized:
			return c.Status(http.StatusUnauthorized).SendString("session has expired")
		case http.StatusForbidden:
			return c.Status(http.StatusForbidden).SendString("user not authorized")
		case http.StatusBadRequest:
			return c.Status(http.StatusBadRequest).SendString("header doesn't have session key")
		}
		token := c.GetReqHeaders()["Session"]
		session, _ := sessions.Load(token)
		id, err := db.GetModeratorId(session.(Session).login)
		if err != nil {
			log.Println(err.Error())
			return c.Status(http.StatusInternalServerError).SendString("unable to get your id")
		}
		var req models.BasicGroup
		err = json.Unmarshal(c.Body(), &req)
		if err != nil {
			log.Println("can't parse json " + err.Error())
			return c.Status(http.StatusBadRequest).SendString("invalid json body")
		}
		group, _ := url.QueryUnescape(c.GetReqHeaders()["Group"])
		gId, err := db.GetGroupId(group)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(http.StatusBadRequest).SendString("no such group")
		} else if err != nil {
			log.Println(err.Error())
			return c.Status(http.StatusInternalServerError).SendString("cannot get group id")
		}
		if !db.IsHostInGroup(gId, id) {
			return c.Status(http.StatusForbidden).SendString("you must be host in group to edit it")
		}
		err = db.EditGroup(gId, req)
		if err != nil {
			return c.Status(http.StatusInternalServerError).SendString("cannot edit group")
		}
		return c.Status(http.StatusOK).SendString("successful")
	})
}
