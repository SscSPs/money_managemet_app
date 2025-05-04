package repositories

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// TransactionManager defines methods for transaction management
type TransactionManager interface {
	// Begin starts a new database transaction
	Begin(ctx context.Context) (pgx.Tx, error)

	// Commit commits a transaction
	Commit(ctx context.Context, tx pgx.Tx) error

	// Rollback rolls back a transaction
	Rollback(ctx context.Context, tx pgx.Tx) error
}

// RepositoryWithTx is a marker interface for repositories that support transactions
type RepositoryWithTx interface {
	TransactionManager
}
