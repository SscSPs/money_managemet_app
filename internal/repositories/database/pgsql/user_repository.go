package pgsql

import (
	"context"
	"errors" // Import errors
	"time"   // Added for MarkUserDeleted

	"github.com/SscSPs/money_managemet_app/internal/apperrors" // Import apperrors
	"github.com/SscSPs/money_managemet_app/internal/core/domain"
	portsrepo "github.com/SscSPs/money_managemet_app/internal/core/ports/repositories"
	"github.com/SscSPs/money_managemet_app/internal/models"
	"github.com/SscSPs/money_managemet_app/internal/utils/mapping"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PgxUserRepository struct {
	BaseRepository
}

func newPgxUserRepository(db *pgxpool.Pool) portsrepo.UserRepositoryWithTx {
	return &PgxUserRepository{
		BaseRepository: BaseRepository{Pool: db},
	}
}

var FULL_USERS_SELECT_QUERY = `
SELECT 
	user_id, username, email, password_hash,
	name, auth_provider, provider_user_id,
	created_at, created_by, last_updated_at,
	last_updated_by, deleted_at, refresh_token_hash,
	refresh_token_expiry_time, version 
FROM users
`

// getUsers private func to get user from the select query filters
func (r *PgxUserRepository) getUsers(ctx context.Context, filterQuery string, args ...any) ([]domain.User, error) {
	query := FULL_USERS_SELECT_QUERY + filterQuery
	rows, err := r.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, apperrors.NewAppError(500, "failed to query users", err)
	}
	defer rows.Close()
	modelUsers, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.User])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) { // It's possible to get no rows, which is not an error for a list.
			return []domain.User{}, nil
		}
		return nil, apperrors.NewAppError(500, "failed to collect user rows", err)
	}

	return mapping.ToDomainUserSlice(modelUsers), nil
}

// FindUserByUsername fetches a user by username and maps to domain.User
func (r *PgxUserRepository) FindUserByUsername(ctx context.Context, username string) (*domain.User, error) {
	query := ` WHERE username = $1 AND deleted_at IS NULL LIMIT 1`
	users, err := r.getUsers(ctx, query, username)
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, apperrors.ErrNotFound
	}
	return &users[0], nil
}

// FindUserByEmail fetches a user by email and maps to domain.User
func (r *PgxUserRepository) FindUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := ` WHERE email = $1 AND deleted_at IS NULL LIMIT 1`
	users, err := r.getUsers(ctx, query, email)
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, apperrors.ErrNotFound
	}
	return &users[0], nil
}

// FindUserByProvider fetches a user by auth provider and provider user ID, then maps to domain.User
func (r *PgxUserRepository) FindUserByProviderDetails(ctx context.Context, authProvider string, providerUserID string) (*domain.User, error) {
	query := ` WHERE auth_provider = $1 AND provider_user_id = $2 AND deleted_at IS NULL LIMIT 1`
	users, err := r.getUsers(ctx, query, authProvider, providerUserID)
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, apperrors.ErrNotFound
	}
	return &users[0], nil
}

func (r *PgxUserRepository) SaveUser(ctx context.Context, user *domain.User) error {
	modelUser := mapping.ToModelUser(*user)
	query := `
        INSERT INTO users (user_id, username, email, password_hash, name, auth_provider, provider_user_id, created_at, created_by, last_updated_at, last_updated_by, refresh_token_hash, refresh_token_expiry_time, version)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, 1);
    `
	_, err := r.Pool.Exec(ctx, query,
		modelUser.UserID,
		modelUser.Username,
		modelUser.Email, // Added email to insert and update
		modelUser.PasswordHash,
		modelUser.Name,
		modelUser.AuthProvider,   // Added auth_provider
		modelUser.ProviderUserID, // Added provider_user_id
		modelUser.CreatedAt,
		modelUser.CreatedBy,
		modelUser.LastUpdatedAt,
		modelUser.LastUpdatedBy,
		modelUser.RefreshTokenHash,
		modelUser.RefreshTokenExpiryTime,
	)
	if err != nil {
		// Check for unique constraint violation on email if necessary here, though the query might handle it
		// e.g., if strings.Contains(err.Error(), "unique constraint") && strings.Contains(err.Error(), "users_email_key") { ... }
		return apperrors.NewAppError(500, "failed to save user", err)
	}
	return nil
}

func (r *PgxUserRepository) FindUserByID(ctx context.Context, userID string) (*domain.User, error) {
	query := ` WHERE user_id = $1 AND deleted_at IS NULL LIMIT 1`
	users, err := r.getUsers(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, apperrors.ErrNotFound
	}
	return &users[0], nil
}

func (r *PgxUserRepository) FindUsers(ctx context.Context, limit int, offset int) ([]domain.User, error) {
	// Default limit if not specified or invalid
	if limit <= 0 {
		limit = 20
	}
	// Ensure offset is non-negative
	if offset < 0 {
		offset = 0
	}

	query := ` WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	users, err := r.getUsers(ctx, query, limit, offset)
	if err != nil {
		return nil, apperrors.NewAppError(500, "failed to query users", err)
	}

	return users, nil
}

func (r *PgxUserRepository) UpdateUser(ctx context.Context, existingUser *domain.User) error {
	modelUser := mapping.ToModelUser(*existingUser)
	query := `
        UPDATE users
        SET name = $1, email = $2, username = $3, auth_provider = $4, provider_user_id = $5, last_updated_at = $6, last_updated_by = $7, version = version + 1
        WHERE user_id = $8 AND version = $9;
    `
	cmdTag, err := r.Pool.Exec(ctx, query,
		modelUser.Name,
		modelUser.Email,          // Added email
		modelUser.Username,       // Added username
		modelUser.AuthProvider,   // Added auth_provider
		modelUser.ProviderUserID, // Added provider_user_id
		modelUser.LastUpdatedAt,
		modelUser.LastUpdatedBy,
		modelUser.UserID,
		modelUser.Version,
	)
	if err != nil {
		return apperrors.NewAppError(500, "failed to execute update user query", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return apperrors.NewAppError(409, "optimistic lock failed for user "+existingUser.UserID, nil)
	}
	return nil
}

func (r *PgxUserRepository) MarkUserDeleted(ctx context.Context, existingUser *domain.User, deletedBy string) error {
	query := `
        UPDATE users
        SET deleted_at = $1, last_updated_at = $1, last_updated_by = $2, version = version + 1
        WHERE user_id = $3 AND version = $4;
    `
	cmdTag, err := r.Pool.Exec(ctx, query, time.Now(), deletedBy, existingUser.UserID, existingUser.Version)
	if err != nil {
		return apperrors.NewAppError(500, "failed to mark user as deleted", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return apperrors.NewAppError(409, "optimistic lock failed for user "+existingUser.UserID, nil)
	}
	return nil
}

func (r *PgxUserRepository) UpdateRefreshToken(ctx context.Context, existingUser *domain.User, refreshTokenHash string, refreshTokenExpiryTime time.Time) error {
	now := time.Now()
	query := `
		UPDATE users 
		SET refresh_token_hash = $1, refresh_token_expiry_time = $2, last_updated_at = $3, last_updated_by = $4, version = version + 1
		WHERE user_id = $5 AND version = $6;
	`
	cmdTag, err := r.Pool.Exec(ctx, query, refreshTokenHash, refreshTokenExpiryTime, now, existingUser.UserID, existingUser.UserID, existingUser.Version)
	if err != nil {
		return apperrors.NewAppError(500, "failed to update refresh token for user "+existingUser.UserID, err)
	}
	if cmdTag.RowsAffected() == 0 {
		return apperrors.NewAppError(409, "optimistic lock failed for user "+existingUser.UserID, nil)
	}
	return nil
}

func (r *PgxUserRepository) ClearRefreshToken(ctx context.Context, existingUser *domain.User) error {
	query := `
		UPDATE users
		SET refresh_token_hash = NULL, refresh_token_expiry_time = NULL, version = version + 1
		WHERE user_id = $1 AND version = $2;
	`
	cmdTag, err := r.Pool.Exec(ctx, query, existingUser.UserID, existingUser.Version)
	if err != nil {
		return apperrors.NewAppError(500, "failed to clear refresh token for user "+existingUser.UserID, err)
	}
	if cmdTag.RowsAffected() == 0 {
		return apperrors.NewAppError(409, "optimistic lock failed for user "+existingUser.UserID, nil)
	}
	return nil
}
