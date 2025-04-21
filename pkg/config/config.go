package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds application configuration.
type Config struct {
	DatabaseURL  string
	Port         string
	IsProduction bool
}

// LoadConfig loads configuration from environment variables.
// It looks for a .env file first.
func LoadConfig() (*Config, error) {
	// Attempt to load .env file, ignore error if it doesn't exist
	_ = godotenv.Load()

	dbURL := os.Getenv("PGSQL_URL")
	if dbURL == "" {
		log.Println("Warning: PGSQL_URL environment variable not set.")
		// Consider returning an error depending on requirements
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default port
		log.Printf("Warning: PORT environment variable not set. Defaulting to %s\n", port)
	}

	isProdStr := os.Getenv("IS_PRODUCTION")
	isProd, err := strconv.ParseBool(isProdStr)
	if err != nil {
		// Default to false if not set or invalid boolean
		isProd = false
		if isProdStr != "" {
			log.Printf("Warning: Invalid value for IS_PRODUCTION ('%s'). Defaulting to false.\n", isProdStr)
		}
	}

	return &Config{
		DatabaseURL:  dbURL,
		Port:         port,
		IsProduction: isProd,
	}, nil
}
