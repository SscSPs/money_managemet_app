package services

import (
	portsrepo "github.com/SscSPs/money_managemet_app/internal/core/ports/repositories"
	portssvc "github.com/SscSPs/money_managemet_app/internal/core/ports/services"
	"github.com/SscSPs/money_managemet_app/internal/platform/config"
)

// NewServiceContainer creates a new service container with properly initialized dependencies
func NewServiceContainer(cfg *config.Config, repos portsrepo.RepositoryProvider) *portssvc.ServiceContainer {
	// Create the container structure first
	container := &portssvc.ServiceContainer{}

	// Initialize workplace service first since other services depend on it
	container.Workplace = NewWorkplaceService(
		repos.WorkplaceRepo,
		repos.CurrencyRepo,
	)

	// Create workplace authorizer for service dependencies
	workplaceAuthorizer := container.Workplace.(portssvc.WorkplaceAuthorizerSvc)
	workplaceReader := container.Workplace.(portssvc.WorkplaceReaderSvc)

	// Create account service with dependencies using the new implementation
	container.Account = NewAccountService(
		repos.AccountRepo,
		WithWorkplaceService(workplaceReader),
		WithWorkplaceAuthorizer(workplaceAuthorizer),
		WithCurrencyRepository(repos.CurrencyRepo),
	)

	// Initialize other services using their original constructors for now
	container.Currency = NewCurrencyService(repos.CurrencyRepo)
	container.User = NewUserService(repos.UserRepo)
	container.ExchangeRate = NewExchangeRateService(repos.ExchangeRateRepo, container.Currency)
	container.Journal = NewJournalService(repos.JournalRepo, container.Account, container.Workplace)
	container.Reporting = NewReportingService(repos.ReportingRepo, WithReportingWorkplaceAuthorizer(container.Workplace))

	// Initialize TokenService
	container.TokenService = NewTokenService(cfg, container.User)

	// Initialize GoogleOAuthHandlerSvcFacade
	container.GoogleOAuthHandler = NewGoogleOAuthHandlerService(cfg)

	return container
}

// Helper to check interface implementations at compile time
var (
	_ portssvc.AccountSvcFacade   = (*accountService)(nil)
	_ portssvc.WorkplaceSvcFacade = (*workplaceService)(nil)
	// Add other implementation checks as services are created
)
