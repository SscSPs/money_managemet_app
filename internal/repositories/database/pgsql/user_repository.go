package pgsql

import (
	"context"
	"errors" // Import errors
	"fmt"    // For error wrapping
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

// Ensure PgxUserRepository implements portsrepo.UserRepositoryWithTx
var _ portsrepo.UserRepositoryWithTx = (*PgxUserRepository)(nil)

// GetUserByUsername fetches a user by username
func (r *PgxUserRepository) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	query := `SELECT user_id, username, password_hash, name, created_at, created_by, last_updated_at, last_updated_by, deleted_at, refresh_token_hash, refresh_token_expiry_time FROM users WHERE username = $1 AND deleted_at IS NULL LIMIT 1`
	row := r.Pool.QueryRow(ctx, query, username)
	var user models.User
	err := row.Scan(
		&user.UserID, &user.Username, &user.PasswordHash, &user.Name, 
		&user.CreatedAt, &user.CreatedBy, &user.LastUpdatedAt, &user.LastUpdatedBy, 
		&user.DeletedAt, &user.RefreshTokenHash, &user.RefreshTokenExpiryTime,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

// FindUserByUsername fetches a user by username and maps to domain.User
func (r *PgxUserRepository) FindUserByUsername(ctx context.Context, username string) (*domain.User, error) {
	query := `SELECT user_id, username, password_hash, name, created_at, created_by, last_updated_at, last_updated_by, deleted_at, refresh_token_hash, refresh_token_expiry_time FROM users WHERE username = $1 AND deleted_at IS NULL LIMIT 1`
	row := r.Pool.QueryRow(ctx, query, username)
	var user models.User
	err := row.Scan(
		&user.UserID, &user.Username, &user.PasswordHash, &user.Name, 
		&user.CreatedAt, &user.CreatedBy, &user.LastUpdatedAt, &user.LastUpdatedBy, 
		&user.DeletedAt, &user.RefreshTokenHash, &user.RefreshTokenExpiryTime,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	domainUser := mapping.ToDomainUser(user)
	return &domainUser, nil
}

func (r *PgxUserRepository) SaveUser(ctx context.Context, user domain.User) error {
	modelUser := mapping.ToModelUser(user)
	query := `
        INSERT INTO users (user_id, username, password_hash, name, created_at, created_by, last_updated_at, last_updated_by, refresh_token_hash, refresh_token_expiry_time)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
        ON CONFLICT (user_id) DO UPDATE SET
            username = EXCLUDED.username,
            password_hash = EXCLUDED.password_hash,
            name = EXCLUDED.name,
            last_updated_at = EXCLUDED.last_updated_at,
            last_updated_by = EXCLUDED.last_updated_by,
            refresh_token_hash = EXCLUDED.refresh_token_hash,
            refresh_token_expiry_time = EXCLUDED.refresh_token_expiry_time;
    `
	_, err := r.Pool.Exec(ctx, query,
		modelUser.UserID,
		modelUser.Username,
		modelUser.PasswordHash,
		modelUser.Name,
		modelUser.CreatedAt,
		modelUser.CreatedBy,
		modelUser.LastUpdatedAt,
		modelUser.LastUpdatedBy,
		modelUser.RefreshTokenHash,
		modelUser.RefreshTokenExpiryTime,
	)
	if err != nil {
		return fmt.Errorf("failed to save user: %w", err)
	}
	return nil
}

func (r *PgxUserRepository) FindUserByID(ctx context.Context, userID string) (*domain.User, error) {
	query := `
		SELECT user_id, username, password_hash, name, created_at, created_by, last_updated_at, last_updated_by, deleted_at, refresh_token_hash, refresh_token_expiry_time
		FROM users
		WHERE user_id = $1 AND deleted_at IS NULL;
	`
	var modelUser models.User
	err := r.Pool.QueryRow(ctx, query, userID).Scan(
		&modelUser.UserID,
		&modelUser.Username, // Added username
		&modelUser.PasswordHash, // Added password hash
		&modelUser.Name,
		&modelUser.CreatedAt,
		&modelUser.CreatedBy,
		&modelUser.LastUpdatedAt,
		&modelUser.LastUpdatedBy,
		&modelUser.DeletedAt, 
		&modelUser.RefreshTokenHash, 
		&modelUser.RefreshTokenExpiryTime,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrNotFound
		}
		return nil, fmt.Errorf("failed to find user by ID %s: %w", userID, err)
	}

	domainUser := mapping.ToDomainUser(modelUser)
	return &domainUser, nil
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

	query := `
        SELECT user_id, name, created_at, created_by, last_updated_at, last_updated_by, deleted_at
        FROM users
        WHERE deleted_at IS NULL
        ORDER BY created_at DESC -- Or name, or user_id
        LIMIT $1 OFFSET $2;
    `
	rows, err := r.Pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	modelUsers, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.User])
	if err != nil {
		return nil, fmt.Errorf("failed to collect user rows: %w", err)
	}

	return mapping.ToDomainUserSlice(modelUsers), nil
}

func (r *PgxUserRepository) UpdateUser(ctx context.Context, user domain.User) error {
	modelUser := mapping.ToModelUser(user)
	query := `
        UPDATE users
        SET name = $1, last_updated_at = $2, last_updated_by = $3
        WHERE user_id = $4 AND deleted_at IS NULL;
    `
	cmdTag, err := r.Pool.Exec(ctx, query,
		modelUser.Name,
		modelUser.LastUpdatedAt,
		modelUser.LastUpdatedBy,
		modelUser.UserID,
	)
	if err != nil {
		return fmt.Errorf("failed to execute update user query: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("user not found or already deleted: %w", apperrors.ErrNotFound) // Use app error
	}
	return nil
}

func (r *PgxUserRepository) MarkUserDeleted(ctx context.Context, userID string, deletedAt time.Time, deletedBy string) error {
	query := `
        UPDATE users
        SET deleted_at = $1, last_updated_at = $1, last_updated_by = $2
        WHERE user_id = $3 AND deleted_at IS NULL;
    `
	cmdTag, err := r.Pool.Exec(ctx, query, deletedAt, deletedBy, userID)
	if err != nil {
		return fmt.Errorf("failed to mark user as deleted: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		// User might not exist or was already deleted
		return fmt.Errorf("user not found or already deleted: %w", apperrors.ErrNotFound)
	}
	return nil
}

func (r *PgxUserRepository) UpdateRefreshToken(ctx context.Context, userID string, refreshTokenHash string, refreshTokenExpiryTime time.Time) error {
	now := time.Now()
	query := `
		UPDATE users 
		SET refresh_token_hash = $1, refresh_token_expiry_time = $2, last_updated_at = $3, last_updated_by = $4 
		WHERE user_id = $5 AND deleted_at IS NULL;
	`
	cmdTag, err := r.Pool.Exec(ctx, query, refreshTokenHash, refreshTokenExpiryTime, now, userID, userID)
	if err != nil {
		return fmt.Errorf("failed to update refresh token for user %s: %w", userID, err)
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("user not found or already deleted when updating refresh token for user %s: %w", userID, apperrors.ErrNotFound)
	}
	return nil
}

func (r *PgxUserRepository) ClearRefreshToken(ctx context.Context, userID string) error {
	query := `
		UPDATE users
		SET refresh_token_hash = NULL, refresh_token_expiry_time = NULL
		WHERE user_id = $1;
	`
	cmdTag, err := r.Pool.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to clear refresh token for user %s: %w", userID, err)
	}
	if cmdTag.RowsAffected() == 0 {
		// This could mean the user was not found, or no update was needed.
		// Depending on strictness, this could be an apperrors.ErrNotFound or just logged.
		// For now, we'll consider it not an error if the user simply didn't exist or had no token.
		// If an active user is expected, this might warrant an error.
		// log.Printf("ClearRefreshToken: no rows affected for user %s", userID) // Optional logging
	}
	return nil
}

func (r *PgxUserRepository) UserExists(ctx context.Context, userID string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1
			FROM users
			WHERE user_id = $1 AND deleted_at IS NULL
		);
	`
	var exists bool
	err := r.Pool.QueryRow(ctx, query, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if user exists: %w", err)
	}
	return exists, nil
}
