package handlers

import (
	"github.com/SscSPs/money_managemet_app/cmd/docs"
	portssvc "github.com/SscSPs/money_managemet_app/internal/core/ports/services"
	"github.com/SscSPs/money_managemet_app/internal/middleware"
	"github.com/SscSPs/money_managemet_app/internal/platform/config"
	"github.com/SscSPs/money_managemet_app/internal/utils"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// RegisterRoutes sets up all application routes, injecting dependencies using interfaces
func RegisterRoutes(
	r *gin.Engine,
	cfg *config.Config,
	services *portssvc.ServiceContainer,
	posthogClient *utils.PosthogClientWrapper,
) {

	// Add health check route
	r.GET("/health", func(c *gin.Context) {
		c.String(200, "OK")
	})

	// Register public authentication routes
	registerAuthRoutes(r, cfg, services)

	// Setup API v1 routes with Auth Middleware, passing service interfaces
	setupAPIV1Routes(r, cfg, services, posthogClient)

	// Swagger routes (typically public or conditionally available)
	setupSwaggerRoutes(r, cfg)
}

// setupAPIV1Routes configures the /api/v1 group and delegates to specific entity route registrations
func setupAPIV1Routes(
	r *gin.Engine,
	cfg *config.Config,
	service *portssvc.ServiceContainer,
	posthogClient *utils.PosthogClientWrapper,
) {
	// Create API v1 group with both JWT and API token authentication
	v1 := r.Group("/api/v1", middleware.APITokenAuth(service.APITokenSvc), middleware.AuthMiddleware(cfg.JWTSecret))

	// Register API token routes (protected by both JWT and API token auth)
	RegisterAPITokenRoutes(v1, service.APITokenSvc)

	// Delegate route registration to specific handlers, passing required services
	registerUserRoutes(v1, service.User)
	registerCurrencyRoutes(v1, service.Currency)
	registerExchangeRateRoutes(v1, service.ExchangeRate)
	registerWorkplaceRoutes(v1, service.Workplace, service.Journal, service.Account, service.Reporting, posthogClient)
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
