package internal

import (
	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
	"log"
	"net/http"
	"time"
)

// CookieAuthCheck проверяет, актуальна ли текущая сессия модера
func CookieAuthCheck(c *fiber.Ctx) int {
	session, exists := c.GetReqHeaders()["Session"]
	if !exists {
		log.Println("session header not found")
		return http.StatusBadRequest
	}
	info, exists := sessions[session]
	if !exists {
		log.Printf("user not authorized on access route, session %v", info)
		return http.StatusForbidden
	}
	if info.isExpired() {
		log.Printf("session %v has expired", info)
		delete(sessions, session)
		return http.StatusUnauthorized
	}
	return 0
}

func HashPassword(password string) string {
	bytes, _ := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes)
}

func CheckPasswordHash(password string, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func (s Session) isExpired() bool {
	return s.expiry.Before(time.Now())
}
