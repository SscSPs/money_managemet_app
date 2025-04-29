package handlers

import (
	"github.com/SscSPs/money_managemet_app/cmd/docs"
	"github.com/SscSPs/money_managemet_app/internal/core/services"
	"github.com/SscSPs/money_managemet_app/internal/middleware"
	"github.com/SscSPs/money_managemet_app/internal/platform/config"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// RegisterRoutes sets up all application routes, injecting dependencies
func RegisterRoutes(
	r *gin.Engine,
	cfg *config.Config,
	userService services.UserService, // Renamed for clarity
	accountService services.AccountService,
	currencyService services.CurrencyService,
	exchangeRateService services.ExchangeRateService,
	journalService services.JournalService,
	workplaceService services.WorkplaceService,
) {
	// Register public authentication routes (Auth might need its own service later)
	registerAuthRoutes(r, cfg, userService) // Auth handler likely needs UserService

	// Setup API v1 routes with Auth Middleware, passing services
	setupAPIV1Routes(r, cfg, userService, accountService, currencyService, exchangeRateService, journalService, workplaceService)

	// Swagger routes (typically public or conditionally available)
	setupSwaggerRoutes(r, cfg)
}

// setupAPIV1Routes configures the /api/v1 group and delegates to specific entity route registrations
func setupAPIV1Routes(
	r *gin.Engine,
	cfg *config.Config,
	userService services.UserService,
	accountService services.AccountService,
	currencyService services.CurrencyService,
	exchangeRateService services.ExchangeRateService,
	journalService services.JournalService,
	workplaceService services.WorkplaceService,
) {
	// Apply AuthMiddleware to the entire v1 group
	v1 := r.Group("/api/v1", middleware.AuthMiddleware(cfg.JWTSecret))

	// Delegate route registration to specific handlers, passing required services
	// registerJournalRoutes(v1, journalService)           // REMOVED - Will be nested under workplaces
	registerAccountRoutes(v1, accountService)                     // Pass AccountService
	registerUserRoutes(v1, userService)                           // Pass UserService
	registerCurrencyRoutes(v1, currencyService)                   // Pass CurrencyService
	registerExchangeRateRoutes(v1, exchangeRateService)           // Pass ExchangeRateService
	registerWorkplaceRoutes(v1, workplaceService, journalService) // Pass WorkplaceService AND JournalService
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
