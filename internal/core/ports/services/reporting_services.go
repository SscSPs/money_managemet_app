package services

import (
	"context"
	"time"

	"github.com/SscSPs/money_managemet_app/internal/core/domain"
)

// ReportingService defines operations for generating financial reports
type ReportingService interface {
	// TrialBalance generates a trial balance report as of a specific date
	TrialBalance(ctx context.Context, workplaceID string, asOf time.Time, userID string) ([]domain.TrialBalanceRow, error)

	// ProfitAndLoss generates a profit and loss report for a specific period
	ProfitAndLoss(ctx context.Context, workplaceID string, from, to time.Time, userID string) (*domain.PAndLReport, error)

	// BalanceSheet generates a balance sheet report as of a specific date
	BalanceSheet(ctx context.Context, workplaceID string, asOf time.Time, userID string) (*domain.BalanceSheetReport, error)
}
