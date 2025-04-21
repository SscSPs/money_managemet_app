package dto

import (
	"github.com/SscSPs/money_managemet_app/internal/models"
)

type CreateJournalAndTxn struct {
	Journal      models.Journal
	Transactions []models.Transaction
}
