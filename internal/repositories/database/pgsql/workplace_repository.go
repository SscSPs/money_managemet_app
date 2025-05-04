package pgsql

import (
	"context"
	"database/sql"
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
	BaseRepository
}

// newPgxWorkplaceRepository creates a new repository for workplace data.
func newPgxWorkplaceRepository(pool *pgxpool.Pool) portsrepo.WorkplaceRepositoryWithTx {
	return &PgxWorkplaceRepository{
		BaseRepository: BaseRepository{Pool: pool},
	}
}

// Ensure PgxWorkplaceRepository implements portsrepo.WorkplaceRepositoryWithTx
var _ portsrepo.WorkplaceRepositoryWithTx = (*PgxWorkplaceRepository)(nil)

func (r *PgxWorkplaceRepository) SaveWorkplace(ctx context.Context, workplace domain.Workplace) error {
	query := `
		INSERT INTO workplaces (
			workplace_id, name, description, default_currency_code, is_active,
			created_at, created_by, last_updated_at, last_updated_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);
	`
	_, err := r.Pool.Exec(ctx, query,
		workplace.WorkplaceID,
		workplace.Name,
		workplace.Description,
		workplace.DefaultCurrencyCode,
		workplace.IsActive,
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
			// Handle foreign key violation for currency
			if pgErr.Code == "23503" && pgErr.ConstraintName == "fk_workplace_default_currency" { // foreign_key_violation
				return fmt.Errorf("%w: currency code does not exist", apperrors.ErrValidation)
			}
		}
		return fmt.Errorf("failed to save workplace %s: %w", workplace.WorkplaceID, err)
	}
	return nil
}

func (r *PgxWorkplaceRepository) FindWorkplaceByID(ctx context.Context, workplaceID string) (*domain.Workplace, error) {
	query := `
		SELECT workplace_id, name, description, default_currency_code, is_active,
		       created_at, created_by, last_updated_at, last_updated_by
		FROM workplaces
		WHERE workplace_id = $1;
	`
	var w domain.Workplace
	var defaultCurrencyCode sql.NullString

	err := r.Pool.QueryRow(ctx, query, workplaceID).Scan(
		&w.WorkplaceID,
		&w.Name,
		&w.Description,
		&defaultCurrencyCode,
		&w.IsActive,
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

	if defaultCurrencyCode.Valid {
		w.DefaultCurrencyCode = &defaultCurrencyCode.String
	}

	return &w, nil
}

func (r *PgxWorkplaceRepository) AddUserToWorkplace(ctx context.Context, membership domain.UserWorkplace) error {
	query := `
		INSERT INTO user_workplaces (user_id, workplace_id, role, joined_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, workplace_id) DO UPDATE SET role = EXCLUDED.role;
	` // Upsert: Add user or update their role if they already exist
	_, err := r.Pool.Exec(ctx, query,
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
	err := r.Pool.QueryRow(ctx, query, userID, workplaceID).Scan(
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

func (r *PgxWorkplaceRepository) ListWorkplacesByUserID(ctx context.Context, userID string, includeDisabled bool, role *domain.UserWorkplaceRole) ([]domain.Workplace, error) {
	// Base query component
	baseQuery := `
		SELECT w.workplace_id, w.name, w.description, w.default_currency_code, w.is_active,
			   w.created_at, w.created_by, w.last_updated_at, w.last_updated_by
		FROM workplaces w
		JOIN user_workplaces uw ON w.workplace_id = uw.workplace_id
		WHERE uw.user_id = $1
	`

	// Logic for workplace status and role filtering:
	// - For active workplaces: include all that the user is a member of (any role)
	// - For inactive workplaces: only include those where the user is an admin
	var whereClause string
	var args []interface{}
	args = append(args, userID)

	if !includeDisabled {
		// Simple case: Only include active workplaces
		whereClause = " AND w.is_active = true"

		// If a specific role is requested, add that filter
		if role != nil {
			whereClause += " AND uw.role = $2"
			args = append(args, *role)
		}
	} else {
		// Complex case: All active workplaces + inactive workplaces where user is admin
		whereClause = " AND (w.is_active = true OR (w.is_active = false AND uw.role = $2))"
		args = append(args, domain.RoleAdmin)

		// If a specific role is requested, add that as an additional condition for active workplaces
		if role != nil {
			whereClause = " AND (w.is_active = true AND uw.role = $2 OR (w.is_active = false AND uw.role = $3))"
			args = append(args, *role, domain.RoleAdmin)
		}
	}

	// Complete the query
	query := baseQuery + whereClause + " ORDER BY w.name;"

	// Execute the query
	rows, err := r.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query workplaces for user %s: %w", userID, err)
	}
	defer rows.Close()

	var workplaces []domain.Workplace
	for rows.Next() {
		var w domain.Workplace
		var defaultCurrencyCode sql.NullString
		err := rows.Scan(
			&w.WorkplaceID,
			&w.Name,
			&w.Description,
			&defaultCurrencyCode,
			&w.IsActive,
			&w.CreatedAt,
			&w.CreatedBy,
			&w.LastUpdatedAt,
			&w.LastUpdatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan workplace row: %w", err)
		}
		if defaultCurrencyCode.Valid {
			w.DefaultCurrencyCode = &defaultCurrencyCode.String
		}
		workplaces = append(workplaces, w)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating workplace rows: %w", err)
	}

	return workplaces, nil
}

// UpdateWorkplaceStatus updates the is_active status of a workplace
func (r *PgxWorkplaceRepository) UpdateWorkplaceStatus(ctx context.Context, workplaceID string, isActive bool, updatedByUserID string) error {
	query := `
		UPDATE workplaces
		SET is_active = $1, last_updated_at = NOW(), last_updated_by = $2
		WHERE workplace_id = $3;
	`
	result, err := r.Pool.Exec(ctx, query, isActive, updatedByUserID, workplaceID)
	if err != nil {
		return fmt.Errorf("failed to update workplace status: %w", err)
	}

	// Check if any rows were affected
	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return apperrors.ErrNotFound
	}

	return nil
}

// ListUsersByWorkplaceID retrieves all users that belong to a specific workplace
// By default, it excludes users with the REMOVED role.
// Set includeRemoved to true to include users with the REMOVED role.
func (r *PgxWorkplaceRepository) ListUsersByWorkplaceID(ctx context.Context, workplaceID string, includeRemoved ...bool) ([]domain.UserWorkplace, error) {
	query := `
		SELECT uw.user_id, u.name as user_name, uw.workplace_id, uw.role, uw.joined_at
		FROM user_workplaces uw
		JOIN users u ON uw.user_id = u.user_id
		WHERE uw.workplace_id = $1
	`

	// By default, exclude REMOVED users
	shouldIncludeRemoved := false
	if len(includeRemoved) > 0 {
		shouldIncludeRemoved = includeRemoved[0]
	}

	if !shouldIncludeRemoved {
		query += ` AND uw.role != $2`
	}

	query += ` ORDER BY uw.joined_at DESC;`

	var rows pgx.Rows
	var err error

	if !shouldIncludeRemoved {
		rows, err = r.Pool.Query(ctx, query, workplaceID, domain.RoleRemoved)
	} else {
		rows, err = r.Pool.Query(ctx, query, workplaceID)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to query users for workplace %s: %w", workplaceID, err)
	}
	defer rows.Close()

	var userWorkplaces []domain.UserWorkplace
	for rows.Next() {
		var uw domain.UserWorkplace
		err := rows.Scan(
			&uw.UserID,
			&uw.UserName,
			&uw.WorkplaceID,
			&uw.Role,
			&uw.JoinedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user workplace row: %w", err)
		}
		userWorkplaces = append(userWorkplaces, uw)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating user workplace rows: %w", err)
	}

	return userWorkplaces, nil
}

// RemoveUserFromWorkplace marks a user as removed in a workplace by setting their role to REMOVED
func (r *PgxWorkplaceRepository) RemoveUserFromWorkplace(ctx context.Context, userID, workplaceID string) error {
	// Reuse the UpdateUserWorkplaceRole method with the REMOVED role
	return r.UpdateUserWorkplaceRole(ctx, userID, workplaceID, domain.RoleRemoved)
}

// UpdateUserWorkplaceRole updates a user's role in a workplace
func (r *PgxWorkplaceRepository) UpdateUserWorkplaceRole(ctx context.Context, userID, workplaceID string, newRole domain.UserWorkplaceRole) error {
	query := `
		UPDATE user_workplaces
		SET role = $3
		WHERE user_id = $1 AND workplace_id = $2;
	`

	result, err := r.Pool.Exec(ctx, query, userID, workplaceID, newRole)
	if err != nil {
		return fmt.Errorf("failed to update role for user %s in workplace %s: %w", userID, workplaceID, err)
	}

	// Check if any rows were affected
	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return apperrors.ErrNotFound
	}

	return nil
}
