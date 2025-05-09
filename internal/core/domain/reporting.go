package domain

import (
	"github.com/shopspring/decimal"
)

// TrialBalanceRow represents a single row in a trial balance report
type TrialBalanceRow struct {
	AccountID   string          `json:"accountID"`
	AccountName string          `json:"accountName"`
	AccountType AccountType     `json:"accountType"`
	Debit       decimal.Decimal `json:"debit"`
	Credit      decimal.Decimal `json:"credit"`
}

// AccountAmount represents an account with its net amount for financial reports
type AccountAmount struct {
	AccountID string          `json:"accountID"`
	Name      string          `json:"name"`
	NetAmount decimal.Decimal `json:"netAmount"`
}

// PAndLReport represents a profit and loss report
type PAndLReport struct {
	Revenue   []AccountAmount `json:"revenue"`   // Net revenue accounts
	Expenses  []AccountAmount `json:"expenses"`  // Net expense accounts
	NetProfit decimal.Decimal `json:"netProfit"` // Total revenue minus total expenses
}

// BalanceSheetReport represents a balance sheet report
type BalanceSheetReport struct {
	Assets           []AccountAmount `json:"assets"`
	Liabilities      []AccountAmount `json:"liabilities"`
	Equity           []AccountAmount `json:"equity"`
	TotalAssets      decimal.Decimal `json:"totalAssets"`
	TotalLiabilities decimal.Decimal `json:"totalLiabilities"`
	TotalEquity      decimal.Decimal `json:"totalEquity"`
}
