package mapping

import (
	"github.com/SscSPs/money_managemet_app/internal/core/domain"
	"github.com/SscSPs/money_managemet_app/internal/models"
)

// ToModelAccount converts a domain Account to a model Account
func ToModelAccount(d domain.Account) models.Account {
	return models.Account{
		AccountID:       d.AccountID,
		WorkplaceID:     d.WorkplaceID,
		CFID:            d.CFID,
		Name:            d.Name,
		AccountType:     models.AccountType(d.AccountType),
		CurrencyCode:    d.CurrencyCode,
		ParentAccountID: d.ParentAccountID,
		Description:     d.Description,
		IsActive:        d.IsActive,
		AuditFields: models.AuditFields{
			CreatedAt:     d.CreatedAt,
			CreatedBy:     d.CreatedBy,
			LastUpdatedAt: d.LastUpdatedAt,
			LastUpdatedBy: d.LastUpdatedBy,
		},
		Balance: d.Balance,
	}
}

// ToDomainAccount converts a model Account to a domain Account
func ToDomainAccount(m models.Account) domain.Account {
	return domain.Account{
		AccountID:       m.AccountID,
		WorkplaceID:     m.WorkplaceID,
		CFID:            m.CFID,
		Name:            m.Name,
		AccountType:     domain.AccountType(m.AccountType),
		CurrencyCode:    m.CurrencyCode,
		ParentAccountID: m.ParentAccountID,
		Description:     m.Description,
		IsActive:        m.IsActive,
		AuditFields: domain.AuditFields{
			CreatedAt:     m.CreatedAt,
			CreatedBy:     m.CreatedBy,
			LastUpdatedAt: m.LastUpdatedAt,
			LastUpdatedBy: m.LastUpdatedBy,
		},
		Balance: m.Balance,
	}
}

// ToDomainAccountSlice converts a slice of model Accounts to a slice of domain Accounts
func ToDomainAccountSlice(ms []models.Account) []domain.Account {
	ds := make([]domain.Account, len(ms))
	for i, m := range ms {
		ds[i] = ToDomainAccount(m)
	}
	return ds
}
