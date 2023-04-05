package internal

import (
	"cms/models"
	"gopkg.in/yaml.v3"
	"log"
	"os"
)

func ParseConfig() models.Config {
	var conf models.Config
	data, err := os.ReadFile("./config.yaml")
	if err != nil {
		log.Fatal(err.Error())
	}
	err = yaml.Unmarshal(data, &conf)
	if err != nil {
		log.Fatal(err.Error())
	}
	return conf
}
