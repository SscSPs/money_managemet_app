---
trigger: model_decision
description: when working on creating/updating/handling tests or new code that needs tests
---


## 🧪 Testing Patterns

### Unit Testing

```go
package services_test

import (
    "context"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

// Mock Repository

type MockUserRepository struct {
    mock.Mock
}

func (m *MockUserRepository) FindUserByID(ctx context.Context, userID string) (*domain.User, error) {
    args := m.Called(ctx, userID)
    return args.Get(0).(*domain.User), args.Error(1)
}

func TestUserService_GetUserByID(t *testing.T) {
    // Arrange
    mockRepo := new(MockUserRepository)
    service := NewUserService(mockRepo)
    ctx := context.Background()
    
    expectedUser := &domain.User{UserID: "user-123"}
    mockRepo.On("FindUserByID", ctx, "user-123").Return(expectedUser, nil)
    
    // Act
    user, err := service.GetUserByID(ctx, "user-123")
    
    // Assert
    assert.NoError(t, err)
    assert.Equal(t, "user-123", user.UserID)
    mockRepo.AssertExpectations(t)
}
```

### Integration Testing with Testcontainers

For integration tests, use [Testcontainers for Go](https://golang.testcontainers.org/) to spin up a real PostgreSQL database in a Docker container.

```go
package repositories_test

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestUserRepository_FindUserByID(t *testing.T) {
    // Arrange
    ctx := context.Background()
    
    pgContainer, err := postgres.RunContainer(ctx,
        testcontainers.WithImage("postgres:13"),
        postgres.WithDatabase("test-db"),
        postgres.WithUsername("user"),
        postgres.WithPassword("password"),
    )
    if err != nil {
        t.Fatal(err)
    }
    defer pgContainer.Terminate(ctx)

    connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
    if err != nil {
        t.Fatal(err)
    }

    pool, err := pgxpool.Connect(ctx, connStr)
    if err != nil {
        t.Fatal(err)
    }
    defer pool.Close()

    // ... run migrations and seed data ...

    repo := NewPgxUserRepository(pool)

    // Act
    user, err := repo.FindUserByID(ctx, "user-123")

    // Assert
    assert.NoError(t, err)
    assert.Equal(t, "user-123", user.UserID)
}
```

**Test Commands:**
- `make test` - Unit tests only
- `make test-integration` - Integration tests (requires Docker)
- `make test-all` - All tests

**Test File Naming:** `<name>_test.go`

**Testing Rules:**
- ✅ Use `testify/mock` to create mocks for interfaces in unit tests.
- ✅ Use `testify/assert` for assertions.
- ✅ Use Testcontainers for Go for integration tests that require external services like a database.
- ✅ Aim for a high test coverage and use `go test -cover` to measure it.
