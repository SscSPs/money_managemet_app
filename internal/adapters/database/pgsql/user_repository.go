package pgsql

import (
	"context"
	"fmt" // For error wrapping

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
