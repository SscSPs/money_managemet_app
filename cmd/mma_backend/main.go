package main

import (
	"context"

	"github.com/SscSPs/money_managemet_app/cmd/docs"
	"github.com/SscSPs/money_managemet_app/internal/adapters/database/pgsql"
	"github.com/SscSPs/money_managemet_app/internal/core/services"
	"github.com/SscSPs/money_managemet_app/internal/handlers"
	"github.com/SscSPs/money_managemet_app/pkg/config"
	"github.com/SscSPs/money_managemet_app/pkg/database"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"log"
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

	// Initialize database connection
	dbPool, err := database.NewPgxPool(context.Background(), cfg.DatabaseURL, cfg.EnableDBCheck)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
		return
	}
	// Defer closing the connection pool
	defer dbPool.Close()

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
	ledger.POST("/create", ledgerHandler.PersistJournal)
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
