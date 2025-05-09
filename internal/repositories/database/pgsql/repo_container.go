package pgsql

import (
	"log/slog"

	portsrepo "github.com/SscSPs/money_managemet_app/internal/core/ports/repositories"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewRepositoryProvider(dbPool *pgxpool.Pool, logger *slog.Logger) portsrepo.RepositoryProvider {
	accountRepo := newPgxAccountRepository(dbPool)
	currencyRepo := newPgxCurrencyRepository(dbPool)
	exchangeRateRepo := newPgxExchangeRateRepository(dbPool)
	userRepo := newPgxUserRepository(dbPool)
	journalRepo := newPgxJournalRepository(dbPool, accountRepo)
	workplaceRepo := newPgxWorkplaceRepository(dbPool)
	reportingRepo := newReportingRepository(dbPool)

	return portsrepo.RepositoryProvider{
		AccountRepo:      accountRepo,
		CurrencyRepo:     currencyRepo,
		ExchangeRateRepo: exchangeRateRepo,
		UserRepo:         userRepo,
		JournalRepo:      journalRepo,
		WorkplaceRepo:    workplaceRepo,
		ReportingRepo:    reportingRepo,
	}
}
