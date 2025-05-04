package services

import (
	portsrepo "github.com/SscSPs/money_managemet_app/internal/core/ports/repositories"
	portssvc "github.com/SscSPs/money_managemet_app/internal/core/ports/services"
)

// Container holds all the services and manages their dependencies
type Container struct {
	Account      portssvc.AccountSvcFacade
	Currency     portssvc.CurrencySvcFacade
	ExchangeRate portssvc.ExchangeRateSvcFacade
	User         portssvc.UserSvcFacade
	Journal      portssvc.JournalSvcFacade
	Workplace    portssvc.WorkplaceSvcFacade
}

// NewContainer creates a new service container with properly initialized dependencies
func NewContainer(repos *portsrepo.RepositoryProvider) *Container {
	// Create the container structure first
	container := &Container{}

	// Initialize workplace service first since other services depend on it
	container.Workplace = NewWorkplaceService(
		repos.WorkplaceRepo,
		repos.CurrencyRepo,
	)

	// Create workplace authorizer for service dependencies
	workplaceAuthorizer := container.Workplace.(portssvc.WorkplaceAuthorizerSvc)

	// Create account service with dependencies
	container.Account = NewAccountServiceImpl(
		repos.AccountRepo,
		WithWorkplaceServiceImpl(container.Workplace.(portssvc.WorkplaceReaderSvc)),
		WithWorkplaceAuthorizerImpl(workplaceAuthorizer),
		WithCurrencyRepositoryImpl(repos.CurrencyRepo),
	)

	// Other services would be initialized similarly
	// container.Currency = NewCurrencyService(...)
	// container.ExchangeRate = NewExchangeRateService(...)
	// container.User = NewUserService(...)
	// container.Journal = NewJournalService(...)

	return container
}

// Helper to check interface implementations at compile time
var (
	_ portssvc.AccountSvcFacade   = (*accountService)(nil)
	_ portssvc.WorkplaceSvcFacade = (*workplaceService)(nil)
	// Add other implementation checks as services are created
)
