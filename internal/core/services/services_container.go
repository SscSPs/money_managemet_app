package services

import (
	portsrepo "github.com/SscSPs/money_managemet_app/internal/core/ports/repositories"
	portssvc "github.com/SscSPs/money_managemet_app/internal/core/ports/services"
)

// NewServiceContainer initializes all services and returns a container holding them.
// It resolves dependencies between services.
func NewServiceContainer(repos portsrepo.RepositoryProvider) *portssvc.ServiceContainer {
	// Initialize services with only repository dependencies first
	accountSvc := NewAccountService(repos.AccountRepo)
	currencySvc := NewCurrencyService(repos.CurrencyRepo)
	userSvc := NewUserService(repos.UserRepo)
	workplaceSvc := NewWorkplaceService(repos.WorkplaceRepo, repos.CurrencyRepo)
	// Add StaticDataService initialization if/when implemented

	// Initialize services that depend on other services
	exchangeRateSvc := NewExchangeRateService(repos.ExchangeRateRepo, currencySvc) // Depends on CurrencyService
	journalSvc := NewJournalService(repos.JournalRepo, accountSvc, workplaceSvc)   // Depends on AccountRepo, JournalRepo, WorkplaceService

	return &portssvc.ServiceContainer{
		Account:      accountSvc,
		Currency:     currencySvc,
		ExchangeRate: exchangeRateSvc,
		User:         userSvc,
		Journal:      journalSvc,
		Workplace:    workplaceSvc,
	}
}
