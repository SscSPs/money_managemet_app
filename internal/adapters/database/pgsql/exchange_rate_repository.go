package pgsql

import (
	"context"
	"errors"
	"fmt"

	"github.com/SscSPs/money_managemet_app/internal/apperrors"
	"github.com/SscSPs/money_managemet_app/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PgxExchangeRateRepository implements the ports.ExchangeRateRepository interface using pgxpool.
type PgxExchangeRateRepository struct {
	db *pgxpool.Pool
}

// NewExchangeRateRepository creates a new PgxExchangeRateRepository.
func NewExchangeRateRepository(db *pgxpool.Pool) *PgxExchangeRateRepository {
	return &PgxExchangeRateRepository{db: db}
}

// Create inserts a new exchange rate into the database.
func (r *PgxExchangeRateRepository) SaveExchangeRate(ctx context.Context, rate models.ExchangeRate) error {
	query := `
		INSERT INTO exchange_rates (
			exchange_rate_id, from_currency_code, to_currency_code, rate, date_effective,
			created_at, created_by, last_updated_at, last_updated_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.db.Exec(ctx, query,
		rate.ExchangeRateID, rate.FromCurrencyCode, rate.ToCurrencyCode, rate.Rate, rate.DateEffective,
		rate.CreatedAt, rate.CreatedBy, rate.LastUpdatedAt, rate.LastUpdatedBy,
	)
	if err != nil {
		// TODO: Add specific error handling for constraint violations (e.g., duplicate key)
		// if pgErr, ok := err.(*pgconn.PgError); ok {
		//  if pgErr.Code == "23505" { // unique_violation
		//      return apperrors.ErrDuplicate // Or a more specific error
		//  }
		// }
		return fmt.Errorf("error inserting exchange rate: %w", err)
	}
	return nil
}

// FindExchangeRate retrieves a specific exchange rate from the database.
func (r *PgxExchangeRateRepository) FindExchangeRate(ctx context.Context, fromCode, toCode string) (*models.ExchangeRate, error) {
	query := `
		SELECT
			exchange_rate_id, from_currency_code, to_currency_code, rate, date_effective,
			created_at, created_by, last_updated_at, last_updated_by
		FROM exchange_rates
		WHERE from_currency_code = $1 AND to_currency_code = $2
	`
	// Assuming date_effective is stored as DATE or TIMESTAMP WITHOUT TIME ZONE.
	// If it includes time, the query might need adjustment (e.g., casting to date).
	// For simplicity, using exact match here.
	rate := &models.ExchangeRate{}
	err := r.db.QueryRow(ctx, query, fromCode, toCode).Scan(
		&rate.ExchangeRateID, &rate.FromCurrencyCode, &rate.ToCurrencyCode, &rate.Rate, &rate.DateEffective,
		&rate.CreatedAt, &rate.CreatedBy, &rate.LastUpdatedAt, &rate.LastUpdatedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrNotFound // Use custom not found error
		}
		return nil, fmt.Errorf("error finding exchange rate: %w", err)
	}
	return rate, nil
}
