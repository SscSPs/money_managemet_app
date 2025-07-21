package pgsql

import (
	"context"
	"database/sql"
	"errors"

	"github.com/SscSPs/money_managemet_app/internal/apperrors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB is an interface that both *pgxpool.Pool and pgx.Tx satisfy
type DB interface {
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	Begin(ctx context.Context) (pgx.Tx, error)
}

// BaseRepository provides common functionality for all repositories
type BaseRepository struct {
	Pool *pgxpool.Pool
	tx   pgx.Tx // current transaction, if any
}

// DB returns the current transaction if available, otherwise the pool
func (r *BaseRepository) DB() DB {
	if r.tx != nil {
		return r.tx
	}
	return r.Pool
}

// SetTx sets the current transaction
func (r *BaseRepository) SetTx(tx pgx.Tx) {
	r.tx = tx
}

// GetTx returns the current transaction, if any
func (r *BaseRepository) GetTx() pgx.Tx {
	return r.tx
}

// Begin starts a new database transaction
func (r *BaseRepository) Begin(ctx context.Context) (pgx.Tx, error) {
	// If already in a transaction, return that
	if r.tx != nil {
		return r.tx, nil
	}

	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return nil, apperrors.NewAppError(500, "failed to begin transaction", err)
	}
	r.tx = tx
	return tx, nil
}

// Commit commits the current transaction if one exists
func (r *BaseRepository) Commit(ctx context.Context) error {
	if r.tx == nil {
		return nil // No transaction to commit
	}

	if err := r.tx.Commit(ctx); err != nil {
		return apperrors.NewAppError(500, "failed to commit transaction", err)
	}
	r.tx = nil // Clear the transaction
	return nil
}

// Rollback rolls back the current transaction if one exists
func (r *BaseRepository) Rollback(ctx context.Context) error {
	if r.tx == nil {
		return nil // No transaction to rollback
	}

	if err := r.tx.Rollback(ctx); err != nil && !errors.Is(err, sql.ErrTxDone) {
		return apperrors.NewAppError(500, "failed to rollback transaction", err)
	}
	r.tx = nil // Clear the transaction
	return nil
}

// WithTx runs the provided function within a transaction
func (r *BaseRepository) WithTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := r.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p) // Re-throw panic after rollback
		}
	}()

	// Execute the function
	if err := fn(tx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}

	return tx.Commit(ctx)
}
