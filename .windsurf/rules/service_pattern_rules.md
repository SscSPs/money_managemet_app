---
trigger: model_decision
description: when working on creating/updating/handling services
---

## ⚙️ Service Pattern

```go
package services

var _ portssvc.UserSvcFacade = (*userService)(nil)

type userService struct {
    userRepo portsrepo.UserRepositoryWithTx
}

func NewUserService(repo portsrepo.UserRepositoryWithTx) *userService {
    return &userService{userRepo: repo}
}

func (s *userService) GetUserByID(ctx context.Context, userID string) (*domain.User, error) {
    logger := middleware.GetLoggerFromCtx(ctx)
    logger.Debug("Getting user", "user_id", userID)
    
    user, err := s.userRepo.FindUserByID(ctx, userID)
    if err != nil {
        logger.Error("Error finding user", "error", err)
        return nil, err
    }
    
    return user, nil
}

func (s *userService) CreateUser(ctx context.Context, req dto.CreateUserRequest) (*domain.User, error) {
    logger := middleware.GetLoggerFromCtx(ctx)
    now := time.Now()
    
    user := &domain.User{
        UserID:   uuid.NewString(),
        Username: req.Username,
        AuditFields: domain.AuditFields{
            CreatedAt:     now,
            LastUpdatedAt: now,
            CreatedBy:     uuid.NewString(),
            LastUpdatedBy: uuid.NewString(),
        },
    }
    
    if err := s.userRepo.SaveUser(ctx, user); err != nil {
        logger.Error("Failed to save user", "error", err)
        return nil, fmt.Errorf("failed to save user: %w", err)
    }
    
    return user, nil
}
```

### Transaction Management

Manage transactions in the service layer to ensure that multiple repository calls can be executed within the same transaction. The repository methods should accept a `pgx.Tx` object.

```go
func (s *userService) Transfer(ctx context.Context, fromAccountID, toAccountID string, amount decimal.Decimal) error {
    tx, err := s.userRepo.BeginTx(ctx)
    if err != nil {
        return err
    }
    defer tx.Rollback(ctx)

    // The repository methods now accept a pgx.Tx object
    fromAccount, err := s.userRepo.GetAccountForUpdate(ctx, tx, fromAccountID)
    if err != nil {
        return err
    }

    toAccount, err := s.userRepo.GetAccountForUpdate(ctx, tx, toAccountID)
    if err != nil {
        return err
    }

    if fromAccount.Balance.LessThan(amount) {
        return apperrors.NewAppError(http.StatusBadRequest, "insufficient funds", nil)
    }

    fromAccount.Balance = fromAccount.Balance.Sub(amount)
    toAccount.Balance = toAccount.Balance.Add(amount)

    if err := s.userRepo.UpdateAccount(ctx, tx, fromAccount); err != nil {
        return err
    }

    if err := s.userRepo.UpdateAccount(ctx, tx, toAccount); err != nil {
        return err
    }

    return tx.Commit(ctx)
}
```

**Service Checklist:**
- ✅ First param is `context.Context`
- ✅ Get logger: `middleware.GetLoggerFromCtx(ctx)`
- ✅ Accept DTOs, return domain objects
- ✅ Use `uuid.NewString()` for IDs
- ✅ Set audit fields (CreatedAt, CreatedBy, etc.)
- ✅ Wrap errors: `fmt.Errorf("%w", apperrors.ErrXxx)`
- ✅ Use `errors.Is()` for sentinel errors
- ✅ Manage transactions in the service layer for operations that involve multiple repository calls.
- ❌ Never import handler or database packages

**Functional Options (for complex dependencies):**
```go
type AccountServiceOption func(*accountService)

func WithWorkplaceService(svc portssvc.WorkplaceReaderSvc) AccountServiceOption {
    return func(s *accountService) {
        s.workplaceService = svc
    }
}

func NewAccountService(repo portsrepo.AccountRepositoryFacade, options ...AccountServiceOption) portssvc.AccountSvcFacade {
    svc := &accountService{accountRepo: repo}
    for _, option := range options {
        option(svc)
    }
    return svc
}
```