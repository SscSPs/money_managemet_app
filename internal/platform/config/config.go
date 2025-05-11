package config

import (
	"log"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Config holds application configuration.
type Config struct {
	DatabaseURL       string
	Port              string
	IsProduction      bool
	EnableDBCheck     bool
	JWTSecret         string
	JWTExpiryDuration time.Duration
	JWTIssuer         string // Added JWTIssuer
	// Refresh Token Config
	RefreshTokenExpiryDuration time.Duration
	RefreshTokenCookieName     string
	RefreshTokenCookiePath     string `mapstructure:"REFRESH_TOKEN_COOKIE_PATH"` // Added RefreshTokenCookiePath
	RefreshTokenSecret         string

	// External OAuth Providers
	GoogleClientID     string `mapstructure:"GOOGLE_CLIENT_ID"`
	GoogleClientSecret string `mapstructure:"GOOGLE_CLIENT_SECRET"`
	GoogleRedirectURL  string `mapstructure:"GOOGLE_REDIRECT_URL"`
	FrontendBaseURL    string `mapstructure:"FRONTEND_BASE_URL"`
}

// LoadConfig loads configuration from environment variables and .env file if present.
func LoadConfig() (*Config, error) {
	// Attempt to load .env file, ignore error if it doesn't exist
	_ = godotenv.Load()

	viper.SetDefault("PGSQL_URL", "")
	viper.SetDefault("PORT", "8080")
	viper.SetDefault("IS_PRODUCTION", false)
	viper.SetDefault("ENABLE_DB_CHECK", false)
	viper.SetDefault("JWT_SECRET", "a-very-secret-key-should-be-longer-and-random")
	viper.SetDefault("JWT_EXPIRY_DURATION", "1h")
	viper.SetDefault("JWT_ISSUER", "money-management-app") // Added default for JWT_ISSUER
	viper.SetDefault("REFRESH_TOKEN_EXPIRY_DURATION", "168h")
	viper.SetDefault("REFRESH_TOKEN_COOKIE_NAME", "rtid")
	viper.SetDefault("REFRESH_TOKEN_COOKIE_PATH", "/api/v1/auth") // Added default for REFRESH_TOKEN_COOKIE_PATH
	viper.SetDefault("REFRESH_TOKEN_SECRET", "default_insecure_refresh_secret_please_change_this_!@#$")
	viper.SetDefault("GOOGLE_CLIENT_ID", "")
	viper.SetDefault("GOOGLE_CLIENT_SECRET", "")
	viper.SetDefault("GOOGLE_REDIRECT_URL", "")
	viper.SetDefault("FRONTEND_BASE_URL", "http://localhost:3000")

	// Read .env file if it exists
	// This allows overriding defaults with .env file values, which can then be overridden by actual environment variables.
	viper.AutomaticEnv()

	cfg := &Config{}

	cfg.DatabaseURL = viper.GetString("PGSQL_URL")
	if cfg.DatabaseURL == "" {
		log.Println("Warning: PGSQL_URL environment variable not set.")
		// Consider returning an error depending on requirements
	}

	cfg.Port = viper.GetString("PORT")
	if cfg.Port == "" {
		cfg.Port = "8080" // Default port
		log.Printf("Warning: PORT environment variable not set. Defaulting to %s\n", cfg.Port)
	}

	// Load JWT Secret
	jwtSecret := viper.GetString("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "a-very-secret-key-should-be-longer-and-random" // !! CHANGE IN PRODUCTION !!
		log.Println("Warning: JWT_SECRET environment variable not set. Using default insecure key.")
	}

	// Load JWT Expiry Duration (e.g., "60m", "1h")
	jwtExpiryStr := viper.GetString("JWT_EXPIRY_DURATION")
	jwtExpiryDuration, err := time.ParseDuration(jwtExpiryStr)
	if err != nil {
		jwtExpiryDuration = time.Hour * 1 // Default to 1 hour
		if jwtExpiryStr != "" {
			log.Printf("Warning: Invalid value for JWT_EXPIRY_DURATION ('%s'). Defaulting to %s.\n", jwtExpiryStr, jwtExpiryDuration.String())
		}
	}

	// Load JWT Issuer
	jwtIssuer := viper.GetString("JWT_ISSUER")
	if jwtIssuer == "" {
		jwtIssuer = "money-management-app" // Default JWT issuer
		log.Printf("Warning: JWT_ISSUER not set. Defaulting to %s.\n", jwtIssuer)
	}

	// Load Refresh Token Expiry Duration (e.g., "168h" for 7 days)
	refreshTokenExpiryStr := viper.GetString("REFRESH_TOKEN_EXPIRY_DURATION")
	refreshTokenExpiryDuration, err := time.ParseDuration(refreshTokenExpiryStr)
	if err != nil {
		refreshTokenExpiryDuration = time.Hour * 24 * 7 // Default to 7 days
		if refreshTokenExpiryStr != "" {
			log.Printf("Warning: Invalid value for REFRESH_TOKEN_EXPIRY_DURATION ('%s'). Defaulting to %s.\n", refreshTokenExpiryStr, refreshTokenExpiryDuration.String())
		} else {
			log.Printf("Warning: REFRESH_TOKEN_EXPIRY_DURATION not set. Defaulting to %s.\n", refreshTokenExpiryDuration.String())
		}
	}

	refreshTokenCookieName := viper.GetString("REFRESH_TOKEN_COOKIE_NAME")
	if refreshTokenCookieName == "" {
		refreshTokenCookieName = "rtid" // Default refresh token cookie name
		log.Printf("Warning: REFRESH_TOKEN_COOKIE_NAME not set. Defaulting to %s.\n", refreshTokenCookieName)
	}

	refreshTokenCookiePath := viper.GetString("REFRESH_TOKEN_COOKIE_PATH")
	if refreshTokenCookiePath == "" {
		refreshTokenCookiePath = "/api/v1/auth" // Default refresh token cookie path
		log.Printf("Warning: REFRESH_TOKEN_COOKIE_PATH not set. Defaulting to %s.\n", refreshTokenCookiePath)
	}

	refreshTokenSecret := viper.GetString("REFRESH_TOKEN_SECRET")
	if refreshTokenSecret == "" {
		// Provide a fallback or ensure it's set if critical, for now, let's log if it's empty in a real scenario
		// For development, a default might be acceptable but not for production.
		log.Println("Warning: REFRESH_TOKEN_SECRET is not set, using default insecure secret. THIS IS NOT FOR PRODUCTION.")
		refreshTokenSecret = "default_insecure_refresh_secret_please_change_this_!@#$"
	}

	cfg.GoogleClientID = viper.GetString("GOOGLE_CLIENT_ID")
	cfg.GoogleClientSecret = viper.GetString("GOOGLE_CLIENT_SECRET")
	cfg.GoogleRedirectURL = viper.GetString("GOOGLE_REDIRECT_URL")
	cfg.FrontendBaseURL = viper.GetString("FRONTEND_BASE_URL")

	// Log warnings for missing critical OAuth ENV variables
	if cfg.GoogleClientID == "" {
		log.Println("Warning: GOOGLE_CLIENT_ID not set. Google OAuth will not function.")
	}
	if cfg.GoogleClientSecret == "" {
		log.Println("Warning: GOOGLE_CLIENT_SECRET not set. Google OAuth will not function.")
	}
	if cfg.GoogleRedirectURL == "" {
		log.Println("Warning: GOOGLE_REDIRECT_URL not set. Google OAuth will not function.")
	}

	cfg.DatabaseURL = viper.GetString("PGSQL_URL")
	cfg.Port = viper.GetString("PORT")
	cfg.IsProduction = viper.GetBool("IS_PRODUCTION")
	cfg.EnableDBCheck = viper.GetBool("ENABLE_DB_CHECK")
	cfg.JWTSecret = viper.GetString("JWT_SECRET")
	cfg.JWTExpiryDuration = jwtExpiryDuration
	cfg.JWTIssuer = jwtIssuer
	cfg.RefreshTokenExpiryDuration = refreshTokenExpiryDuration
	cfg.RefreshTokenCookieName = refreshTokenCookieName
	cfg.RefreshTokenCookiePath = refreshTokenCookiePath
	cfg.RefreshTokenSecret = refreshTokenSecret

	return cfg, nil
}
