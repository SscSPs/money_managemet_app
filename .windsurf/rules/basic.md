---
trigger: always_on
---

# MMA Backend Development Ruleset
 
> Last Updated: 2025-11-02  
> Architecture: Clean Architecture with Hexagonal/Ports & Adapters Pattern

## 📋 Quick Reference

- **Language:** Go 1.24.2
- **Framework:** Gin
- **Database:** PostgreSQL (pgx/v5)
- **Auth:** JWT
- **Docs:** Swagger

---

## 🏗️ Architecture Rules

### Layer Dependencies

```
Handlers → Services → Repositories → Database
    ↓         ↓            ↓
  Ports    Ports       Models
```

### Critical Rules

1. **Handlers MUST:**
   - Depend ONLY on service interfaces from `internal/core/ports/services`
   - Never import repository or database packages
   - Use DTOs from [internal/dto](cci:7://file:///Users/sahilsoni/me/projects/money_managemet_app/internal/dto:0:0-0:0)
   - Extract context: `c.Request.Context()`
   - Get user ID: `middleware.GetUserIDFromContext(c)`

2. **Services MUST:**
   - Depend ONLY on repository interfaces from `internal/core/ports/repositories`
   - Return domain objects from [internal/core/domain](cci:7://file:///Users/sahilsoni/me/projects/money_managemet_app/internal/Users/sahilsoni/me/projects/money_managemet_app/internal/core/domain:0:0-0:0)
   - Accept `context.Context` as first parameter
   - Never import handler or database packages

3. **Repositories MUST:**
   - Live under `internal/repositories/database/pgsql`
   - Work with models from [internal/models](cci:7://file:///Users/sahilsoni/me/projects/money_managemet_app/internal/models:0:0-0:0)
   - Map to domain using [internal/utils/mapping](cci:7://file:///Users/sahilsoni/me/projects/money_managemet_app/internal/Users/sahilsoni/me/projects/money_managemet_app/internal/utils/mapping:0:0-0:0)
   - Return `apperrors.ErrNotFound` when no results

### Interface Pattern

**Split by capability:**

```go
type UserReaderSvc interface { /* read methods */ }
type UserWriterSvc interface { /* write methods */ }
type UserLifecycleSvc interface { /* lifecycle */ }

// Facade combines all
type UserSvcFacade interface {
    UserReaderSvc
    UserWriterSvc
    UserLifecycleSvc
}
```

---

## 📁 Directory Structure

```
/cmd/mma_backend/          # Entry point
/internal/
  /apperrors/              # Custom errors
  /core/
    /domain/               # Domain models
    /ports/
      /services/           # Service interfaces
      /repositories/       # Repository interfaces  
    /services/             # Service implementations
  /dto/                    # Request/Response DTOs
  /handlers/               # HTTP handlers
  /middleware/             # Middleware
  /models/                 # Database models
  /platform/               # Config, database
  /repositories/database/pgsql/  # Repo implementations
  /utils/                  # Utilities
/migrations/               # SQL migrations
```

---

## 🏷️ Naming Conventions

| Type | Example | Location |
|------|---------|----------|
| Domain | [User](cci:2://file:///Users/sahilsoni/me/projects/money_managemet_app/internal/models/user.go:10:0-24:1), `Account` | [core/domain/](cci:7://file:///Users/sahilsoni/me/projects/money_managemet_app/internal/Users/sahilsoni/me/projects/money_managemet_app/internal/core/domain:0:0-0:0) |
| Service | [userService](cci:2://file:///Users/sahilsoni/me/projects/money_managemet_app/internal/core/services/user_service.go:22:0-24:1) | [core/services/](cci:7://file:///Users/sahilsoni/me/projects/money_managemet_app/internal/Users/sahilsoni/me/projects/money_managemet_app/internal/core/services:0:0-0:0) |
| Repo | [PgxUserRepository](cci:2://file:///Users/sahilsoni/me/projects/money_managemet_app/internal/repositories/database/pgsql/user_repository.go:16:0-18:1) | `repositories/database/pgsql/` |
| Handler | [userHandler](cci:2://file:///Users/sahilsoni/me/projects/money_managemet_app/internal/handlers/handler_user.go:16:0-18:1) | [handlers/](cci:7://file:///Users/sahilsoni/me/projects/money_managemet_app/internal/handlers:0:0-0:0) |
| DTO | `CreateUserRequest` | [dto/](cci:7://file:///Users/sahilsoni/me/projects/money_managemet_app/internal/dto:0:0-0:0) |
| Interface (Svc) | [UserSvcFacade](cci:2://file:///Users/sahilsoni/me/projects/money_managemet_app/internal/core/ports/services/user_services.go:63:0-68:1) | `core/ports/services/` |
| Interface (Repo) | [UserRepositoryFacade](cci:2://file:///Users/sahilsoni/me/projects/money_managemet_app/internal/core/ports/repositories/user_repositories.go:50:0-54:1) | `core/ports/repositories/` |

**Import Aliases:**
```go
import (
    portssvc "github.com/SscSPs/money_managemet_app/internal/core/ports/services"
    portsrepo "github.com/SscSPs/money_managemet_app/internal/core/ports/repositories"
)
```

---

## ✅ Code Quality Checklist

**Before Committing:**
- [ ] Run `make test`
- [ ] Run `make lint` (if available)
- [ ] Update Swagger: `make swagger`
- [ ] Check no circular dependencies
- [ ] All handlers have Swagger annotations
- [ ] All methods use `context.Context`
- [ ] Errors wrapped with [apperrors](cci:7://file:///Users/sahilsoni/me/projects/money_managemet_app/internal/apperrors:0:0-0:0)
- [ ] Structured logging used
- [ ] DTOs have validation tags

**Common Validation Tags:**
```go
type CreateUserRequest struct {
    Username string          `json:"username" binding:"required,min=3,max=50"`
    Email    string          `json:"email" binding:"required,email"`
    Age      int             `json:"age" binding:"required,gte=0,lte=130"`
    Amount   decimal.Decimal `json:"amount" binding:"required,decimal_gtz"` // Custom validator
}
```

---

## 🔧 Makefile Commands

```bash
make build          # Build binary
make run            # Run with tests
make run-fast       # Run without tests  
make test           # Unit tests
make test-all       # All tests
make swagger        # Generate Swagger docs
make lint           # Run linter
make clean          # Clean artifacts
make deps           # Install dependencies
```

---

## 📦 Dependency Management

- **Adding a new dependency:** `go get github.com/new/dependency`
- **Cleaning up dependencies:** `go mod tidy`
- **Updating dependencies:** `go get -u ./...`

---

## 💅 Code Formatting

- **Format all Go files:** `gofmt -w .`


---

## 📌 Common Patterns

### DTO Conversion
```go
// Domain → DTO
func ToUserResponse(user *domain.User) UserResponse {
    return UserResponse{
        UserID:   user.UserID,
        Username: user.Username,
        Name:     user.Name,
    }
}

// Slice conversion
func ToUserResponseList(users []domain.User) []UserResponse {
    responses := make([]UserResponse, len(users))
    for i, user := range users {
        responses[i] = ToUserResponse(&user)
    }
    return responses
}
```

### Pagination
```go
type ListParams struct {
    Limit  int `form:"limit,default=20"`
    Offset int `form:"offset,default=0"`
}

type ListResponse struct {
    Items []UserResponse `json:"items"`
    Total int            `json:"total,omitempty"`
}
```

### Soft Delete
```go
func (r *Repo) MarkUserDeleted(ctx context.Context, user *domain.User, deletedBy string) error {
    now := time.Now()
    query := `
        UPDATE users 
        SET deleted_at = $1, last_updated_by = $2, last_updated_at = $3 
        WHERE user_id = $4 AND deleted_at IS NULL
    `
    _, err := r.Pool.Exec(ctx, query, now, deletedBy, now, user.UserID)
    return err
}
```

### Audit Fields Pattern
```go
// On create
user := &domain.User{
    UserID: uuid.NewString(),
    AuditFields: domain.AuditFields{
        CreatedAt:     time.Now(),
        LastUpdatedAt: time.Now(),
        CreatedBy:     creatorUserID,
        LastUpdatedBy: creatorUserID,
    },
}

// On update
user.LastUpdatedAt = time.Now()
user.LastUpdatedBy = updaterUserID
```

### Authorization Pattern (from BaseService)
```go
// In service
type BaseService struct {
    WorkplaceAuthorizer portssvc.WorkplaceAuthorizerSvc
}

func (s *accountService) CreateAccount(ctx context.Context, workplaceID string, req dto.CreateAccountRequest, userID string) (*domain.Account, error) {
    // Authorize first
    if err := s.AuthorizeUser(ctx, userID, workplaceID, domain.RoleMember); err != nil {
        return nil, err
    }
    // ... proceed with creation
}
```

---

## 🎯 Quick Tips

1. **Always pass context** - Every service/repo method starts with `context.Context`
2. **Use interfaces** - Handlers → Service interfaces, Services → Repo interfaces
3. **Map at boundaries** - DB Models ↔ Domain at repo layer, Domain ↔ DTO at handler layer
4. **Validate early** - In DTOs with `binding` tags
5. **Log structured** - Use `slog` with key-value pairs
6. **Handle errors properly** - Use `errors.Is()` and wrap with [apperrors](cci:7://file:///Users/sahilsoni/me/projects/money_managemet_app/internal/apperrors:0:0-0:0)
7. **Document APIs** - Add Swagger annotations to all handler methods
8. **Test isolation** - Use mocks/interfaces for dependencies
9. **Soft delete** - Always add `deleted_at IS NULL` to queries
10. **Consistent naming** - Follow established patterns (Find*, Save*, Create*, Get*)

---

## 🚫 Common Mistakes to Avoid

1. ❌ Importing concrete implementations in handlers (use interfaces)
2. ❌ Forgetting to pass `c.Request.Context()` to services
3. ❌ Returning database models from repositories (use domain objects)
4. ❌ Not checking `errors.Is()` for sentinel errors
5. ❌ Hardcoding user IDs (get from context)
6. ❌ Missing Swagger annotations
7. ❌ Not setting audit fields on create/update
8. ❌ Circular dependencies between packages
9. ❌ Exposing sensitive fields in JSON (use `json:"-"`)
10. ❌ Not filtering soft-deleted records