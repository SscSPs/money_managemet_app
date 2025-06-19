package pgsql

import (
	"context"
	"errors"
	"time"

	"github.com/SscSPs/money_managemet_app/internal/core/domain"
	portsrepo "github.com/SscSPs/money_managemet_app/internal/core/ports/repositories"
	"github.com/SscSPs/money_managemet_app/internal/models"
	"github.com/SscSPs/money_managemet_app/internal/utils/mapping"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PgxAPITokenRepository struct {
	BaseRepository
}

// newPgxAPITokenRepository creates a new instance of PgxAPITokenRepository
func newPgxAPITokenRepository(db *pgxpool.Pool) portsrepo.APITokenRepositoryWithTx {
	return &PgxAPITokenRepository{
		BaseRepository: BaseRepository{Pool: db},
	}
}

// queryRow is a helper method to execute a query that returns a single row
func (r *PgxAPITokenRepository) queryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return r.Pool.QueryRow(ctx, sql, args...)
}

// query is a helper method to execute a query that returns multiple rows
func (r *PgxAPITokenRepository) query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return r.Pool.Query(ctx, sql, args...)
}

// exec is a helper method to execute a query that doesn't return rows
func (r *PgxAPITokenRepository) exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	return r.Pool.Exec(ctx, sql, args...)
}

const (
	apiTokensTable = "api_tokens"

	selectAPITokenFields = `
		api_token_id, user_id, name, token_hash, 
		last_used_at, expires_at, created_at, updated_at
	`

	insertAPITokenQuery = `
		INSERT INTO ` + apiTokensTable + ` (
			user_id, name, token_hash, expires_at
		) VALUES ($1, $2, $3, $4)
		RETURNING ` + selectAPITokenFields

	findAPITokenByIDQuery = `
		SELECT ` + selectAPITokenFields + `
		FROM ` + apiTokensTable + `
		WHERE api_token_id = $1 AND deleted_at IS NULL
	`

	findAPITokenByUserIDQuery = `
		SELECT ` + selectAPITokenFields + `
		FROM ` + apiTokensTable + `
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
	`

	findAPITokenByHashQuery = `
		SELECT ` + selectAPITokenFields + `
		FROM ` + apiTokensTable + `
		WHERE token_hash = $1 AND deleted_at IS NULL
	`

	updateAPITokenQuery = `
		UPDATE ` + apiTokensTable + `
		SET 
			last_used_at = COALESCE($2, last_used_at),
			updated_at = NOW()
		WHERE api_token_id = $1
		RETURNING ` + selectAPITokenFields

	deleteAPITokenQuery = `
		UPDATE ` + apiTokensTable + `
		SET deleted_at = NOW()
		WHERE api_token_id = $1
	`

	deleteAPITokensByUserIDQuery = `
		UPDATE ` + apiTokensTable + `
		SET deleted_at = NOW()
		WHERE user_id = $1
	`

	deleteExpiredAPITokensQuery = `
		UPDATE ` + apiTokensTable + `
		SET deleted_at = NOW()
		WHERE expires_at < $1 AND deleted_at IS NULL
	`
)

// Create persists a new API token
func (r *PgxAPITokenRepository) Create(ctx context.Context, token *domain.APIToken) error {
	if token == nil {
		return errors.New("token cannot be nil")
	}

	modelToken := mapping.ToModelAPIToken(*token)

	row := r.queryRow(
		ctx,
		insertAPITokenQuery,
		modelToken.UserID,
		modelToken.Name,
		modelToken.TokenHash,
		modelToken.ExpiresAt,
	)

	var createdToken models.APIToken
	err := row.Scan(
		&createdToken.ID,
		&createdToken.UserID,
		&createdToken.Name,
		&createdToken.TokenHash,
		&createdToken.LastUsedAt,
		&createdToken.ExpiresAt,
		&createdToken.CreatedAt,
		&createdToken.UpdatedAt,
	)

	if err != nil {
		return err
	}

	// Update the original token with the generated values
	token.ID = createdToken.ID
	token.CreatedAt = createdToken.CreatedAt
	token.UpdatedAt = createdToken.UpdatedAt

	return nil
}

// FindByID retrieves an API token by its ID
func (r *PgxAPITokenRepository) FindByID(ctx context.Context, id string) (*domain.APIToken, error) {
	if id == "" {
		return nil, errors.New("id cannot be empty")
	}

	row := r.queryRow(ctx, findAPITokenByIDQuery, id)
	token, err := scanAPIToken(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("token not found")
		}
		return nil, err
	}

	domainToken := mapping.ToDomainAPIToken(*token)
	return &domainToken, nil
}

// FindByUserID retrieves all API tokens for a specific user
func (r *PgxAPITokenRepository) FindByUserID(ctx context.Context, userID string) ([]domain.APIToken, error) {
	if userID == "" {
		return nil, errors.New("user ID cannot be empty")
	}

	rows, err := r.query(ctx, findAPITokenByUserIDQuery, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []domain.APIToken
	for rows.Next() {
		token, err := scanAPIToken(rows)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, mapping.ToDomainAPIToken(*token))
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return tokens, nil
}

// FindByToken finds a token by its hash
func (r *PgxAPITokenRepository) FindByToken(ctx context.Context, tokenString string) (*domain.APIToken, error) {
	if tokenString == "" {
		return nil, errors.New("token string cannot be empty")
	}

	row := r.queryRow(ctx, findAPITokenByHashQuery, tokenString)
	token, err := scanAPIToken(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("token not found")
		}
		return nil, err
	}

	domainToken := mapping.ToDomainAPIToken(*token)
	return &domainToken, nil
}

// Update updates an existing API token
func (r *PgxAPITokenRepository) Update(ctx context.Context, token *domain.APIToken) error {
	if token == nil {
		return errors.New("token cannot be nil")
	}

	modelToken := mapping.ToModelAPIToken(*token)
	updatedAt := time.Now()

	result, err := r.exec(
		ctx,
		updateAPITokenQuery,
		modelToken.ID,
		modelToken.LastUsedAt,
	)

	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return errors.New("token not found or not updated")
	}

	// Update the token with the new updated_at timestamp
	token.UpdatedAt = updatedAt

	return nil
}

// Delete removes an API token by ID (soft delete)
func (r *PgxAPITokenRepository) Delete(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("id cannot be empty")
	}

	result, err := r.exec(ctx, deleteAPITokenQuery, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return errors.New("token not found or already deleted")
	}

	return nil
}

// DeleteByUserID removes all API tokens for a specific user (soft delete)
func (r *PgxAPITokenRepository) DeleteByUserID(ctx context.Context, userID string) error {
	if userID == "" {
		return errors.New("user ID cannot be empty")
	}

	_, err := r.exec(ctx, deleteAPITokensByUserIDQuery, userID)
	return err
}

// DeleteExpired removes all expired API tokens (soft delete)
func (r *PgxAPITokenRepository) DeleteExpired(ctx context.Context, before time.Time) (int64, error) {
	if before.IsZero() {
		return 0, errors.New("invalid time provided")
	}

	result, err := r.exec(ctx, deleteExpiredAPITokensQuery, before)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected(), nil
}

// WithTx returns a new repository instance with the given transaction
func (r *PgxAPITokenRepository) WithTx(tx interface{}) portsrepo.APITokenRepository {
	// TODO: Implement proper transaction support when BaseRepository is updated
	// For now, we'll just return the original repository as transactions aren't fully supported yet
	// This is a temporary workaround until we can properly implement transaction support
	return r
}

// scanAPIToken scans an API token from a row
func scanAPIToken(row pgx.Row) (*models.APIToken, error) {
	var token models.APIToken
	err := row.Scan(
		&token.ID,
		&token.UserID,
		&token.Name,
		&token.TokenHash,
		&token.LastUsedAt,
		&token.ExpiresAt,
		&token.CreatedAt,
		&token.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &token, nil
}
