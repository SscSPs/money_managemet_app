package handlers

import (
	"github.com/SscSPs/money_managemet_app/cmd/docs"
	"github.com/SscSPs/money_managemet_app/internal/middleware"
	"github.com/SscSPs/money_managemet_app/pkg/config"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// RegisterRoutes sets up all application routes
func RegisterRoutes(r *gin.Engine, cfg *config.Config, dbPool *pgxpool.Pool) {
	// Register public authentication routes
	RegisterAuthRoutes(r, cfg)

	// Setup API v1 routes with Auth Middleware
	setupAPIV1Routes(r, cfg, dbPool)

	// Swagger routes (typically public or conditionally available)
	setupSwaggerRoutes(r, cfg)
}

// setupAPIV1Routes configures the /api/v1 group and delegates to specific entity route registrations
func setupAPIV1Routes(r *gin.Engine, cfg *config.Config, dbPool *pgxpool.Pool) {
	// Apply AuthMiddleware to the entire v1 group
	v1 := r.Group("/api/v1", middleware.AuthMiddleware(cfg.JWTSecret))

	// Delegate route registration to specific handlers
	registerExampleRoutes(v1)
	registerJournalRoutes(v1, dbPool)
	registerAccountRoutes(v1, dbPool)
	registerUserRoutes(v1, dbPool)
	registerCurrencyRoutes(v1, dbPool)
}

// setupSwaggerRoutes configures the swagger documentation routes
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
