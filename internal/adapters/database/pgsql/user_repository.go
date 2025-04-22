package pgsql

import (
	"context"
	"fmt"  // For error wrapping
	"time" // Added for MarkUserDeleted

	"github.com/SscSPs/money_managemet_app/internal/core/ports"
	"github.com/SscSPs/money_managemet_app/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

// Ensure UserRepository implements ports.UserRepository
var _ ports.UserRepository = (*UserRepository)(nil)

func (r *UserRepository) SaveUser(ctx context.Context, user models.User) error {
	query := `
        INSERT INTO users (user_id, name, created_at, created_by, last_updated_at, last_updated_by)
        VALUES ($1, $2, $3, $4, $5, $6)
        ON CONFLICT (user_id) DO UPDATE SET
            name = EXCLUDED.name,
            last_updated_at = EXCLUDED.last_updated_at,
            last_updated_by = EXCLUDED.last_updated_by;
    `
	_, err := r.db.Exec(ctx, query,
		user.UserID,
		user.Name,
		user.CreatedAt,
		user.CreatedBy,
		user.LastUpdatedAt,
		user.LastUpdatedBy,
	)
	if err != nil {
		return fmt.Errorf("failed to save user: %w", err)
	}
	return nil
}

func (r *UserRepository) FindUserByID(ctx context.Context, userID string) (*models.User, error) {
	query := `
        SELECT user_id, name, created_at, created_by, last_updated_at, last_updated_by
        FROM users
        WHERE user_id = $1;
    `
	var user models.User
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&user.UserID,
		&user.Name,
		&user.CreatedAt,
		&user.CreatedBy,
		&user.LastUpdatedAt,
		&user.LastUpdatedBy,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // Indicate not found explicitly
		}
		return nil, fmt.Errorf("failed to find user by ID: %w", err)
	}
	return &user, nil
}

func (r *UserRepository) FindUsers(ctx context.Context, limit int, offset int) ([]models.User, error) {
	// Default limit if not specified or invalid
	if limit <= 0 {
		limit = 20
	}
	// Ensure offset is non-negative
	if offset < 0 {
		offset = 0
	}

	query := `
        SELECT user_id, name, created_at, created_by, last_updated_at, last_updated_by
        FROM users
        WHERE deleted_at IS NULL
        ORDER BY created_at DESC -- Or name, or user_id
        LIMIT $1 OFFSET $2;
    `
	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	users := []models.User{}
	for rows.Next() {
		var user models.User
		err := rows.Scan(
			&user.UserID,
			&user.Name,
			&user.CreatedAt,
			&user.CreatedBy,
			&user.LastUpdatedAt,
			&user.LastUpdatedBy,
			// Note: We are not selecting deleted_at here as we filter by IS NULL
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user row: %w", err)
		}
		users = append(users, user)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("error iterating user rows: %w", rows.Err())
	}

	return users, nil
}

func (r *UserRepository) UpdateUser(ctx context.Context, user models.User) error {
	query := `
        UPDATE users
        SET name = $1, last_updated_at = $2, last_updated_by = $3
        WHERE user_id = $4 AND deleted_at IS NULL;
    `
	cmdTag, err := r.db.Exec(ctx, query,
		user.Name,
		user.LastUpdatedAt, // Should be set by the service layer before calling
		user.LastUpdatedBy, // Should be set by the service layer before calling
		user.UserID,
	)
	if err != nil {
		return fmt.Errorf("failed to execute update user query: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		// This could mean the user doesn't exist or is already deleted.
		// The service layer might want to check FindUserByID first if this distinction matters.
		return fmt.Errorf("user not found or already deleted: %w", pgx.ErrNoRows) // Return an error compatible with ErrNotFound checks
	}
	return nil
}

func (r *UserRepository) MarkUserDeleted(ctx context.Context, userID string, deletedAt time.Time, deletedBy string) error {
	query := `
        UPDATE users
        SET deleted_at = $1, last_updated_at = $1, last_updated_by = $2
        WHERE user_id = $3 AND deleted_at IS NULL;
    `
	cmdTag, err := r.db.Exec(ctx, query, deletedAt, deletedBy, userID)
	if err != nil {
		return fmt.Errorf("failed to mark user as deleted: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		// User might not exist or was already deleted
		return fmt.Errorf("user not found or already deleted: %w", pgx.ErrNoRows)
	}
	return nil
}
