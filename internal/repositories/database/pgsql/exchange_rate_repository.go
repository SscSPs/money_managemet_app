package pgsql

import (
	"context"
	"errors"

	"github.com/SscSPs/money_managemet_app/internal/apperrors"
	"github.com/SscSPs/money_managemet_app/internal/core/domain"
	portsrepo "github.com/SscSPs/money_managemet_app/internal/core/ports/repositories"
	"github.com/SscSPs/money_managemet_app/internal/models"
	"github.com/SscSPs/money_managemet_app/internal/utils/mapping"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PgxExchangeRateRepository implements the ports.ExchangeRateRepository interface using pgxpool.
type PgxExchangeRateRepository struct {
	BaseRepository
}

// newPgxExchangeRateRepository creates a new PgxExchangeRateRepository.
func newPgxExchangeRateRepository(db *pgxpool.Pool) portsrepo.ExchangeRateRepositoryWithTx {
	return &PgxExchangeRateRepository{
		BaseRepository: BaseRepository{Pool: db},
	}
}

var _ portsrepo.ExchangeRateRepositoryWithTx = (*PgxExchangeRateRepository)(nil)

// SaveExchangeRate inserts or updates an exchange rate.
func (r *PgxExchangeRateRepository) SaveExchangeRate(ctx context.Context, rate domain.ExchangeRate) error {
	modelRate := mapping.ToModelExchangeRate(rate)
	query := `
		INSERT INTO exchange_rates (
			exchange_rate_id, from_currency_code, to_currency_code, rate, date_effective,
			created_at, created_by, last_updated_at, last_updated_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (from_currency_code, to_currency_code, date_effective) DO UPDATE SET -- Example conflict target
			rate = EXCLUDED.rate,
			last_updated_at = EXCLUDED.last_updated_at,
			last_updated_by = EXCLUDED.last_updated_by;
	`
	_, err := r.Pool.Exec(ctx, query,
		modelRate.ExchangeRateID, modelRate.FromCurrencyCode, modelRate.ToCurrencyCode, modelRate.Rate, modelRate.DateEffective,
		modelRate.CreatedAt, modelRate.CreatedBy, modelRate.LastUpdatedAt, modelRate.LastUpdatedBy,
	)
	if err != nil {
		// Check for unique constraint violation (e.g., duplicate rate for pair/date)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // 23505 is unique_violation
			// Consider logging the specific constraint name if available (pgErr.ConstraintName)
			return apperrors.NewConflictError("exchange rate for this pair and date already exists")
		}
		return apperrors.NewAppError(500, "failed to save exchange rate", err)
	}
	return nil
}

// FindExchangeRate retrieves a specific exchange rate from the database.
func (r *PgxExchangeRateRepository) FindExchangeRate(ctx context.Context, fromCurrencyCode, toCurrencyCode string) (*domain.ExchangeRate, error) {
	// Simplified: Finds the latest rate. Needs adjustment based on requirements (e.g., find rate for specific date).
	query := `
		SELECT
			exchange_rate_id, from_currency_code, to_currency_code, rate, date_effective,
			created_at, created_by, last_updated_at, last_updated_by
		FROM exchange_rates
		WHERE from_currency_code = $1 AND to_currency_code = $2
		ORDER BY date_effective DESC
		LIMIT 1;
	`
	var modelRate models.ExchangeRate
	err := r.Pool.QueryRow(ctx, query, fromCurrencyCode, toCurrencyCode).Scan(
		&modelRate.ExchangeRateID, &modelRate.FromCurrencyCode, &modelRate.ToCurrencyCode, &modelRate.Rate, &modelRate.DateEffective,
		&modelRate.CreatedAt, &modelRate.CreatedBy, &modelRate.LastUpdatedAt, &modelRate.LastUpdatedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFoundError("exchange rate not found")
		}
		return nil, apperrors.NewAppError(500, "failed to find exchange rate "+fromCurrencyCode+"->"+toCurrencyCode, err)
	}
	domainRate := mapping.ToDomainExchangeRate(modelRate)
	return &domainRate, nil
}
