package main

import (
	"context"
	"database/sql"
	"log"

	"github.com/SscSPs/money_managemet_app/cmd/docs"
	"github.com/SscSPs/money_managemet_app/internal/adapters/database/pgsql"
	"github.com/SscSPs/money_managemet_app/internal/core/services"
	"github.com/SscSPs/money_managemet_app/internal/handlers"
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

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
		return
	}

	// Initialize database connection pool (for application use)
	dbPool, err := database.NewPgxPool(context.Background(), cfg.DatabaseURL, cfg.EnableDBCheck)
	if err != nil {
		log.Fatalf("Failed to initialize database pool: %v", err)
		return
	}
	// Defer closing the connection pool
	defer dbPool.Close()
	log.Println("Database connection pool established.")

	// --- Run Database Migrations ---
	log.Println("Running database migrations...")
	// Open a temporary standard sql.DB connection for migrations
	// Using pgx/v5/stdlib driver to be compatible with the main pool
	migrationDB, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to open database connection for migrations: %v", err)
	}
	if err := migrationDB.Ping(); err != nil {
		log.Fatalf("Failed to ping database for migrations: %v", err)
	}
	defer func() {
		if cerr := migrationDB.Close(); cerr != nil {
			log.Printf("Error closing migration DB connection: %v", cerr)
		}
	}()

	// Create a postgres driver instance for migrate
	driver, err := postgres.WithInstance(migrationDB, &postgres.Config{})
	if err != nil {
		log.Fatalf("Could not create postgres driver instance for migrations: %v", err)
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
		log.Fatalf("Could not create migrate instance: %v", err)
	}

	// Apply all available "up" migrations
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		log.Fatalf("Failed to apply migrations: %v", err)
	}

	// Check for dirty migrations after running Up.
	sourceErr, dbErr := m.Close()
	if sourceErr != nil {
		log.Fatalf("Migration source error: %v", sourceErr)
	}
	if dbErr != nil {
		log.Fatalf("Migration database error: %v", dbErr)
	}

	if err == migrate.ErrNoChange {
		log.Println("No new migrations to apply.")
	} else {
		log.Println("Database migrations applied successfully.")
	}
	// --- End Database Migrations ---

	if cfg.IsProduction {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	err = r.SetTrustedProxies(nil)
	if err != nil {
		log.Fatal(err)
		return
	}

	setupAPIV1Routes(r, cfg, dbPool)
	setupSwaggerRoutes(r, cfg)

	log.Printf("Server is running on port %s...\n", cfg.Port)
	log.Fatal(r.Run(":" + cfg.Port))
}

func setupAPIV1Routes(r *gin.Engine, cfg *config.Config, dbPool *pgxpool.Pool) {
	v1 := r.Group("/api/v1")
	addExampleAPI(v1)
	addLedgerAPI(v1, dbPool)
}

func addLedgerAPI(v1 *gin.RouterGroup, dbPool *pgxpool.Pool) {
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
