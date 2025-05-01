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
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PgxUserRepository struct {
	db *pgxpool.Pool
}

func newPgxUserRepository(db *pgxpool.Pool) portsrepo.UserRepository {
	return &PgxUserRepository{db: db}
}

// Ensure PgxUserRepository implements portsrepo.UserRepository
var _ portsrepo.UserRepository = (*PgxUserRepository)(nil)

// Helper to convert domain.User to models.User
func toModelUser(d domain.User) models.User {
	return models.User{
		UserID: d.UserID,
		Name:   d.Name,
		AuditFields: models.AuditFields{
			CreatedAt:     d.CreatedAt,
			CreatedBy:     d.CreatedBy,
			LastUpdatedAt: d.LastUpdatedAt,
			LastUpdatedBy: d.LastUpdatedBy,
		},
		DeletedAt: d.DeletedAt,
	}
}

// Helper to convert models.User to domain.User
func toDomainUser(m models.User) domain.User {
	return domain.User{
		UserID: m.UserID,
		Name:   m.Name,
		AuditFields: domain.AuditFields{
			CreatedAt:     m.CreatedAt,
			CreatedBy:     m.CreatedBy,
			LastUpdatedAt: m.LastUpdatedAt,
			LastUpdatedBy: m.LastUpdatedBy,
		},
		DeletedAt: m.DeletedAt,
	}
}

// Helper to convert slice of models.User to slice of domain.User
func toDomainUserSlice(ms []models.User) []domain.User {
	ds := make([]domain.User, len(ms))
	for i, m := range ms {
		ds[i] = toDomainUser(m)
	}
	return ds
}

func (r *PgxUserRepository) SaveUser(ctx context.Context, user domain.User) error {
	modelUser := toModelUser(user)
	query := `
        INSERT INTO users (user_id, name, created_at, created_by, last_updated_at, last_updated_by)
        VALUES ($1, $2, $3, $4, $5, $6)
        ON CONFLICT (user_id) DO UPDATE SET
            name = EXCLUDED.name,
            last_updated_at = EXCLUDED.last_updated_at,
            last_updated_by = EXCLUDED.last_updated_by;
    `
	_, err := r.db.Exec(ctx, query,
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
	err := r.db.QueryRow(ctx, query, userID).Scan(
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

	domainUser := toDomainUser(modelUser)
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
	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	modelUsers := []models.User{}
	for rows.Next() {
		var modelUser models.User
		err := rows.Scan(
			&modelUser.UserID,
			&modelUser.Name,
			&modelUser.CreatedAt,
			&modelUser.CreatedBy,
			&modelUser.LastUpdatedAt,
			&modelUser.LastUpdatedBy,
			&modelUser.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user row: %w", err)
		}
		modelUsers = append(modelUsers, modelUser)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("error iterating user rows: %w", rows.Err())
	}

	return toDomainUserSlice(modelUsers), nil
}

func (r *PgxUserRepository) UpdateUser(ctx context.Context, user domain.User) error {
	modelUser := toModelUser(user)
	query := `
        UPDATE users
        SET name = $1, last_updated_at = $2, last_updated_by = $3
        WHERE user_id = $4 AND deleted_at IS NULL;
    `
	cmdTag, err := r.db.Exec(ctx, query,
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
