package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port         string
	IsProduction bool
}

func LoadConfig() Config {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	return Config{
		Port:         os.Getenv("PORT"),
		IsProduction: os.Getenv("IS_PRODUCTION") == "true",
	}
}
