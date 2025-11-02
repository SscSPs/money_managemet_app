---
trigger: model_decision
description: when working on creating/updating/handling repositories/db
---

## 🗄️ Repository Pattern

```go
package pgsql

type PgxUserRepository struct {
    BaseRepository
}

func newPgxUserRepository(db *pgxpool.Pool) portsrepo.UserRepositoryWithTx {
    return &PgxUserRepository{BaseRepository: BaseRepository{Pool: db}}
}

const FULL_USERS_SELECT = `SELECT user_id, username, email ... FROM users`

func (r *PgxUserRepository) FindUserByID(ctx context.Context, userID string) (*domain.User, error) {
    query := FULL_USERS_SELECT + ` WHERE user_id = $1 AND deleted_at IS NULL`
    
    rows, err := r.Pool.Query(ctx, query, userID)
    if err != nil {
        return nil, apperrors.NewAppError(500, "failed to query", err)
    }
    defer rows.Close()
    
    modelUsers, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.User])
    if err != nil || len(modelUsers) == 0 {
        return nil, apperrors.ErrNotFound
    }
    
    domainUsers := mapping.ToDomainUserSlice(modelUsers)
    return &domainUsers[0], nil
}

func (r *PgxUserRepository) SaveUser(ctx context.Context, user *domain.User) error {
    modelUser := mapping.ToModelUser(*user)
    query := `INSERT INTO users (user_id, username, ...) VALUES ($1, $2, ...)`
    _, err := r.Pool.Exec(ctx, query, modelUser.UserID, modelUser.Username, ...)
    if err != nil {
        return apperrors.NewAppError(500, "failed to save", err)
    }
    return nil
}
```

### Bulk Operations with `pgx.Batch`

For bulk inserts, updates, or deletes, use `pgx.Batch` to improve performance by reducing the number of round trips to the database.

```go
func (r *PgxUserRepository) CreateMultipleUsers(ctx context.Context, users []domain.User) error {
    batch := &pgx.Batch{}
    for _, user := range users {
        modelUser := mapping.ToModelUser(user)
        query := `INSERT INTO users (user_id, username, ...) VALUES ($1, $2, ...)`
        batch.Queue(query, modelUser.UserID, modelUser.Username, ...)
    }

    br := r.Pool.SendBatch(ctx, batch)
    defer br.Close()

    // Check for errors
    if _, err := br.Exec(); err != nil {
        return apperrors.NewAppError(500, "failed to create multiple users", err)
    }

    return nil
}
```

**Repository Checklist:**
- ✅ Embed `BaseRepository`
- ✅ Use constants for SELECT queries
- ✅ Use `pgx.CollectRows()` for multiple rows
- ✅ Use `pgx.Batch` for bulk operations.
- ✅ Map with `mapping.ToDomainXxx()`
- ✅ Return `apperrors.ErrNotFound` when empty
- ✅ Filter soft deletes: `WHERE deleted_at IS NULL`
- ✅ Use parameterized queries: `$1, $2, ...`
- ❌ Never return database models directly