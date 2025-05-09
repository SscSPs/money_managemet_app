package pgsql

import (
	"context"
	"fmt"
	"time"

	"github.com/SscSPs/money_managemet_app/internal/core/domain"
	portsrepo "github.com/SscSPs/money_managemet_app/internal/core/ports/repositories"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// reportingRepository implements the ReportingRepository interface
type reportingRepository struct {
	BaseRepository
}

// NewReportingRepository creates a new reporting repository
func newReportingRepository(db *pgxpool.Pool) portsrepo.ReportingRepository {
	return &reportingRepository{
		BaseRepository: BaseRepository{Pool: db},
	}
}

// GetTrialBalanceData retrieves trial balance data as of a specific date
func (r *reportingRepository) GetTrialBalanceData(ctx context.Context, workplaceID string, asOf time.Time) ([]domain.TrialBalanceRow, error) {
	query := `
		SELECT
			a.account_id,
			a.name AS account_name,
			a.account_type,
			SUM(CASE WHEN t.transaction_type = 'DEBIT' THEN t.amount ELSE 0 END) AS total_debit,
			SUM(CASE WHEN t.transaction_type = 'CREDIT' THEN t.amount ELSE 0 END) AS total_credit
		FROM transactions t
		JOIN accounts a ON t.account_id = a.account_id
		JOIN journals j ON t.journal_id = j.journal_id
		WHERE j.journal_date <= $1
			AND a.workplace_id = $2
			AND j.status = 'POSTED'
			AND j.original_journal_id IS NULL
		GROUP BY a.account_id, a.name, a.account_type
	`

	rows, err := r.Pool.Query(ctx, query, asOf, workplaceID)
	if err != nil {
		return nil, fmt.Errorf("error querying trial balance data: %w", err)
	}
	defer rows.Close()

	var result []domain.TrialBalanceRow
	for rows.Next() {
		var row domain.TrialBalanceRow
		var accountType string

		if err := rows.Scan(
			&row.AccountID,
			&row.AccountName,
			&accountType,
			&row.Debit,
			&row.Credit,
		); err != nil {
			return nil, fmt.Errorf("error scanning trial balance row: %w", err)
		}

		row.AccountType = domain.AccountType(accountType)
		result = append(result, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating trial balance rows: %w", err)
	}

	if len(result) == 0 {
		// Return empty slice instead of nil
		return []domain.TrialBalanceRow{}, nil
	}

	return result, nil
}

// GetProfitAndLossData retrieves profit and loss data for a specific period
func (r *reportingRepository) GetProfitAndLossData(ctx context.Context, workplaceID string, from, to time.Time) ([]domain.AccountAmount, []domain.AccountAmount, error) {
	query := `
		SELECT
			a.account_type,
			a.account_id,
			a.name,
			SUM(CASE WHEN t.transaction_type = 'DEBIT' THEN t.amount ELSE -t.amount END) AS net
		FROM transactions t
		JOIN accounts a ON t.account_id = a.account_id
		JOIN journals j ON t.journal_id = j.journal_id
		WHERE j.journal_date BETWEEN $1 AND $2
			AND a.workplace_id = $3
			AND j.status = 'POSTED'
			AND j.original_journal_id IS NULL
			AND a.account_type IN ('REVENUE', 'EXPENSE')
		GROUP BY a.account_type, a.account_id, a.name
	`

	rows, err := r.Pool.Query(ctx, query, from, to, workplaceID)
	if err != nil {
		return nil, nil, fmt.Errorf("error querying profit and loss data: %w", err)
	}
	defer rows.Close()

	var revenue []domain.AccountAmount
	var expenses []domain.AccountAmount

	for rows.Next() {
		var accountType, accountID, name string
		var netAmount decimal.Decimal

		if err := rows.Scan(&accountType, &accountID, &name, &netAmount); err != nil {
			return nil, nil, fmt.Errorf("error scanning profit and loss row: %w", err)
		}

		accountAmount := domain.AccountAmount{
			AccountID: accountID,
			Name:      name,
			NetAmount: netAmount.Abs(), // Store absolute value
		}

		// For revenue accounts, credit increases (negative net amount means credit)
		// For expense accounts, debit increases (positive net amount means debit)
		if accountType == string(domain.Revenue) {
			// Revenue accounts: invert sign since credits increase revenue
			accountAmount.NetAmount = netAmount.Neg()
			revenue = append(revenue, accountAmount)
		} else if accountType == string(domain.Expense) {
			// Expense accounts: keep sign as is since debits increase expenses
			expenses = append(expenses, accountAmount)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("error iterating profit and loss rows: %w", err)
	}

	// Return empty slices instead of nil
	if revenue == nil {
		revenue = []domain.AccountAmount{}
	}
	if expenses == nil {
		expenses = []domain.AccountAmount{}
	}

	return revenue, expenses, nil
}

// GetBalanceSheetData retrieves balance sheet data as of a specific date
func (r *reportingRepository) GetBalanceSheetData(ctx context.Context, workplaceID string, asOf time.Time) ([]domain.AccountAmount, []domain.AccountAmount, []domain.AccountAmount, error) {
	query := `
		SELECT
			a.account_type,
			a.account_id,
			a.name,
			SUM(CASE WHEN t.transaction_type = 'DEBIT' THEN t.amount ELSE -t.amount END) AS net
		FROM transactions t
		JOIN accounts a ON t.account_id = a.account_id
		JOIN journals j ON t.journal_id = j.journal_id
		WHERE j.journal_date <= $1
			AND a.workplace_id = $2
			AND j.status = 'POSTED'
			AND j.original_journal_id IS NULL
			AND a.account_type IN ('ASSET', 'LIABILITY', 'EQUITY')
		GROUP BY a.account_type, a.account_id, a.name
	`

	rows, err := r.Pool.Query(ctx, query, asOf, workplaceID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error querying balance sheet data: %w", err)
	}
	defer rows.Close()

	var assets []domain.AccountAmount
	var liabilities []domain.AccountAmount
	var equity []domain.AccountAmount

	for rows.Next() {
		var accountType, accountID, name string
		var netAmount decimal.Decimal

		if err := rows.Scan(&accountType, &accountID, &name, &netAmount); err != nil {
			return nil, nil, nil, fmt.Errorf("error scanning balance sheet row: %w", err)
		}

		accountAmount := domain.AccountAmount{
			AccountID: accountID,
			Name:      name,
			NetAmount: netAmount,
		}

		switch accountType {
		case string(domain.Asset):
			// Asset accounts: debit increases (positive net amount)
			assets = append(assets, accountAmount)
		case string(domain.Liability):
			// Liability accounts: credit increases (negative net amount)
			// Invert sign for display purposes
			accountAmount.NetAmount = netAmount.Neg()
			liabilities = append(liabilities, accountAmount)
		case string(domain.Equity):
			// Equity accounts: credit increases (negative net amount)
			// Invert sign for display purposes
			accountAmount.NetAmount = netAmount.Neg()
			equity = append(equity, accountAmount)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, nil, nil, fmt.Errorf("error iterating balance sheet rows: %w", err)
	}

	// Return empty slices instead of nil
	if assets == nil {
		assets = []domain.AccountAmount{}
	}
	if liabilities == nil {
		liabilities = []domain.AccountAmount{}
	}
	if equity == nil {
		equity = []domain.AccountAmount{}
	}

	return assets, liabilities, equity, nil
}
