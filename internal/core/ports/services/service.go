package services

import (
	"context"
)

// ServiceContainer holds instances of all the application services.
// This is the main entry point for accessing service functionality and
// is used throughout the application, particularly in the handlers.
type ServiceContainer struct {
	Account      AccountSvcFacade
	Currency     CurrencySvcFacade
	ExchangeRate ExchangeRateSvcFacade
	User         UserSvcFacade
	Journal      JournalSvcFacade
	Workplace    WorkplaceSvcFacade
	Reporting    ReportingService
}

// Legacy monolithic service interfaces have been removed.
// The codebase has been refactored to use more specialized service interfaces
// defined in the following files:
//   - account_services.go: AccountSvcFacade and related interfaces
//   - journal_services.go: JournalSvcFacade and related interfaces
//   - workplace_services.go: WorkplaceSvcFacade and related interfaces
//   - user_services.go: UserSvcFacade and related interfaces
//   - currency_services.go: CurrencySvcFacade and ExchangeRateSvcFacade

// StaticDataService defines the interface for managing static data like currencies.
// This will be refactored in a future update.
type StaticDataService interface {
	InitializeStaticData(ctx context.Context) error
}

// Note: The commented-out TransactionService was never implemented and can be safely removed
