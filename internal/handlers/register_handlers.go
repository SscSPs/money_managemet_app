package handlers

import (
	"github.com/SscSPs/money_managemet_app/cmd/docs"
	portssvc "github.com/SscSPs/money_managemet_app/internal/core/ports/services"
	"github.com/SscSPs/money_managemet_app/internal/middleware"
	"github.com/SscSPs/money_managemet_app/internal/platform/config"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// RegisterRoutes sets up all application routes, injecting dependencies using interfaces
func RegisterRoutes(
	r *gin.Engine,
	cfg *config.Config,
	services *portssvc.ServiceContainer,
) {

	// Add health check route
	r.GET("/health", func(c *gin.Context) {
		c.String(200, "OK")
	})

	// Register public authentication routes
	registerAuthRoutes(r, cfg, services)

	// Setup API v1 routes with Auth Middleware, passing service interfaces
	setupAPIV1Routes(r, cfg, services)

	// Swagger routes (typically public or conditionally available)
	setupSwaggerRoutes(r, cfg)
}

// setupAPIV1Routes configures the /api/v1 group and delegates to specific entity route registrations
func setupAPIV1Routes(
	r *gin.Engine,
	cfg *config.Config,
	service *portssvc.ServiceContainer,
) {
	// Apply AuthMiddleware to the entire v1 group
	v1 := r.Group("/api/v1", middleware.AuthMiddleware(cfg.JWTSecret))

	// Delegate route registration to specific handlers, passing required services
	registerUserRoutes(v1, service.User)
	registerCurrencyRoutes(v1, service.Currency)
	registerExchangeRateRoutes(v1, service.ExchangeRate)
	registerWorkplaceRoutes(v1, service.Workplace, service.Journal, service.Account, service.Reporting)
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
