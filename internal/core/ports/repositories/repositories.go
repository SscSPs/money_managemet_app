package repositories

// RepositoryProvider holds all repository interfaces needed by services.
// This makes passing dependencies to the service container constructor cleaner.
type RepositoryProvider struct {
	AccountRepo      AccountRepositoryFacade
	CurrencyRepo     CurrencyRepositoryFacade
	ExchangeRateRepo ExchangeRateRepositoryFacade
	UserRepo         UserRepositoryFacade
	JournalRepo      JournalRepositoryFacade
	WorkplaceRepo    WorkplaceRepositoryFacade
}
