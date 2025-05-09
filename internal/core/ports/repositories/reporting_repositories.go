package repositories

import (
	"context"
	"time"

	"github.com/SscSPs/money_managemet_app/internal/core/domain"
)

// ReportingRepository defines operations for retrieving financial report data
type ReportingRepository interface {
	// GetTrialBalanceData retrieves trial balance data as of a specific date
	GetTrialBalanceData(ctx context.Context, workplaceID string, asOf time.Time) ([]domain.TrialBalanceRow, error)

	// GetProfitAndLossData retrieves profit and loss data for a specific period
	GetProfitAndLossData(ctx context.Context, workplaceID string, from, to time.Time) ([]domain.AccountAmount, []domain.AccountAmount, error)

	// GetBalanceSheetData retrieves balance sheet data as of a specific date
	GetBalanceSheetData(ctx context.Context, workplaceID string, asOf time.Time) ([]domain.AccountAmount, []domain.AccountAmount, []domain.AccountAmount, error)
}
