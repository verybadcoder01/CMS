package config

import (
	"gopkg.in/yaml.v3"
	"log"
	"os"
)

var ISDEBUG = false

type Config struct {
	DbPath            string `yaml:"db_path"`
	LogPath           string `yaml:"log_path"`
	Port              string `yaml:"port"`
	IsDebug           bool   `yaml:"is_debug"`
	AdminLogin        string `yaml:"admin_login"`
	AdminPassword     string `yaml:"admin_password"`
	SessionExpiryTime int    `yaml:"session_expiry_time"` // в часах, при is_debug=false
}

func ParseConfig() Config {
	var conf Config
	data, err := os.ReadFile("./config.yaml")
	if err != nil {
		log.Fatal(err.Error())
	}
	err = yaml.Unmarshal(data, &conf)
	if err != nil {
		log.Fatal(err.Error())
	}
	ISDEBUG = conf.IsDebug
	return conf
}
