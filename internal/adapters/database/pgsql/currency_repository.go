package pgsql

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/SscSPs/money_managemet_app/internal/apperrors"
	"github.com/SscSPs/money_managemet_app/internal/core/ports"
	"github.com/SscSPs/money_managemet_app/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PgxCurrencyRepository struct {
	pool *pgxpool.Pool
}

// NewPgxCurrencyRepository creates a new repository for currency data.
func NewPgxCurrencyRepository(pool *pgxpool.Pool) ports.CurrencyRepository {
	return &PgxCurrencyRepository{pool: pool}
}

// SaveCurrency inserts or updates a currency (primarily for initial setup).
// Assumes CurrencyCode is the unique identifier.
func (r *PgxCurrencyRepository) SaveCurrency(ctx context.Context, currency models.Currency) error {
	// In a real app, use proper UserID. Placeholder for M1.
	creatorUserID := currency.CreatedBy
	now := time.Now().UTC()

	// Simple UPSERT logic (adjust based on actual table schema/constraints)
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
		currency.CurrencyCode,
		currency.Symbol,
		currency.Name,
		now,           // created_at
		creatorUserID, // created_by
		now,           // last_updated_at
		creatorUserID, // last_updated_by
	)

	if err != nil {
		return fmt.Errorf("failed to save currency %s: %w", currency.CurrencyCode, err)
	}
	return nil
}

// FindCurrencyByCode retrieves a currency by its 3-letter code.
func (r *PgxCurrencyRepository) FindCurrencyByCode(ctx context.Context, currencyCode string) (*models.Currency, error) {
	query := `
		SELECT currency_code, symbol, name, created_at, created_by, last_updated_at, last_updated_by
		FROM currencies
		WHERE currency_code = $1;
	`
	var currency models.Currency
	err := r.pool.QueryRow(ctx, query, currencyCode).Scan(
		&currency.CurrencyCode,
		&currency.Symbol,
		&currency.Name,
		&currency.CreatedAt,
		&currency.CreatedBy,
		&currency.LastUpdatedAt,
		&currency.LastUpdatedBy,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Map db not found error to application specific error
			return nil, apperrors.ErrNotFound
		}
		// Wrap other potential errors
		return nil, fmt.Errorf("failed to find currency by code %s: %w", currencyCode, err)
	}

	return &currency, nil
}

// ListCurrencies retrieves all currencies.
func (r *PgxCurrencyRepository) ListCurrencies(ctx context.Context) ([]models.Currency, error) {
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

	currencies, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.Currency, error) {
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
		// Check specifically for ErrNoRows which might occur if CollectRows is used
		// although Query itself usually returns nil error and 0 rows in this case.
		// The pgx documentation isn't perfectly clear if CollectRows can return ErrNoRows.
		// Safest to check, but note it might be redundant.
		if errors.Is(err, pgx.ErrNoRows) {
			return []models.Currency{}, nil // Return empty slice, not an error
		}
		return nil, fmt.Errorf("failed to scan currencies: %w", err)
	}

	// Return empty slice if no rows found, CollectRows should handle this.
	return currencies, nil
}
