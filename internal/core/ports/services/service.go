package services

// ServiceContainer holds instances of all the application services.
// This is the main entry point for accessing service functionality and
// is used throughout the application, particularly in the handlers.
type ServiceContainer struct {
	Account            AccountSvcFacade
	Currency           CurrencySvcFacade
	ExchangeRate       ExchangeRateSvcFacade
	User               UserSvcFacade
	Journal            JournalSvcFacade
	Workplace          WorkplaceSvcFacade
	Reporting          ReportingService
	TokenService       TokenSvcFacade
	GoogleOAuthHandler GoogleOAuthHandlerSvcFacade
}
