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
	token := c.Cookies(AuthCookieName, "-1")
	if token == "-1" {
		log.Println("cookie not found")
		return http.StatusForbidden
	}
	session, exists := sessions[token]
	if !exists {
		log.Printf("user not authorized on access route, token %v", token)
		return http.StatusBadRequest
	}
	if session.isExpired() {
		log.Printf("session %v has expired", session)
		delete(sessions, token)
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
