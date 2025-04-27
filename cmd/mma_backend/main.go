package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"os"

	"github.com/SscSPs/money_managemet_app/internal/core/services"
	"github.com/SscSPs/money_managemet_app/internal/handlers"
	"github.com/SscSPs/money_managemet_app/internal/middleware"
	"github.com/SscSPs/money_managemet_app/internal/platform/config"
	"github.com/SscSPs/money_managemet_app/internal/platform/database"
	"github.com/SscSPs/money_managemet_app/internal/repositories/database/pgsql"
	"github.com/gin-gonic/gin"

	migrate "github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
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

	// Run database migrations before initializing the main pool
	runDatabaseMigrations(logger, cfg)

	// Initialize database connection pool (for application use)
	dbPool := setupDatabaseConnection(logger, cfg)
	// Defer closing the connection pool
	defer dbPool.Close()
	logger.Info("Database connection pool established.")

	// --- Dependency Injection Setup ---
	logger.Info("Initializing repositories and services...")

	// Repositories
	accountRepo := pgsql.NewPgxAccountRepository(dbPool)
	currencyRepo := pgsql.NewPgxCurrencyRepository(dbPool)
	exchangeRateRepo := pgsql.NewPgxExchangeRateRepository(dbPool)
	userRepo := pgsql.NewPgxUserRepository(dbPool)
	journalRepo := pgsql.NewPgxJournalRepository(dbPool)
	workplaceRepo := pgsql.NewPgxWorkplaceRepository(dbPool)

	// Services
	accountService := services.NewAccountService(accountRepo)
	currencyService := services.NewCurrencyService(currencyRepo)
	exchangeRateService := services.NewExchangeRateService(exchangeRateRepo, currencyService)
	userService := services.NewUserService(userRepo)
	workplaceService := services.NewWorkplaceService(workplaceRepo)
	journalService := services.NewJournalService(accountRepo, journalRepo, workplaceService)

	logger.Info("Dependencies initialized.")
	// --- End Dependency Injection Setup ---

	// Initialize Gin engine
	r := setupGinEngine(logger, cfg)

	// Pass initialized services to route registration
	handlers.RegisterRoutes(r, cfg, *userService, *accountService, *currencyService, *exchangeRateService, *journalService, *workplaceService)

	logger.Info("Server starting", slog.String("port", cfg.Port))
	if err := r.Run(":" + cfg.Port); err != nil {
		logger.Error("Server failed to run", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

// setupDatabaseConnection initializes the PostgreSQL connection pool.
func setupDatabaseConnection(logger *slog.Logger, cfg *config.Config) *pgxpool.Pool {
	dbPool, err := database.NewPgxPool(context.Background(), cfg.DatabaseURL, cfg.EnableDBCheck)
	if err != nil {
		logger.Error("Failed to initialize database pool", slog.String("error", err.Error()))
		os.Exit(1)
	}
	return dbPool
}

// setupGinEngine initializes and configures the Gin engine.
func setupGinEngine(logger *slog.Logger, cfg *config.Config) *gin.Engine {
	if cfg.IsProduction {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()

	// Global middleware (logging, recovery)
	r.Use(middleware.StructuredLoggingMiddleware(logger), gin.Recovery())

	err := r.SetTrustedProxies(nil) // Set trusted proxies (nil means trust nothing, adjust as needed)
	if err != nil {
		logger.Error("Failed to set trusted proxies", slog.String("error", err.Error()))
		os.Exit(1) // Exit if setting trusted proxies fails
	}

	return r
}

// runDatabaseMigrations handles the process of applying database migrations.
func runDatabaseMigrations(logger *slog.Logger, cfg *config.Config) {
	logger.Info("Running database migrations...")
	// Open a temporary standard sql.DB connection for migrations
	// Using pgx/v5/stdlib driver to be compatible with the main pool
	migrationDB, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		logger.Error("Failed to open database connection for migrations", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Ping the database to ensure the connection is valid before proceeding
	if err := migrationDB.Ping(); err != nil {
		logger.Error("Failed to ping database for migrations", slog.String("error", err.Error()))
		os.Exit(1)
	}
	logger.Info("Migration database connection established.")

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
		// If there's an error other than "no change", log it and exit.
		logger.Error("Failed to apply migrations", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Check for migration source or database errors during close.
	// It's important to check these as they can indicate issues like lock contention or corrupted migration state.
	sourceErr, dbErr := m.Close()
	if sourceErr != nil {
		logger.Error("Migration source error on close", slog.String("error", sourceErr.Error()))
		os.Exit(1)
	}
	if dbErr != nil {
		logger.Error("Migration database error on close", slog.String("error", dbErr.Error()))
		os.Exit(1)
	}

	// Log the outcome of the migration process.
	if errors.Is(err, migrate.ErrNoChange) {
		logger.Info("No new migrations to apply.")
	} else {
		// Only log success if migrations were actually applied (err was nil initially)
		logger.Info("Database migrations applied successfully.")
	}
}
