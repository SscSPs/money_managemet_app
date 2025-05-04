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

func (r *PgxUserRepository) SaveUser(ctx context.Context, user domain.User) error {
	modelUser := mapping.ToModelUser(user)
	query := `
        INSERT INTO users (user_id, name, created_at, created_by, last_updated_at, last_updated_by)
        VALUES ($1, $2, $3, $4, $5, $6)
        ON CONFLICT (user_id) DO UPDATE SET
            name = EXCLUDED.name,
            last_updated_at = EXCLUDED.last_updated_at,
            last_updated_by = EXCLUDED.last_updated_by;
    `
	_, err := r.Pool.Exec(ctx, query,
		modelUser.UserID,
		modelUser.Name,
		modelUser.CreatedAt,
		modelUser.CreatedBy,
		modelUser.LastUpdatedAt,
		modelUser.LastUpdatedBy,
	)
	if err != nil {
		return fmt.Errorf("failed to save user: %w", err)
	}
	return nil
}

func (r *PgxUserRepository) FindUserByID(ctx context.Context, userID string) (*domain.User, error) {
	query := `
		SELECT user_id, name, created_at, created_by, last_updated_at, last_updated_by, deleted_at
		FROM users
		WHERE user_id = $1 AND deleted_at IS NULL;
	`
	var modelUser models.User
	err := r.Pool.QueryRow(ctx, query, userID).Scan(
		&modelUser.UserID,
		&modelUser.Name,
		&modelUser.CreatedAt,
		&modelUser.CreatedBy,
		&modelUser.LastUpdatedAt,
		&modelUser.LastUpdatedBy,
		&modelUser.DeletedAt, // Scan DeletedAt
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
