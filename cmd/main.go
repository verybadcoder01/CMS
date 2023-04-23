package main

import (
	"cms/db"
	"cms/internal"
	"cms/models"
	"github.com/gofiber/fiber/v2"
	"gopkg.in/natefinch/lumberjack.v2"
	"gorm.io/gorm/logger"
	"log"
	"os"
	"time"
)

func main() {
	conf := internal.ParseConfig()
	models.ISDEBUG = conf.IsDebug
	f, err := os.OpenFile(conf.LogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	// gorm logger
	gormLog := logger.New(
		log.New(f, "", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second, // Slow SQL threshold
			LogLevel:                  logger.Info, // Log level
			IgnoreRecordNotFoundError: false,
			Colorful:                  false,
		},
	)
	log.SetOutput(&lumberjack.Logger{
		Filename:   conf.LogPath,
		MaxSize:    32, // megabytes
		MaxBackups: 3,
		MaxAge:     28,   //days
		Compress:   true, // disabled by default
	})
	log.Println("This is a test log entry")
	app := fiber.New()
	db.CreateDbFile(conf.DbPath, gormLog, models.SimpleModerator{Login: conf.AdminLogin, Password: internal.HashPassword(conf.AdminPassword)})
	internal.SessionExpiryTime = time.Duration(conf.SessionExpiryTime) * time.Hour
	internal.CookieExpiryTime = time.Duration(conf.CookieExpiryTime) * time.Hour
	internal.SetupRouting(app)
	err = app.Listen(conf.Port)
	if err != nil {
		log.Fatal("Port unavailable: " + err.Error())
	}
}
