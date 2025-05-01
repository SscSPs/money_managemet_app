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
	"github.com/jackc/pgx/v5/pgxpool"
)

type PgxCurrencyRepository struct {
	pool *pgxpool.Pool
}

// newPgxCurrencyRepository creates a new repository for currency data.
func newPgxCurrencyRepository(pool *pgxpool.Pool) portsrepo.CurrencyRepository {
	return &PgxCurrencyRepository{pool: pool}
}

// Ensure implementation matches interface
var _ portsrepo.CurrencyRepository = (*PgxCurrencyRepository)(nil)

// Helper to convert domain.Currency to models.Currency
func toModelCurrency(d domain.Currency) models.Currency {
	return models.Currency{
		CurrencyCode: d.CurrencyCode,
		Symbol:       d.Symbol,
		Name:         d.Name,
		AuditFields: models.AuditFields{
			CreatedAt:     d.CreatedAt,
			CreatedBy:     d.CreatedBy,
			LastUpdatedAt: d.LastUpdatedAt,
			LastUpdatedBy: d.LastUpdatedBy,
		},
	}
}

// Helper to convert models.Currency to domain.Currency
func toDomainCurrency(m models.Currency) domain.Currency {
	return domain.Currency{
		CurrencyCode: m.CurrencyCode,
		Symbol:       m.Symbol,
		Name:         m.Name,
		AuditFields: domain.AuditFields{
			CreatedAt:     m.CreatedAt,
			CreatedBy:     m.CreatedBy,
			LastUpdatedAt: m.LastUpdatedAt,
			LastUpdatedBy: m.LastUpdatedBy,
		},
	}
}

// Helper to convert slice of models.Currency to slice of domain.Currency
func toDomainCurrencySlice(ms []models.Currency) []domain.Currency {
	ds := make([]domain.Currency, len(ms))
	for i, m := range ms {
		ds[i] = toDomainCurrency(m)
	}
	return ds
}

// SaveCurrency inserts or updates a currency (primarily for initial setup).
func (r *PgxCurrencyRepository) SaveCurrency(ctx context.Context, currency domain.Currency) error {
	modelCurr := toModelCurrency(currency)
	creatorUserID := modelCurr.CreatedBy

	query := `
		INSERT INTO currencies (currency_code, symbol, name, created_at, created_by, last_updated_at, last_updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (currency_code) DO UPDATE SET
			symbol = EXCLUDED.symbol,
			name = EXCLUDED.name,
			last_updated_at = EXCLUDED.last_updated_at,
			last_updated_by = EXCLUDED.last_updated_by;
	`

	_, err := r.pool.Exec(ctx, query,
		modelCurr.CurrencyCode,
		modelCurr.Symbol,
		modelCurr.Name,
		modelCurr.CreatedAt,
		creatorUserID,
		modelCurr.LastUpdatedAt,
		creatorUserID,
	)

	if err != nil {
		return fmt.Errorf("failed to save currency %s: %w", modelCurr.CurrencyCode, err)
	}
	return nil
}

// FindCurrencyByCode retrieves a currency by its 3-letter code.
func (r *PgxCurrencyRepository) FindCurrencyByCode(ctx context.Context, currencyCode string) (*domain.Currency, error) {
	query := `
		SELECT currency_code, symbol, name, created_at, created_by, last_updated_at, last_updated_by
		FROM currencies
		WHERE currency_code = $1;
	`
	var modelCurr models.Currency
	err := r.pool.QueryRow(ctx, query, currencyCode).Scan(
		&modelCurr.CurrencyCode,
		&modelCurr.Symbol,
		&modelCurr.Name,
		&modelCurr.CreatedAt,
		&modelCurr.CreatedBy,
		&modelCurr.LastUpdatedAt,
		&modelCurr.LastUpdatedBy,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrNotFound
		}
		return nil, fmt.Errorf("failed to find currency by code %s: %w", currencyCode, err)
	}

	domainCurr := toDomainCurrency(modelCurr)
	return &domainCurr, nil
}

// ListCurrencies retrieves all currencies.
func (r *PgxCurrencyRepository) ListCurrencies(ctx context.Context) ([]domain.Currency, error) {
	query := `
		SELECT currency_code, symbol, name, created_at, created_by, last_updated_at, last_updated_by
		FROM currencies
		ORDER BY currency_code;
	`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query currencies: %w", err)
	}
	defer rows.Close()

	modelCurrencies, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.Currency, error) {
		var currency models.Currency
		err := row.Scan(
			&currency.CurrencyCode,
			&currency.Symbol,
			&currency.Name,
			&currency.CreatedAt,
			&currency.CreatedBy,
			&currency.LastUpdatedAt,
			&currency.LastUpdatedBy,
		)
		return currency, err
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return []domain.Currency{}, nil // Return empty domain slice
		}
		return nil, fmt.Errorf("failed to scan currencies: %w", err)
	}

	return toDomainCurrencySlice(modelCurrencies), nil
}
