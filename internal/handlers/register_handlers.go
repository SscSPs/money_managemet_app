package handlers

import (
	"github.com/SscSPs/money_managemet_app/cmd/docs"
	portssvc "github.com/SscSPs/money_managemet_app/internal/core/ports/services" // Use ports services import

	// "github.com/SscSPs/money_managemet_app/internal/core/services" // Remove concrete services import
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
	userService portssvc.UserService, // Use interface type
	accountService portssvc.AccountService, // Use interface type
	currencyService portssvc.CurrencyService, // Use interface type
	exchangeRateService portssvc.ExchangeRateService, // Use interface type
	journalService portssvc.JournalService, // Use interface type
	workplaceService portssvc.WorkplaceService, // Use interface type
) {
	// Register public authentication routes
	registerAuthRoutes(r, cfg, userService)

	// Setup API v1 routes with Auth Middleware, passing service interfaces
	setupAPIV1Routes(r, cfg, userService, accountService, currencyService, exchangeRateService, journalService, workplaceService)

	// Swagger routes (typically public or conditionally available)
	setupSwaggerRoutes(r, cfg)
}

// setupAPIV1Routes configures the /api/v1 group and delegates to specific entity route registrations
func setupAPIV1Routes(
	r *gin.Engine,
	cfg *config.Config,
	userService portssvc.UserService, // Use interface type
	accountService portssvc.AccountService, // Use interface type
	currencyService portssvc.CurrencyService, // Use interface type
	exchangeRateService portssvc.ExchangeRateService, // Use interface type
	journalService portssvc.JournalService, // Use interface type
	workplaceService portssvc.WorkplaceService, // Use interface type
) {
	// Apply AuthMiddleware to the entire v1 group
	v1 := r.Group("/api/v1", middleware.AuthMiddleware(cfg.JWTSecret))

	// Delegate route registration to specific handlers, passing required services
	registerUserRoutes(v1, userService)
	registerCurrencyRoutes(v1, currencyService)
	registerExchangeRateRoutes(v1, exchangeRateService)
	registerWorkplaceRoutes(v1, workplaceService, journalService, accountService)
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
