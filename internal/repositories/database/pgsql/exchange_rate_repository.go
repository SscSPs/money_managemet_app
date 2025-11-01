package pgsql

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/SscSPs/money_managemet_app/internal/apperrors"
	"github.com/SscSPs/money_managemet_app/internal/core/domain"
	"github.com/SscSPs/money_managemet_app/internal/models"
	"github.com/SscSPs/money_managemet_app/internal/utils/mapping"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// PgxExchangeRateRepository implements the ports.ExchangeRateRepository interface using pgxpool.
type PgxExchangeRateRepository struct {
	BaseRepository
}

// NewPgxExchangeRateRepository creates a new PgxExchangeRateRepository.
func NewPgxExchangeRateRepository(db *pgxpool.Pool) *PgxExchangeRateRepository {
	return &PgxExchangeRateRepository{
		BaseRepository: BaseRepository{Pool: db},
	}
}

// FindExchangeRateByID retrieves an exchange rate by its ID.
func (r *PgxExchangeRateRepository) FindExchangeRateByID(ctx context.Context, rateID string) (*domain.ExchangeRate, error) {
	return r.GetExchangeRateByID(ctx, rateID)
}

// SaveExchangeRate inserts or updates an exchange rate.
func (r *PgxExchangeRateRepository) SaveExchangeRate(ctx context.Context, rate domain.ExchangeRate) error {
	// Normalize currency codes to uppercase
	fromCurrency := strings.ToUpper(rate.FromCurrencyCode)
	toCurrency := strings.ToUpper(rate.ToCurrencyCode)

	// Validate we're not saving a rate with the same from and to currency
	if fromCurrency == toCurrency {
		return apperrors.NewValidationError("from and to currencies cannot be the same")
	}

	modelRate := mapping.ToModelExchangeRate(rate)
	modelRate.FromCurrencyCode = fromCurrency
	modelRate.ToCurrencyCode = toCurrency

	// Start a transaction if not already in one
	tx, err := r.Begin(ctx)
	if err != nil {
		return apperrors.NewAppError(500, "failed to begin transaction", err)
	}

	// Check if a rate already exists for this currency pair and date
	var existingID string
	err = tx.QueryRow(ctx,
		`SELECT exchange_rate_id FROM exchange_rates 
		WHERE from_currency_code = $1 AND to_currency_code = $2 AND date_effective = $3`,
		fromCurrency, toCurrency, rate.DateEffective,
	).Scan(&existingID)

	// If we found an existing rate, update it
	if err == nil && existingID != "" {
		_, err = tx.Exec(ctx, `
			UPDATE exchange_rates 
			SET rate = $1, last_updated_at = $2, last_updated_by = $3
			WHERE exchange_rate_id = $4`,
			modelRate.Rate, modelRate.LastUpdatedAt, modelRate.LastUpdatedBy, existingID,
		)
	} else if errors.Is(err, pgx.ErrNoRows) {
		// No existing rate, insert a new one
		_, err = tx.Exec(ctx, `
			INSERT INTO exchange_rates (
				exchange_rate_id, from_currency_code, to_currency_code, rate, date_effective,
				created_at, created_by, last_updated_at, last_updated_by
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			modelRate.ExchangeRateID, modelRate.FromCurrencyCode, modelRate.ToCurrencyCode,
			modelRate.Rate, modelRate.DateEffective, modelRate.CreatedAt,
			modelRate.CreatedBy, modelRate.LastUpdatedAt, modelRate.LastUpdatedBy,
		)
	}

	if err != nil {
		_ = r.Rollback(ctx, tx)
		return apperrors.NewAppError(500, "failed to save exchange rate", err)
	}

	r.Commit(ctx, tx)
	return nil
}

// FindExchangeRate retrieves the most recent exchange rate between two currencies.
func (r *PgxExchangeRateRepository) FindExchangeRate(ctx context.Context, fromCurrencyCode, toCurrencyCode string) (*domain.ExchangeRate, error) {
	// Normalize currency codes
	fromCurrency := strings.ToUpper(fromCurrencyCode)
	toCurrency := strings.ToUpper(toCurrencyCode)

	// If the currencies are the same, return a 1:1 rate
	if fromCurrency == toCurrency {
		rate := decimal.NewFromInt(1)
		now := time.Now().Truncate(24 * time.Hour)
		return &domain.ExchangeRate{
			FromCurrencyCode: fromCurrency,
			ToCurrencyCode:   toCurrency,
			Rate:             rate,
			DateEffective:    now,
		}, nil
	}

	// First try to find the direct rate
	directRate, err := r.findRate(ctx, fromCurrency, toCurrency)
	if err == nil {
		return directRate, nil
	}

	// If direct rate not found, try to find the inverse rate
	if errors.Is(err, apperrors.ErrNotFound) {
		inverseRate, inverseErr := r.findRate(ctx, toCurrency, fromCurrency)
		if inverseErr == nil {
			// Calculate the inverse rate
			inverseRate.FromCurrencyCode = fromCurrency
			inverseRate.ToCurrencyCode = toCurrency
			if !inverseRate.Rate.IsZero() {
				inverseRate.Rate = decimal.NewFromInt(1).Div(inverseRate.Rate)
			}
			return inverseRate, nil
		}
	}

	return nil, apperrors.NewNotFoundError("no exchange rate found for currency pair " + fromCurrency + " to " + toCurrency)
}

// findRate is a helper method to find the most recent exchange rate
func (r *PgxExchangeRateRepository) findRate(ctx context.Context, fromCurrency, toCurrency string) (*domain.ExchangeRate, error) {
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
	err := r.Pool.QueryRow(ctx, query, fromCurrency, toCurrency).Scan(
		&modelRate.ExchangeRateID, &modelRate.FromCurrencyCode, &modelRate.ToCurrencyCode,
		&modelRate.Rate, &modelRate.DateEffective, &modelRate.CreatedAt,
		&modelRate.CreatedBy, &modelRate.LastUpdatedAt, &modelRate.LastUpdatedBy,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFoundError("exchange rate not found")
		}
		return nil, apperrors.NewAppError(500, "failed to find exchange rate", err)
	}

	domainRate := mapping.ToDomainExchangeRate(modelRate)
	return &domainRate, nil
}

// FindExchangeRateByIDs gets rates by ids
func (r *PgxExchangeRateRepository) FindExchangeRateByIDs(ctx context.Context, rateIDs []string) ([]domain.ExchangeRate, error) {
	query := `
		SELECT
			exchange_rate_id, from_currency_code, to_currency_code, rate, date_effective,
			created_at, created_by, last_updated_at, last_updated_by
		FROM exchange_rates
		WHERE exchange_rate_id IN ($1);
	`

	rows, err := r.Pool.Query(ctx, query, rateIDs)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFoundError("exchange rates not found")
		}
		return nil, apperrors.NewAppError(500, "failed to find exchange rates", err)
	}

	defer rows.Close()

	var modelRates []models.ExchangeRate
	for rows.Next() {
		var modelRate models.ExchangeRate
		err := rows.Scan(
			&modelRate.ExchangeRateID, &modelRate.FromCurrencyCode, &modelRate.ToCurrencyCode,
			&modelRate.Rate, &modelRate.DateEffective, &modelRate.CreatedAt,
			&modelRate.CreatedBy, &modelRate.LastUpdatedAt, &modelRate.LastUpdatedBy,
		)
		if err != nil {
			return nil, apperrors.NewAppError(500, "failed to scan exchange rate", err)
		}
		modelRates = append(modelRates, modelRate)
	}
	domainRates := make([]domain.ExchangeRate, len(modelRates))
	for i, modelRate := range modelRates {
		domainRates[i] = mapping.ToDomainExchangeRate(modelRate)
	}
	return domainRates, nil
}

// GetExchangeRateByID retrieves an exchange rate by its ID.
func (r *PgxExchangeRateRepository) GetExchangeRateByID(ctx context.Context, rateID string) (*domain.ExchangeRate, error) {
	query := `
		SELECT
			exchange_rate_id, from_currency_code, to_currency_code, rate, date_effective,
			created_at, created_by, last_updated_at, last_updated_by
		FROM exchange_rates
		WHERE exchange_rate_id = $1;
	`

	var modelRate models.ExchangeRate
	err := r.Pool.QueryRow(ctx, query, rateID).Scan(
		&modelRate.ExchangeRateID, &modelRate.FromCurrencyCode, &modelRate.ToCurrencyCode,
		&modelRate.Rate, &modelRate.DateEffective, &modelRate.CreatedAt,
		&modelRate.CreatedBy, &modelRate.LastUpdatedAt, &modelRate.LastUpdatedBy,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFoundError("exchange rate with ID " + rateID + " not found")
		}
		return nil, apperrors.NewAppError(500, "failed to get exchange rate by ID", err)
	}

	domainRate := mapping.ToDomainExchangeRate(modelRate)
	return &domainRate, nil
}

// ListExchangeRates retrieves all exchange rates with optional filtering.
func (r *PgxExchangeRateRepository) ListExchangeRates(
	ctx context.Context,
	fromCurrency, toCurrency *string,
	effectiveDate *time.Time,
	page, pageSize int,
) ([]domain.ExchangeRate, int, error) {
	// Build the base query and count query
	baseQuery := `FROM exchange_rates WHERE 1=1`
	args := []interface{}{}
	argNum := 1

	// Add filters
	if fromCurrency != nil {
		baseQuery += fmt.Sprintf(" AND from_currency_code = $%d", argNum)
		args = append(args, strings.ToUpper(*fromCurrency))
		argNum++
	}

	if toCurrency != nil {
		baseQuery += fmt.Sprintf(" AND to_currency_code = $%d", argNum)
		args = append(args, strings.ToUpper(*toCurrency))
		argNum++
	}

	if effectiveDate != nil {
		baseQuery += fmt.Sprintf(" AND date_effective <= $%d", argNum)
		args = append(args, effectiveDate.Truncate(24*time.Hour))
		argNum++
	}

	// Get total count
	var total int
	countQuery := "SELECT COUNT(*) " + baseQuery
	err := r.Pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, apperrors.NewAppError(500, "failed to count exchange rates", err)
	}

	// If no results, return early
	if total == 0 {
		return []domain.ExchangeRate{}, 0, nil
	}

	// Get paginated results
	baseQuery += " ORDER BY from_currency_code, to_currency_code, date_effective DESC"
	if pageSize > 0 {
		offset := (page - 1) * pageSize
		baseQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argNum, argNum+1)
		args = append(args, pageSize, offset)
	}

	rows, err := r.Pool.Query(ctx, "SELECT * "+baseQuery, args...)
	if err != nil {
		return nil, 0, apperrors.NewAppError(500, "failed to list exchange rates", err)
	}
	defer rows.Close()

	var rates []domain.ExchangeRate
	for rows.Next() {
		var modelRate models.ExchangeRate
		err := rows.Scan(
			&modelRate.ExchangeRateID, &modelRate.FromCurrencyCode, &modelRate.ToCurrencyCode,
			&modelRate.Rate, &modelRate.DateEffective, &modelRate.CreatedAt,
			&modelRate.CreatedBy, &modelRate.LastUpdatedAt, &modelRate.LastUpdatedBy,
		)
		if err != nil {
			return nil, 0, apperrors.NewAppError(500, "failed to scan exchange rate", err)
		}
		rates = append(rates, mapping.ToDomainExchangeRate(modelRate))
	}

	if err := rows.Err(); err != nil {
		return nil, 0, apperrors.NewAppError(500, "error iterating exchange rates", err)
	}

	return rates, total, nil
}
