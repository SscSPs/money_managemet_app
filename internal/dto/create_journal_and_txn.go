package dto

import (
	"github.com/SscSPs/money_managemet_app/internal/models"
)

type CreateJournalAndTxn struct {
	Journal      models.Journal       `json:"journal"`
	Transactions []models.Transaction `json:"transactions"`
}
