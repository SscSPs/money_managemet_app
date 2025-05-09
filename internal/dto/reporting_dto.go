package dto

import (
	"time"

	"github.com/SscSPs/money_managemet_app/internal/core/domain"
	"github.com/shopspring/decimal"
)

// TrialBalanceRowResponse represents a row in the trial balance report response
type TrialBalanceRowResponse struct {
	AccountID   string          `json:"accountID"`
	AccountName string          `json:"accountName"`
	AccountType string          `json:"accountType"`
	Debit       decimal.Decimal `json:"debit"`
	Credit      decimal.Decimal `json:"credit"`
}

// TrialBalanceResponse represents the trial balance report response
type TrialBalanceResponse struct {
	AsOf   string                    `json:"asOf"`
	Rows   []TrialBalanceRowResponse `json:"rows"`
	Totals struct {
		Debit  decimal.Decimal `json:"debit"`
		Credit decimal.Decimal `json:"credit"`
	} `json:"totals"`
}

// AccountAmountResponse represents an account with its amount in a financial report
type AccountAmountResponse struct {
	AccountID string          `json:"accountID"`
	Name      string          `json:"name"`
	Amount    decimal.Decimal `json:"amount"`
}

// ProfitAndLossResponse represents the profit and loss report response
type ProfitAndLossResponse struct {
	FromDate string                  `json:"fromDate"`
	ToDate   string                  `json:"toDate"`
	Revenue  []AccountAmountResponse `json:"revenue"`
	Expenses []AccountAmountResponse `json:"expenses"`
	Summary  struct {
		TotalRevenue  decimal.Decimal `json:"totalRevenue"`
		TotalExpenses decimal.Decimal `json:"totalExpenses"`
		NetProfit     decimal.Decimal `json:"netProfit"`
	} `json:"summary"`
}

// BalanceSheetResponse represents the balance sheet report response
type BalanceSheetResponse struct {
	AsOf        string                  `json:"asOf"`
	Assets      []AccountAmountResponse `json:"assets"`
	Liabilities []AccountAmountResponse `json:"liabilities"`
	Equity      []AccountAmountResponse `json:"equity"`
	Summary     struct {
		TotalAssets      decimal.Decimal `json:"totalAssets"`
		TotalLiabilities decimal.Decimal `json:"totalLiabilities"`
		TotalEquity      decimal.Decimal `json:"totalEquity"`
	} `json:"summary"`
}

// ToTrialBalanceResponse converts domain trial balance rows to a DTO response
func ToTrialBalanceResponse(rows []domain.TrialBalanceRow, asOf time.Time) TrialBalanceResponse {
	response := TrialBalanceResponse{
		AsOf: asOf.Format("2006-01-02"),
		Rows: make([]TrialBalanceRowResponse, len(rows)),
	}

	totalDebit := decimal.Zero
	totalCredit := decimal.Zero

	for i, row := range rows {
		response.Rows[i] = TrialBalanceRowResponse{
			AccountID:   row.AccountID,
			AccountName: row.AccountName,
			AccountType: string(row.AccountType),
			Debit:       row.Debit,
			Credit:      row.Credit,
		}

		totalDebit = totalDebit.Add(row.Debit)
		totalCredit = totalCredit.Add(row.Credit)
	}

	response.Totals.Debit = totalDebit
	response.Totals.Credit = totalCredit

	return response
}

// ToProfitAndLossResponse converts a domain P&L report to a DTO response
func ToProfitAndLossResponse(report *domain.PAndLReport, from, to time.Time) ProfitAndLossResponse {
	response := ProfitAndLossResponse{
		FromDate: from.Format("2006-01-02"),
		ToDate:   to.Format("2006-01-02"),
		Revenue:  make([]AccountAmountResponse, len(report.Revenue)),
		Expenses: make([]AccountAmountResponse, len(report.Expenses)),
	}

	totalRevenue := decimal.Zero
	for i, rev := range report.Revenue {
		response.Revenue[i] = AccountAmountResponse{
			AccountID: rev.AccountID,
			Name:      rev.Name,
			Amount:    rev.NetAmount,
		}
		totalRevenue = totalRevenue.Add(rev.NetAmount)
	}

	totalExpenses := decimal.Zero
	for i, exp := range report.Expenses {
		response.Expenses[i] = AccountAmountResponse{
			AccountID: exp.AccountID,
			Name:      exp.Name,
			Amount:    exp.NetAmount,
		}
		totalExpenses = totalExpenses.Add(exp.NetAmount)
	}

	response.Summary.TotalRevenue = totalRevenue
	response.Summary.TotalExpenses = totalExpenses
	response.Summary.NetProfit = report.NetProfit

	return response
}

// ToBalanceSheetResponse converts a domain balance sheet report to a DTO response
func ToBalanceSheetResponse(report *domain.BalanceSheetReport, asOf time.Time) BalanceSheetResponse {
	response := BalanceSheetResponse{
		AsOf:        asOf.Format("2006-01-02"),
		Assets:      make([]AccountAmountResponse, len(report.Assets)),
		Liabilities: make([]AccountAmountResponse, len(report.Liabilities)),
		Equity:      make([]AccountAmountResponse, len(report.Equity)),
	}

	for i, asset := range report.Assets {
		response.Assets[i] = AccountAmountResponse{
			AccountID: asset.AccountID,
			Name:      asset.Name,
			Amount:    asset.NetAmount,
		}
	}

	for i, liability := range report.Liabilities {
		response.Liabilities[i] = AccountAmountResponse{
			AccountID: liability.AccountID,
			Name:      liability.Name,
			Amount:    liability.NetAmount,
		}
	}

	for i, equity := range report.Equity {
		response.Equity[i] = AccountAmountResponse{
			AccountID: equity.AccountID,
			Name:      equity.Name,
			Amount:    equity.NetAmount,
		}
	}

	response.Summary.TotalAssets = report.TotalAssets
	response.Summary.TotalLiabilities = report.TotalLiabilities
	response.Summary.TotalEquity = report.TotalEquity

	return response
}
