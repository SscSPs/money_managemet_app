package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"

	"github.com/SscSPs/money_managemet_app/cmd/docs"
	"github.com/SscSPs/money_managemet_app/internal/adapters/database/pgsql"
	"github.com/SscSPs/money_managemet_app/internal/core/services"
	"github.com/SscSPs/money_managemet_app/internal/handlers"
	"github.com/SscSPs/money_managemet_app/internal/middleware"
	"github.com/SscSPs/money_managemet_app/pkg/config"
	"github.com/SscSPs/money_managemet_app/pkg/database"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	migrate "github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
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
	defer func() {
		if cerr := migrationDB.Close(); cerr != nil {
			logger.Error("Error closing migration DB connection", slog.String("error", cerr.Error()))
		}
	}()

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
	if err != nil && err != migrate.ErrNoChange {
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

	if err == migrate.ErrNoChange {
		logger.Info("No new migrations to apply.")
	} else {
		logger.Info("Database migrations applied successfully.")
	}
	// --- End Database Migrations ---

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

	// Setup handlers
	authHandler := handlers.NewAuthHandler(cfg)

	// Public routes (e.g., login, health check)
	authRoutes := r.Group("/auth")
	{
		authRoutes.POST("/login", authHandler.Login)
		// TODO: Add refresh token route later
	}

	// Setup API v1 routes with Auth Middleware
	setupAPIV1Routes(r, cfg, dbPool, logger)

	// Swagger routes (typically public or conditionally available)
	setupSwaggerRoutes(r, cfg)

	logger.Info("Server starting", slog.String("port", cfg.Port))
	if err := r.Run(":" + cfg.Port); err != nil {
		logger.Error("Server failed to run", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func setupAPIV1Routes(r *gin.Engine, cfg *config.Config, dbPool *pgxpool.Pool, logger *slog.Logger) {
	// Apply AuthMiddleware to the entire v1 group
	v1 := r.Group("/api/v1", middleware.AuthMiddleware(cfg.JWTSecret))

	// Pass dbPool and the base logger (handlers get request-scoped logger from context)
	addExampleAPI(v1) // Should example be protected? Maybe move outside v1 or make public.
	addLedgerAPI(v1, dbPool, logger)
	addAccountAPI(v1, dbPool, logger)
	addUserAPI(v1, dbPool, logger)
	addCurrencyAPI(v1, dbPool, logger)
}

func addLedgerAPI(v1 *gin.RouterGroup, dbPool *pgxpool.Pool, logger *slog.Logger) {
	ledger := v1.Group("/ledger")
	ledgerService := services.NewLedgerService(pgsql.NewAccountRepository(dbPool), pgsql.NewJournalRepository(dbPool))
	ledgerHandler := handlers.NewLedgerHandler(ledgerService)
	ledger.POST("/", ledgerHandler.PersistJournal)
	ledger.GET("/:journalID", ledgerHandler.GetJournal)
}

func addExampleAPI(v1 *gin.RouterGroup) {
	eg := v1.Group("/example")
	eg.GET("/helloworld", handlers.GetHome)
}

func addAccountAPI(v1 *gin.RouterGroup, dbPool *pgxpool.Pool, logger *slog.Logger) {
	accountRepo := pgsql.NewAccountRepository(dbPool)
	accountService := services.NewAccountService(accountRepo)
	accountHandler := handlers.NewAccountHandler(accountService)

	accounts := v1.Group("/accounts")
	accounts.POST("/", accountHandler.CreateAccount)
	accounts.GET("/:accountID", accountHandler.GetAccount)
}

func addUserAPI(v1 *gin.RouterGroup, dbPool *pgxpool.Pool, logger *slog.Logger) {
	userRepo := pgsql.NewUserRepository(dbPool)
	userService := services.NewUserService(userRepo)
	userHandler := handlers.NewUserHandler(userService)

	users := v1.Group("/users")
	{
		users.POST("/", userHandler.CreateUser)          // Create
		users.GET("/", userHandler.ListUsers)            // List (Read all)
		users.GET("/:userID", userHandler.GetUser)       // Read one
		users.PUT("/:userID", userHandler.UpdateUser)    // Update
		users.DELETE("/:userID", userHandler.DeleteUser) // Delete
	}
}

func addCurrencyAPI(v1 *gin.RouterGroup, dbPool *pgxpool.Pool, logger *slog.Logger) {
	currencyRepo := pgsql.NewCurrencyRepository(dbPool)
	currencyService := services.NewCurrencyService(currencyRepo)
	currencyHandler := handlers.NewCurrencyHandler(currencyService)

	currencies := v1.Group("/currencies")
	currencies.POST("/", currencyHandler.CreateCurrency)
	currencies.GET("/", currencyHandler.ListCurrencies)
	currencies.GET("/:currencyCode", currencyHandler.GetCurrency)
}

func setupSwaggerRoutes(r *gin.Engine, cfg *config.Config) {
	// Swagger setup
	if cfg.IsProduction {
		//no swagger in prod
		return
	}
	docs.SwaggerInfo.BasePath = "/api/v1"
	swagger := r.Group("/swagger")
	swagger.GET("/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
