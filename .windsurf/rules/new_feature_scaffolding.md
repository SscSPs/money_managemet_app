---
trigger: model_decision
description: when working on creating/updating/handling any new feature or flag
---


## 🚀 New Feature Scaffolding

When adding a new entity (e.g., `Product`):

1. **Domain Model:** `internal/core/domain/product.go`
```go
type Product struct {
    ProductID string
    Name      string
    Price     decimal.Decimal
    AuditFields
    DeletedAt *time.Time
}
```

2. **Repository Interface:** `internal/core/ports/repositories/product_repositories.go`
```go
type ProductReader interface {
    FindProductByID(ctx context.Context, productID string) (*domain.Product, error)
}

type ProductWriter interface {
    SaveProduct(ctx context.Context, product *domain.Product) error
}

type ProductRepositoryFacade interface {
    ProductReader
    ProductWriter
}
```

3. **Service Interface:** `internal/core/ports/services/product_services.go`
```go
type ProductReaderSvc interface {
    GetProductByID(ctx context.Context, productID string) (*domain.Product, error)
}

type ProductSvcFacade interface {
    ProductReaderSvc
}
```

4. **Repository Impl:** `internal/repositories/database/pgsql/product_repository.go`
5. **Service Impl:** `internal/core/services/product_service.go`
6. **DTOs:** `internal/dto/product.go`
7. **Handler:** `internal/handlers/handler_product.go`
8. **Database Model:** `internal/models/product.go`
9. **Migration:** `migrations/000018_add_products.up.sql`
10. **Register Routes:** Update [internal/handlers/register_handlers.go](cci:7://file:///Users/sahilsoni/me/projects/money_managemet_app/internal/handlers/register_handlers.go:0:0-0:0)
11. **Wire Dependencies:** 
    - [internal/repositories/database/pgsql/repo_container.go](cci:7://file:///Users/sahilsoni/me/projects/money_managemet_app/internal/repositories/Users/sahilsoni/me/projects/money_managemet_app/internal/repositories/database/pgsql/repo_container.go:0:0-0:0)
    - [internal/core/services/services_container.go](cci:7://file:///Users/sahilsoni/me/projects/money_managemet_app/internal/core/services/Users/sahilsoni/me/projects/money_managemet_app/internal/core/services/services_container.go:0:0-0:0)
12. **Generate Swagger:** `make swagger`
