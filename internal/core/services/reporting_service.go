package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/SscSPs/money_managemet_app/internal/core/domain"
	portsrepo "github.com/SscSPs/money_managemet_app/internal/core/ports/repositories"
	portssvc "github.com/SscSPs/money_managemet_app/internal/core/ports/services"
	"github.com/shopspring/decimal"
)

// reportingService implements the ReportingService interface
type reportingService struct {
	BaseService
	reportingRepo portsrepo.ReportingRepository
}

// ReportingServiceOption is a functional option for configuring the reporting service
type ReportingServiceOption func(*reportingService)

// WithReportingWorkplaceAuthorizer sets the workplace authorizer for the reporting service.
func WithReportingWorkplaceAuthorizer(authorizer portssvc.WorkplaceAuthorizerSvc) ReportingServiceOption {
	return func(s *reportingService) {
		s.WorkplaceAuthorizer = authorizer
	}
}

// NewReportingService creates a new reporting service with the provided options
func NewReportingService(repo portsrepo.ReportingRepository, options ...ReportingServiceOption) portssvc.ReportingService {
	svc := &reportingService{
		reportingRepo: repo,
	}

	// Apply all options
	for _, option := range options {
		option(svc)
	}

	return svc
}

// Ensure reportingService implements the ReportingService interface
var _ portssvc.ReportingService = (*reportingService)(nil)

// TrialBalance generates a trial balance report as of a specific date
func (s *reportingService) TrialBalance(ctx context.Context, workplaceID string, asOf time.Time, userID string) ([]domain.TrialBalanceRow, error) {
	// Authorize user action (ReadOnly is sufficient for viewing reports)
	if err := s.AuthorizeUser(ctx, userID, workplaceID, domain.RoleReadOnly); err != nil {
		s.LogError(ctx, err, "User not authorized to view trial balance report",
			slog.String("user_id", userID),
			slog.String("workplace_id", workplaceID))
		return nil, err
	}

	// Get trial balance data from repository
	trialBalanceRows, err := s.reportingRepo.GetTrialBalanceData(ctx, workplaceID, asOf)
	if err != nil {
		s.LogError(ctx, err, "Failed to retrieve trial balance data",
			slog.String("workplace_id", workplaceID),
			slog.String("asOf", asOf.Format(time.RFC3339)))
		return nil, fmt.Errorf("failed to retrieve trial balance data: %w", err)
	}

	s.LogInfo(ctx, "Trial balance report generated successfully",
		slog.String("workplace_id", workplaceID),
		slog.String("asOf", asOf.Format(time.RFC3339)),
		slog.Int("row_count", len(trialBalanceRows)))
	return trialBalanceRows, nil
}

// ProfitAndLoss generates a profit and loss report for a specific period
func (s *reportingService) ProfitAndLoss(ctx context.Context, workplaceID string, from, to time.Time, userID string) (*domain.PAndLReport, error) {
	// Authorize user action (ReadOnly is sufficient for viewing reports)
	if err := s.AuthorizeUser(ctx, userID, workplaceID, domain.RoleReadOnly); err != nil {
		s.LogError(ctx, err, "User not authorized to view profit and loss report",
			slog.String("user_id", userID),
			slog.String("workplace_id", workplaceID))
		return nil, err
	}

	// Get profit and loss data from repository
	revenue, expenses, err := s.reportingRepo.GetProfitAndLossData(ctx, workplaceID, from, to)
	if err != nil {
		s.LogError(ctx, err, "Failed to retrieve profit and loss data",
			slog.String("workplace_id", workplaceID),
			slog.String("from", from.Format(time.RFC3339)),
			slog.String("to", to.Format(time.RFC3339)))
		return nil, fmt.Errorf("failed to retrieve profit and loss data: %w", err)
	}

	// Calculate net profit
	totalRevenue := decimal.Zero
	for _, r := range revenue {
		totalRevenue = totalRevenue.Add(r.NetAmount)
	}

	totalExpenses := decimal.Zero
	for _, e := range expenses {
		totalExpenses = totalExpenses.Add(e.NetAmount)
	}

	netProfit := totalRevenue.Sub(totalExpenses)

	report := &domain.PAndLReport{
		Revenue:   revenue,
		Expenses:  expenses,
		NetProfit: netProfit,
	}

	s.LogInfo(ctx, "Profit and loss report generated successfully",
		slog.String("workplace_id", workplaceID),
		slog.String("from", from.Format(time.RFC3339)),
		slog.String("to", to.Format(time.RFC3339)),
		slog.Int("revenue_accounts", len(revenue)),
		slog.Int("expense_accounts", len(expenses)))
	return report, nil
}

// BalanceSheet generates a balance sheet report as of a specific date
func (s *reportingService) BalanceSheet(ctx context.Context, workplaceID string, asOf time.Time, userID string) (*domain.BalanceSheetReport, error) {

	// Authorize user action (ReadOnly is sufficient for viewing reports)
	if err := s.AuthorizeUser(ctx, userID, workplaceID, domain.RoleReadOnly); err != nil {
		s.LogError(ctx, err, "User not authorized to view balance sheet report",
			slog.String("user_id", userID),
			slog.String("workplace_id", workplaceID))
		return nil, err
	}

	// Get balance sheet data from repository
	assets, liabilities, equity, err := s.reportingRepo.GetBalanceSheetData(ctx, workplaceID, asOf)
	if err != nil {
		s.LogError(ctx, err, "Failed to retrieve balance sheet data",
			slog.String("workplace_id", workplaceID),
			slog.String("asOf", asOf.Format(time.RFC3339)))
		return nil, fmt.Errorf("failed to retrieve balance sheet data: %w", err)
	}

	// Calculate totals
	totalAssets := decimal.Zero
	for _, a := range assets {
		totalAssets = totalAssets.Add(a.NetAmount)
	}

	totalLiabilities := decimal.Zero
	for _, l := range liabilities {
		totalLiabilities = totalLiabilities.Add(l.NetAmount)
	}

	totalEquity := decimal.Zero
	for _, e := range equity {
		totalEquity = totalEquity.Add(e.NetAmount)
	}

	report := &domain.BalanceSheetReport{
		Assets:           assets,
		Liabilities:      liabilities,
		Equity:           equity,
		TotalAssets:      totalAssets,
		TotalLiabilities: totalLiabilities,
		TotalEquity:      totalEquity,
	}

	s.LogInfo(ctx, "Balance sheet report generated successfully",
		slog.String("workplace_id", workplaceID),
		slog.String("asOf", asOf.Format(time.RFC3339)),
		slog.Int("asset_accounts", len(assets)),
		slog.Int("liability_accounts", len(liabilities)),
		slog.Int("equity_accounts", len(equity)))
	return report, nil
}
