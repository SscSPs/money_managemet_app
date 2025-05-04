package repositories

// RepositoryProvider holds all repository interfaces needed by services.
// This makes passing dependencies to the service container constructor cleaner.
type RepositoryProvider struct {
	AccountRepo      AccountRepositoryWithTx
	CurrencyRepo     CurrencyRepositoryWithTx
	ExchangeRateRepo ExchangeRateRepositoryWithTx
	UserRepo         UserRepositoryWithTx
	JournalRepo      JournalRepositoryWithTx
	WorkplaceRepo    WorkplaceRepositoryWithTx
}
