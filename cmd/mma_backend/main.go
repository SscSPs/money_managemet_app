package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"os"

	"github.com/SscSPs/money_managemet_app/internal/handlers"
	"github.com/SscSPs/money_managemet_app/internal/middleware"
	"github.com/SscSPs/money_managemet_app/pkg/config"
	"github.com/SscSPs/money_managemet_app/pkg/database"
	"github.com/gin-gonic/gin"

	migrate "github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// @title MMA Backend API
// @version 1.0
// @description This is a sample server for MMA backend.

// @host localhost:8080
// @BasePath /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

// @security BearerAuth
func main() {
	// Initialize structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger) // Optional: Set as default logger

	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Error("Failed to load config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Initialize database connection pool (for application use)
	dbPool, err := database.NewPgxPool(context.Background(), cfg.DatabaseURL, cfg.EnableDBCheck)
	if err != nil {
		logger.Error("Failed to initialize database pool", slog.String("error", err.Error()))
		os.Exit(1)
	}
	// Defer closing the connection pool
	defer dbPool.Close()
	logger.Info("Database connection pool established.")

	// --- Run Database Migrations ---
	logger.Info("Running database migrations...")
	// Open a temporary standard sql.DB connection for migrations
	// Using pgx/v5/stdlib driver to be compatible with the main pool
	migrationDB, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		logger.Error("Failed to open database connection for migrations", slog.String("error", err.Error()))
		os.Exit(1)
	}
	if err := migrationDB.Ping(); err != nil {
		logger.Error("Failed to ping database for migrations", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Create a postgres driver instance for migrate
	driver, err := postgres.WithInstance(migrationDB, &postgres.Config{})
	if err != nil {
		logger.Error("Could not create postgres driver instance for migrations", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Point to the migrations directory (adjust path if needed)
	migrationsPath := "file://migrations"

	// Create a new migrate instance
	m, err := migrate.NewWithDatabaseInstance(
		migrationsPath,
		"postgres", // Database name used by migrate
		driver,
	)
	if err != nil {
		logger.Error("Could not create migrate instance", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Apply all available "up" migrations
	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		logger.Error("Failed to apply migrations", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Check for dirty migrations after running Up.
	sourceErr, dbErr := m.Close()
	if sourceErr != nil {
		logger.Error("Migration source error", slog.String("error", sourceErr.Error()))
		os.Exit(1)
	}
	if dbErr != nil {
		logger.Error("Migration database error", slog.String("error", dbErr.Error()))
		os.Exit(1)
	}

	if errors.Is(err, migrate.ErrNoChange) {
		logger.Info("No new migrations to apply.")
	} else {
		logger.Info("Database migrations applied successfully.")
	}
	// --- End Database Migrations ---

	// Initialize Gin engine
	if cfg.IsProduction {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()

	// Global middleware (logging, recovery)
	r.Use(middleware.StructuredLoggingMiddleware(logger), gin.Recovery())

	err = r.SetTrustedProxies(nil)
	if err != nil {
		logger.Error("Failed to set trusted proxies", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Register all routes
	handlers.RegisterRoutes(r, cfg, dbPool)

	logger.Info("Server starting", slog.String("port", cfg.Port))
	if err := r.Run(":" + cfg.Port); err != nil {
		logger.Error("Server failed to run", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
