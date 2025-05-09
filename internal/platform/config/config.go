package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds application configuration.
type Config struct {
	DatabaseURL       string
	Port              string
	IsProduction      bool
	EnableDBCheck     bool
	JWTSecret         string
	JWTExpiryDuration time.Duration

	// Refresh Token Config
	RefreshTokenExpiryDuration time.Duration
	RefreshTokenCookieName     string
	RefreshTokenSecret         string
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

	// Load IsProduction flag
	isProdStr := os.Getenv("IS_PRODUCTION")
	isProd, err := strconv.ParseBool(isProdStr)
	if err != nil {
		// Default to false if not set or invalid boolean
		isProd = false
		if isProdStr != "" {
			log.Printf("Warning: Invalid value for IS_PRODUCTION ('%s'). Defaulting to false.\n", isProdStr)
		}
	}

	enableDBCheckStr := os.Getenv("ENABLE_DB_CHECK")
	enableDBCheck, err := strconv.ParseBool(enableDBCheckStr)
	if err != nil {
		enableDBCheck = false
		if enableDBCheckStr != "" {
			log.Printf("Warning: Invalid value for ENABLE_DB_CHECK ('%s'). Defaulting to false.\n", enableDBCheckStr)
		}
	}

	// Load JWT Secret
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "a-very-secret-key-should-be-longer-and-random" // !! CHANGE IN PRODUCTION !!
		log.Println("Warning: JWT_SECRET environment variable not set. Using default insecure key.")
	}

	// Load JWT Expiry Duration (e.g., "60m", "1h")
	jwtExpiryStr := os.Getenv("JWT_EXPIRY_DURATION")
	jwtExpiryDuration, err := time.ParseDuration(jwtExpiryStr)
	if err != nil {
		jwtExpiryDuration = time.Hour * 1 // Default to 1 hour
		if jwtExpiryStr != "" {
			log.Printf("Warning: Invalid value for JWT_EXPIRY_DURATION ('%s'). Defaulting to %s.\n", jwtExpiryStr, jwtExpiryDuration.String())
		}
	}

	// Load Refresh Token Expiry Duration (e.g., "168h" for 7 days)
	refreshTokenExpiryStr := os.Getenv("REFRESH_TOKEN_EXPIRY_DURATION")
	refreshTokenExpiryDuration, err := time.ParseDuration(refreshTokenExpiryStr)
	if err != nil {
		refreshTokenExpiryDuration = time.Hour * 24 * 7 // Default to 7 days
		if refreshTokenExpiryStr != "" {
			log.Printf("Warning: Invalid value for REFRESH_TOKEN_EXPIRY_DURATION ('%s'). Defaulting to %s.\n", refreshTokenExpiryStr, refreshTokenExpiryDuration.String())
		} else {
			log.Printf("Warning: REFRESH_TOKEN_EXPIRY_DURATION not set. Defaulting to %s.\n", refreshTokenExpiryDuration.String())
		}
	}

	refreshTokenCookieName := os.Getenv("REFRESH_TOKEN_COOKIE_NAME")
	if refreshTokenCookieName == "" {
		refreshTokenCookieName = "rtid" // Default refresh token cookie name
		log.Printf("Warning: REFRESH_TOKEN_COOKIE_NAME not set. Defaulting to %s.\n", refreshTokenCookieName)
	}

	refreshTokenSecret := os.Getenv("REFRESH_TOKEN_SECRET")
	if refreshTokenSecret == "" {
		// Provide a fallback or ensure it's set if critical, for now, let's log if it's empty in a real scenario
		// For development, a default might be acceptable but not for production.
		log.Println("Warning: REFRESH_TOKEN_SECRET is not set, using default insecure secret. THIS IS NOT FOR PRODUCTION.")
		refreshTokenSecret = "default_insecure_refresh_secret_please_change_this_!@#$"
	}

	return &Config{
		DatabaseURL:                dbURL,
		Port:                       port,
		IsProduction:               isProd,
		EnableDBCheck:              enableDBCheck,
		JWTSecret:                  jwtSecret,
		JWTExpiryDuration:          jwtExpiryDuration,
		RefreshTokenExpiryDuration: refreshTokenExpiryDuration,
		RefreshTokenCookieName:     refreshTokenCookieName,
		RefreshTokenSecret:         refreshTokenSecret,
	}, nil
}
