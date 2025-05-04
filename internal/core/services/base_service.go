package services

import (
	"context"
	"log/slog"

	"github.com/SscSPs/money_managemet_app/internal/core/domain"
	portssvc "github.com/SscSPs/money_managemet_app/internal/core/ports/services"
	"github.com/SscSPs/money_managemet_app/internal/middleware"
)

// BaseService provides common functionality for all services
type BaseService struct {
	WorkplaceAuthorizer portssvc.WorkplaceAuthorizerSvc
}

// GetLogger gets the logger from context or returns a default one
func (s *BaseService) GetLogger(ctx context.Context) *slog.Logger {
	logger := middleware.GetLoggerFromCtx(ctx)
	if logger == nil {
		// Return a default logger if not found in context
		return slog.Default()
	}
	return logger
}

// LogError logs an error with consistent formatting
func (s *BaseService) LogError(ctx context.Context, err error, msg string, keyvals ...any) {
	logger := s.GetLogger(ctx)
	args := make([]any, 0, len(keyvals)+2)
	args = append(args, slog.String("error", err.Error()))
	args = append(args, keyvals...)
	logger.Error(msg, args...)
}

// LogInfo logs an info message with consistent formatting
func (s *BaseService) LogInfo(ctx context.Context, msg string, keyvals ...any) {
	logger := s.GetLogger(ctx)
	logger.Info(msg, keyvals...)
}

// LogDebug logs a debug message with consistent formatting
func (s *BaseService) LogDebug(ctx context.Context, msg string, keyvals ...any) {
	logger := s.GetLogger(ctx)
	logger.Debug(msg, keyvals...)
}

// AuthorizeUser checks if a user has the required role for a workplace
func (s *BaseService) AuthorizeUser(ctx context.Context, userID, workplaceID string, requiredRole domain.UserWorkplaceRole) error {
	if s.WorkplaceAuthorizer != nil {
		return s.WorkplaceAuthorizer.AuthorizeUserAction(ctx, userID, workplaceID, requiredRole)
	}
	// If no authorizer is provided, we could either:
	// 1. Skip authorization (security risk)
	// 2. Deny all access (safe but might break functionality)
	// 3. Log a warning and allow (development mode behavior)
	s.LogDebug(ctx, "No workplace authorizer provided, access granted by default",
		slog.String("user_id", userID),
		slog.String("workplace_id", workplaceID),
		slog.String("required_role", string(requiredRole)))
	return nil
}
