package pgsql

import (
	"context"
	"database/sql"
	"errors"

	"github.com/SscSPs/money_managemet_app/internal/apperrors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// BaseRepository provides common functionality for all repositories
type BaseRepository struct {
	Pool *pgxpool.Pool
}

// Begin starts a new database transaction
func (r *BaseRepository) Begin(ctx context.Context) (pgx.Tx, error) {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return nil, apperrors.NewAppError(500, "failed to begin transaction", err)
	}
	return tx, nil
}

// Commit commits a transaction
func (r *BaseRepository) Commit(ctx context.Context, tx pgx.Tx) error {
	if err := tx.Commit(ctx); err != nil {
		return apperrors.NewAppError(500, "failed to commit transaction", err)
	}
	return nil
}

// Rollback rolls back a transaction
func (r *BaseRepository) Rollback(ctx context.Context, tx pgx.Tx) error {
	if err := tx.Rollback(ctx); err != nil && !errors.Is(err, sql.ErrTxDone) {
		return apperrors.NewAppError(500, "failed to rollback transaction", err)
	}
	return nil
}
