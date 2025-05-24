package mapping

import (
	"github.com/SscSPs/money_managemet_app/internal/core/domain"
	"github.com/SscSPs/money_managemet_app/internal/models"
)

// ToModelJournal converts a domain Journal to a model Journal
func ToModelJournal(d domain.Journal) models.Journal {
	return models.Journal{
		JournalID:          d.JournalID,
		WorkplaceID:        d.WorkplaceID,
		JournalDate:        d.JournalDate,
		Description:        d.Description,
		CurrencyCode:       d.CurrencyCode,
		Status:             models.JournalStatus(d.Status),
		OriginalJournalID:  d.OriginalJournalID,
		ReversingJournalID: d.ReversingJournalID,
		Amount:             d.Amount,
		AuditFields:        ToModelAuditFields(d.AuditFields),
	}
}

// ToDomainJournal converts a model Journal to a domain Journal
func ToDomainJournal(m models.Journal) domain.Journal {
	return domain.Journal{
		JournalID:          m.JournalID,
		WorkplaceID:        m.WorkplaceID,
		JournalDate:        m.JournalDate,
		Description:        m.Description,
		CurrencyCode:       m.CurrencyCode,
		Status:             domain.JournalStatus(m.Status),
		OriginalJournalID:  m.OriginalJournalID,
		ReversingJournalID: m.ReversingJournalID,
		Amount:             m.Amount,
		AuditFields:        ToDomainAuditFields(m.AuditFields),
	}
}

// ToModelTransaction converts a domain Transaction to a model Transaction
func ToModelTransaction(d domain.Transaction) models.Transaction {
	return models.Transaction{
		TransactionID:      d.TransactionID,
		JournalID:          d.JournalID,
		AccountID:          d.AccountID,
		Amount:             d.Amount,
		TransactionType:    models.TransactionType(d.TransactionType),
		CurrencyCode:       d.CurrencyCode,
		Notes:              d.Notes,
		AuditFields:        ToModelAuditFields(d.AuditFields),
		RunningBalance:     d.RunningBalance,
		JournalDate:        d.JournalDate,
		JournalDescription: d.JournalDescription,
	}
}

// ToDomainTransaction converts a model Transaction to a domain Transaction
func ToDomainTransaction(m models.Transaction) domain.Transaction {
	return domain.Transaction{
		TransactionID:      m.TransactionID,
		JournalID:          m.JournalID,
		AccountID:          m.AccountID,
		Amount:             m.Amount,
		TransactionType:    domain.TransactionType(m.TransactionType),
		CurrencyCode:       m.CurrencyCode,
		Notes:              m.Notes,
		AuditFields:        ToDomainAuditFields(m.AuditFields),
		RunningBalance:     m.RunningBalance,
		JournalDate:        m.JournalDate,
		JournalDescription: m.JournalDescription,
	}
}

// ToDomainTransactionSlice converts a slice of model Transactions to a slice of domain Transactions
func ToDomainTransactionSlice(ms []models.Transaction) []domain.Transaction {
	ds := make([]domain.Transaction, len(ms))
	for i, m := range ms {
		ds[i] = ToDomainTransaction(m)
	}
	return ds
}
