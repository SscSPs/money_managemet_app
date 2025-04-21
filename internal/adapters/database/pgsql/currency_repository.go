package pgsql

import (
	"context"
	"fmt"
	"time"

	"github.com/SscSPs/money_managemet_app/internal/core/ports"
	"github.com/SscSPs/money_managemet_app/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type currencyRepository struct {
	pool *pgxpool.Pool
}

// NewCurrencyRepository creates a new repository for currency data.
func NewCurrencyRepository(pool *pgxpool.Pool) ports.CurrencyRepository {
	return &currencyRepository{pool: pool}
}

// SaveCurrency inserts or updates a currency (primarily for initial setup).
// Assumes CurrencyCode is the unique identifier.
func (r *currencyRepository) SaveCurrency(ctx context.Context, currency models.Currency) error {
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

// FindCurrencyByCode retrieves a currency by its code.
func (r *currencyRepository) FindCurrencyByCode(ctx context.Context, currencyCode string) (*models.Currency, error) {
	query := `
		SELECT currency_code, symbol, name, created_at, created_by, last_updated_at, last_updated_by
		FROM currencies
		WHERE currency_code = $1;
	`
	var c models.Currency
	err := r.pool.QueryRow(ctx, query, currencyCode).Scan(
		&c.CurrencyCode,
		&c.Symbol,
		&c.Name,
		&c.CreatedAt,
		&c.CreatedBy,
		&c.LastUpdatedAt,
		&c.LastUpdatedBy,
	)

	if err != nil {
		// TODO: Handle pgx.ErrNoRows specifically if needed
		return nil, fmt.Errorf("failed to find currency by code %s: %w", currencyCode, err)
	}
	return &c, nil
}

// ListCurrencies retrieves all currencies.
func (r *currencyRepository) ListCurrencies(ctx context.Context) ([]models.Currency, error) {
	query := `
		SELECT currency_code, symbol, name, created_at, created_by, last_updated_at, last_updated_by
		FROM currencies ORDER BY currency_code;
	`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list currencies: %w", err)
	}
	defer rows.Close()

	currencies := []models.Currency{}
	for rows.Next() {
		var c models.Currency
		if err := rows.Scan(
			&c.CurrencyCode,
			&c.Symbol,
			&c.Name,
			&c.CreatedAt,
			&c.CreatedBy,
			&c.LastUpdatedAt,
			&c.LastUpdatedBy,
		); err != nil {
			return nil, fmt.Errorf("failed to scan currency row: %w", err)
		}
		currencies = append(currencies, c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating currency rows: %w", err)
	}

	return currencies, nil
}
