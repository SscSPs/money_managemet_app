package pgsql

import (
	"context"
	"errors"
	"fmt"

	"github.com/SscSPs/money_managemet_app/internal/apperrors"
	"github.com/SscSPs/money_managemet_app/internal/core/domain"
	portsrepo "github.com/SscSPs/money_managemet_app/internal/core/ports/repositories"
	"github.com/SscSPs/money_managemet_app/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PgxExchangeRateRepository implements the ports.ExchangeRateRepository interface using pgxpool.
type PgxExchangeRateRepository struct {
	db *pgxpool.Pool
}

// NewExchangeRateRepository creates a new PgxExchangeRateRepository.
func NewExchangeRateRepository(db *pgxpool.Pool) portsrepo.ExchangeRateRepository {
	return &PgxExchangeRateRepository{db: db}
}

var _ portsrepo.ExchangeRateRepository = (*PgxExchangeRateRepository)(nil)

func toModelExchangeRate(d domain.ExchangeRate) models.ExchangeRate {
	return models.ExchangeRate{
		ExchangeRateID:   d.ExchangeRateID,
		FromCurrencyCode: d.FromCurrencyCode,
		ToCurrencyCode:   d.ToCurrencyCode,
		Rate:             d.Rate,
		DateEffective:    d.DateEffective,
		AuditFields: models.AuditFields{
			CreatedAt:     d.CreatedAt,
			CreatedBy:     d.CreatedBy,
			LastUpdatedAt: d.LastUpdatedAt,
			LastUpdatedBy: d.LastUpdatedBy,
		},
	}
}

func toDomainExchangeRate(m models.ExchangeRate) domain.ExchangeRate {
	return domain.ExchangeRate{
		ExchangeRateID:   m.ExchangeRateID,
		FromCurrencyCode: m.FromCurrencyCode,
		ToCurrencyCode:   m.ToCurrencyCode,
		Rate:             m.Rate,
		DateEffective:    m.DateEffective,
		AuditFields: domain.AuditFields{
			CreatedAt:     m.CreatedAt,
			CreatedBy:     m.CreatedBy,
			LastUpdatedAt: m.LastUpdatedAt,
			LastUpdatedBy: m.LastUpdatedBy,
		},
	}
}

// SaveExchangeRate inserts or updates an exchange rate.
func (r *PgxExchangeRateRepository) SaveExchangeRate(ctx context.Context, rate domain.ExchangeRate) error {
	modelRate := toModelExchangeRate(rate)
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
	_, err := r.db.Exec(ctx, query,
		modelRate.ExchangeRateID, modelRate.FromCurrencyCode, modelRate.ToCurrencyCode, modelRate.Rate, modelRate.DateEffective,
		modelRate.CreatedAt, modelRate.CreatedBy, modelRate.LastUpdatedAt, modelRate.LastUpdatedBy,
	)
	if err != nil {
		// Check for unique constraint violation (e.g., duplicate rate for pair/date)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // 23505 is unique_violation
			// Consider logging the specific constraint name if available (pgErr.ConstraintName)
			return fmt.Errorf("%w: exchange rate for this pair and date already exists", apperrors.ErrDuplicate)
		}
		return fmt.Errorf("failed to save exchange rate: %w", err)
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
	err := r.db.QueryRow(ctx, query, fromCurrencyCode, toCurrencyCode).Scan(
		&modelRate.ExchangeRateID, &modelRate.FromCurrencyCode, &modelRate.ToCurrencyCode, &modelRate.Rate, &modelRate.DateEffective,
		&modelRate.CreatedAt, &modelRate.CreatedBy, &modelRate.LastUpdatedAt, &modelRate.LastUpdatedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrNotFound
		}
		return nil, fmt.Errorf("failed to find exchange rate %s->%s: %w", fromCurrencyCode, toCurrencyCode, err)
	}
	domainRate := toDomainExchangeRate(modelRate)
	return &domainRate, nil
}
