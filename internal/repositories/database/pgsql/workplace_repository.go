package pgsql

import (
	"context"
	"errors"
	"fmt"

	"github.com/SscSPs/money_managemet_app/internal/apperrors"
	"github.com/SscSPs/money_managemet_app/internal/core/domain"
	portsrepo "github.com/SscSPs/money_managemet_app/internal/core/ports/repositories"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PgxWorkplaceRepository struct {
	pool *pgxpool.Pool
}

// NewPgxWorkplaceRepository creates a new repository for workplace data.
func NewPgxWorkplaceRepository(pool *pgxpool.Pool) portsrepo.WorkplaceRepository {
	return &PgxWorkplaceRepository{pool: pool}
}

// Ensure PgxWorkplaceRepository implements portsrepo.WorkplaceRepository
var _ portsrepo.WorkplaceRepository = (*PgxWorkplaceRepository)(nil)

func (r *PgxWorkplaceRepository) SaveWorkplace(ctx context.Context, workplace domain.Workplace) error {
	query := `
		INSERT INTO workplaces (workplace_id, name, description, created_at, created_by, last_updated_at, last_updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7);
	`
	_, err := r.pool.Exec(ctx, query,
		workplace.WorkplaceID,
		workplace.Name,
		workplace.Description,
		workplace.CreatedAt,
		workplace.CreatedBy,
		workplace.LastUpdatedAt,
		workplace.LastUpdatedBy,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" { // unique_violation
				return fmt.Errorf("%w: workplace ID %s already exists", apperrors.ErrDuplicate, workplace.WorkplaceID)
			}
		}
		return fmt.Errorf("failed to save workplace %s: %w", workplace.WorkplaceID, err)
	}
	return nil
}

func (r *PgxWorkplaceRepository) FindWorkplaceByID(ctx context.Context, workplaceID string) (*domain.Workplace, error) {
	query := `
		SELECT workplace_id, name, description, created_at, created_by, last_updated_at, last_updated_by
		FROM workplaces
		WHERE workplace_id = $1;
	`
	var w domain.Workplace
	err := r.pool.QueryRow(ctx, query, workplaceID).Scan(
		&w.WorkplaceID,
		&w.Name,
		&w.Description,
		&w.CreatedAt,
		&w.CreatedBy,
		&w.LastUpdatedAt,
		&w.LastUpdatedBy,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrNotFound
		}
		return nil, fmt.Errorf("failed to find workplace by ID %s: %w", workplaceID, err)
	}
	return &w, nil
}

func (r *PgxWorkplaceRepository) AddUserToWorkplace(ctx context.Context, membership domain.UserWorkplace) error {
	query := `
		INSERT INTO user_workplaces (user_id, workplace_id, role, joined_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, workplace_id) DO UPDATE SET role = EXCLUDED.role;
	` // Upsert: Add user or update their role if they already exist
	_, err := r.pool.Exec(ctx, query,
		membership.UserID,
		membership.WorkplaceID,
		membership.Role,
		membership.JoinedAt,
	)

	if err != nil {
		// Check for foreign key violation if needed (e.g., user_id or workplace_id doesn't exist)
		return fmt.Errorf("failed to add/update user %s in workplace %s: %w", membership.UserID, membership.WorkplaceID, err)
	}
	return nil
}

func (r *PgxWorkplaceRepository) FindUserWorkplaceRole(ctx context.Context, userID, workplaceID string) (*domain.UserWorkplace, error) {
	query := `
		SELECT user_id, workplace_id, role, joined_at
		FROM user_workplaces
		WHERE user_id = $1 AND workplace_id = $2;
	`
	var uw domain.UserWorkplace
	err := r.pool.QueryRow(ctx, query, userID, workplaceID).Scan(
		&uw.UserID,
		&uw.WorkplaceID,
		&uw.Role,
		&uw.JoinedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Consider if ErrNotFound is appropriate or if absence means 'no access'
			return nil, apperrors.ErrNotFound // User not found within this specific workplace
		}
		return nil, fmt.Errorf("failed to find user %s workplace role in %s: %w", userID, workplaceID, err)
	}
	return &uw, nil
}

func (r *PgxWorkplaceRepository) ListWorkplacesByUserID(ctx context.Context, userID string) ([]domain.Workplace, error) {
	query := `
		SELECT w.workplace_id, w.name, w.description, w.created_at, w.created_by, w.last_updated_at, w.last_updated_by
		FROM workplaces w
		JOIN user_workplaces uw ON w.workplace_id = uw.workplace_id
		WHERE uw.user_id = $1
		ORDER BY w.name;
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query workplaces for user %s: %w", userID, err)
	}
	defer rows.Close()

	workplaces := []domain.Workplace{}
	for rows.Next() {
		var w domain.Workplace
		err := rows.Scan(
			&w.WorkplaceID,
			&w.Name,
			&w.Description,
			&w.CreatedAt,
			&w.CreatedBy,
			&w.LastUpdatedAt,
			&w.LastUpdatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan workplace row for user %s: %w", userID, err)
		}
		workplaces = append(workplaces, w)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating workplace rows for user %s: %w", userID, err)
	}

	return workplaces, nil
}
