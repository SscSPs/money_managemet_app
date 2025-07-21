package domain_test

import (
	"testing"
	"time"

	"github.com/SscSPs/money_managemet_app/internal/core/domain"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestTransaction_IsMultiCurrency(t *testing.T) {
	tests := []struct {
		name       string
		transaction domain.Transaction
		want       bool
	}{
		{
			name: "single currency transaction",
			transaction: domain.Transaction{
				OriginalAmount:   nil,
				OriginalCurrency: nil,
			},
			want: false,
		},
		{
			name: "multi-currency transaction with USD to EUR",
			transaction: domain.Transaction{
				OriginalAmount:   decimalPtr(decimal.NewFromFloat(100.00)),
				OriginalCurrency: stringPtr("USD"),
				CurrencyCode:     "EUR",
			},
			want: true,
		},
		{
			name: "partially set multi-currency (missing original amount)",
			transaction: domain.Transaction{
				OriginalAmount:   nil,
				OriginalCurrency: stringPtr("USD"),
				CurrencyCode:     "EUR",
			},
			want: false,
		},
		{
			name: "partially set multi-currency (missing original currency)",
			transaction: domain.Transaction{
				OriginalAmount:   decimalPtr(decimal.NewFromFloat(100.00)),
				OriginalCurrency: nil,
				CurrencyCode:     "EUR",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.transaction.IsMultiCurrency()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTransaction_Validate(t *testing.T) {
	now := time.Now()
	
	tests := []struct {
		name    string
		tx      domain.Transaction
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid single currency transaction",
			tx: domain.Transaction{
				TransactionID:   "txn_123",
				JournalID:       "journal_123",
				AccountID:       "acc_123",
				Amount:          decimal.NewFromFloat(100.00),
				TransactionType: domain.Debit,
				CurrencyCode:    "USD",
				AuditFields: domain.AuditFields{
					CreatedAt: now,
					CreatedBy: "user_123",
				},
			},
			wantErr: false,
		},
		{
			name: "valid multi-currency transaction",
			tx: domain.Transaction{
				TransactionID:   "txn_123",
				JournalID:       "journal_123",
				AccountID:       "acc_123",
				Amount:          decimal.NewFromFloat(85.00),
				OriginalAmount:  decimalPtr(decimal.NewFromFloat(100.00)),
				OriginalCurrency: stringPtr("USD"),
				ExchangeRateID:  stringPtr("rate_123"),
				TransactionType: domain.Debit,
				CurrencyCode:    "EUR",
				AuditFields: domain.AuditFields{
					CreatedAt: now,
					CreatedBy: "user_123",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid multi-currency (missing exchange rate ID)",
			tx: domain.Transaction{
				TransactionID:   "txn_123",
				JournalID:       "journal_123",
				AccountID:       "acc_123",
				Amount:          decimal.NewFromFloat(85.00),
				OriginalAmount:  decimalPtr(decimal.NewFromFloat(100.00)),
				OriginalCurrency: stringPtr("USD"),
				TransactionType: domain.Debit,
				CurrencyCode:    "EUR",
			},
			wantErr: true,
			errMsg:  "exchange rate ID is required for multi-currency transactions",
		},
		{
			name: "invalid multi-currency (original amount zero)",
			tx: domain.Transaction{
				TransactionID:   "txn_123",
				JournalID:       "journal_123",
				AccountID:       "acc_123",
				Amount:          decimal.NewFromFloat(85.00),
				OriginalAmount:  decimalPtr(decimal.Zero),
				OriginalCurrency: stringPtr("USD"),
				ExchangeRateID:  stringPtr("rate_123"),
				TransactionType: domain.Debit,
				CurrencyCode:    "EUR",
			},
			wantErr: true,
			errMsg:  "original amount must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tx.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Helper functions
func decimalPtr(d decimal.Decimal) *decimal.Decimal {
	return &d
}

func stringPtr(s string) *string {
	return &s
}
