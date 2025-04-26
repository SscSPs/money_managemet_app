package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/SscSPs/money_managemet_app/internal/apperrors"
	"github.com/SscSPs/money_managemet_app/internal/core/domain"
	"github.com/SscSPs/money_managemet_app/internal/core/services"
	"github.com/SscSPs/money_managemet_app/internal/dto"
	"github.com/SscSPs/money_managemet_app/internal/models"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// --- Mock JournalRepository ---
type MockJournalRepository struct {
	mock.Mock
}

func (m *MockJournalRepository) SaveJournal(ctx context.Context, journal domain.Journal, transactions []domain.Transaction) error {
	args := m.Called(ctx, journal, transactions)
	return args.Error(0)
}

func (m *MockJournalRepository) FindJournalByID(ctx context.Context, journalID string) (*domain.Journal, error) {
	args := m.Called(ctx, journalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Journal), args.Error(1)
}

func (m *MockJournalRepository) FindTransactionsByJournalID(ctx context.Context, journalID string) ([]domain.Transaction, error) {
	args := m.Called(ctx, journalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Transaction), args.Error(1)
}

func (m *MockJournalRepository) FindTransactionsByAccountID(ctx context.Context, accountID string) ([]domain.Transaction, error) {
	args := m.Called(ctx, accountID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Transaction), args.Error(1)
}

// --- Mock AccountRepository (Use definition from account_service_test.go) ---
/* // Remove duplicated definition
type MockAccountRepository struct {
	mock.Mock
}

func (m *MockAccountRepository) SaveAccount(ctx context.Context, account domain.Account) error {
	args := m.Called(ctx, account)
	return args.Error(0)
}

func (m *MockAccountRepository) FindAccountByID(ctx context.Context, accountID string) (*domain.Account, error) {
	args := m.Called(ctx, accountID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Account), args.Error(1)
}

func (m *MockAccountRepository) FindAccountsByIDs(ctx context.Context, accountIDs []string) (map[string]domain.Account, error) {
	args := m.Called(ctx, accountIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]domain.Account), args.Error(1)
}

func (m *MockAccountRepository) ListAccounts(ctx context.Context, limit int, offset int) ([]domain.Account, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Account), args.Error(1)
}

func (m *MockAccountRepository) DeactivateAccount(ctx context.Context, accountID string, userID string, now time.Time) error {
	args := m.Called(ctx, accountID, userID, now)
	return args.Error(0)
}
*/

// --- Test Suite ---
type JournalServiceTestSuite struct {
	suite.Suite
	mockJournalRepo *MockJournalRepository
	mockAccountRepo *MockAccountRepository // This will use the definition from the other file
	service         *services.JournalService
}

func (suite *JournalServiceTestSuite) SetupTest() {
	suite.mockJournalRepo = new(MockJournalRepository)
	suite.mockAccountRepo = new(MockAccountRepository)
	suite.service = services.NewJournalService(suite.mockAccountRepo, suite.mockJournalRepo)
}

// --- Test Cases ---

// Helper function to create a basic valid DTO for PersistJournal tests
func createValidJournalDTO(currency string, accID1, accID2 string, amount decimal.Decimal) dto.CreateJournalAndTxn {
	return dto.CreateJournalAndTxn{
		Journal: models.Journal{
			JournalDate:  time.Now(),
			Description:  "Test Journal",
			CurrencyCode: currency,
		},
		Transactions: []models.Transaction{
			{
				AccountID:       accID1,
				Amount:          amount,
				TransactionType: models.Debit,
				CurrencyCode:    currency,
			},
			{
				AccountID:       accID2,
				Amount:          amount,
				TransactionType: models.Credit,
				CurrencyCode:    currency,
			},
		},
	}
}

// --- PersistJournal Tests ---
func (suite *JournalServiceTestSuite) TestPersistJournal_Success() {
	ctx := context.Background()
	userID := uuid.NewString()
	accID1 := uuid.NewString() // Asset
	accID2 := uuid.NewString() // << CHANGE: Make this an Asset too for balance
	currency := "USD"
	amount := decimal.NewFromInt(100)
	req := createValidJournalDTO(currency, accID1, accID2, amount) // Debit acc1, Credit acc2

	// Mock account fetching
	accountsMap := map[string]domain.Account{
		accID1: {AccountID: accID1, AccountType: domain.Asset, IsActive: true, CurrencyCode: currency},
		accID2: {AccountID: accID2, AccountType: domain.Asset, IsActive: true, CurrencyCode: currency}, // << CHANGE: Type is Asset
	}
	// Balance check: Debit Asset (+100), Credit Asset (-100) -> Sum = 0. OK.
	suite.mockAccountRepo.On("FindAccountsByIDs", ctx, mock.AnythingOfType("[]string")).Return(accountsMap, nil).Once() // Adjusted mock expectation

	// Mock journal saving
	suite.mockJournalRepo.On("SaveJournal", ctx, mock.AnythingOfType("domain.Journal"), mock.AnythingOfType("[]domain.Transaction")).Return(nil).Once().Run(func(args mock.Arguments) {
		jnl := args.Get(1).(domain.Journal)
		txns := args.Get(2).([]domain.Transaction)

		suite.Equal(req.Journal.Description, jnl.Description)
		suite.Equal(req.Journal.CurrencyCode, jnl.CurrencyCode)
		suite.Equal(domain.Posted, jnl.Status)
		suite.NotEmpty(jnl.JournalID)
		suite.Equal(userID, jnl.CreatedBy)
		suite.Len(txns, 2)
		// Basic checks on transactions
		for _, txn := range txns {
			suite.Equal(jnl.JournalID, txn.JournalID)
			suite.NotEmpty(txn.TransactionID)
			suite.Equal(amount, txn.Amount)
			suite.Equal(currency, txn.CurrencyCode)
			suite.Equal(userID, txn.CreatedBy)
		}
	})

	journal, err := suite.service.PersistJournal(ctx, req, userID)

	suite.Require().NoError(err)
	suite.Require().NotNil(journal)
	suite.NotEmpty(journal.JournalID)
	suite.Equal(req.Journal.CurrencyCode, journal.CurrencyCode)
	suite.Equal(domain.Posted, journal.Status)

	suite.mockAccountRepo.AssertExpectations(suite.T())
	suite.mockJournalRepo.AssertExpectations(suite.T())
}

func (suite *JournalServiceTestSuite) TestPersistJournal_LessThanTwoTransactions() {
	ctx := context.Background()
	userID := uuid.NewString()
	req := createValidJournalDTO("USD", "acc1", "acc2", decimal.NewFromInt(10))
	req.Transactions = req.Transactions[:1]
	journal, err := suite.service.PersistJournal(ctx, req, userID)
	suite.Require().Error(err)
	suite.Nil(journal)
	suite.ErrorIs(err, services.ErrJournalMinEntries)
	suite.mockAccountRepo.AssertNotCalled(suite.T(), "FindAccountsByIDs", mock.Anything, mock.Anything)
	suite.mockJournalRepo.AssertNotCalled(suite.T(), "SaveJournal", mock.Anything, mock.Anything, mock.Anything)
}

func (suite *JournalServiceTestSuite) TestPersistJournal_CurrencyMismatch() {
	ctx := context.Background()
	userID := uuid.NewString()
	req := createValidJournalDTO("USD", "acc1", "acc2", decimal.NewFromInt(10))
	req.Transactions[1].CurrencyCode = "EUR"
	journal, err := suite.service.PersistJournal(ctx, req, userID)
	suite.Require().Error(err)
	suite.Nil(journal)
	suite.ErrorIs(err, services.ErrCurrencyMismatch)
	suite.Contains(err.Error(), "journal is USD, transaction involves EUR")
	suite.mockAccountRepo.AssertNotCalled(suite.T(), "FindAccountsByIDs", mock.Anything, mock.Anything)
	suite.mockJournalRepo.AssertNotCalled(suite.T(), "SaveJournal", mock.Anything, mock.Anything, mock.Anything)
}

func (suite *JournalServiceTestSuite) TestPersistJournal_AccountNotFound() {
	ctx := context.Background()
	userID := uuid.NewString()
	accID1 := uuid.NewString()
	accID2 := uuid.NewString()
	req := createValidJournalDTO("CAD", accID1, accID2, decimal.NewFromInt(50))
	accountsMap := map[string]domain.Account{
		accID1: {AccountID: accID1, AccountType: domain.Asset, IsActive: true, CurrencyCode: "CAD"},
	}
	suite.mockAccountRepo.On("FindAccountsByIDs", ctx, mock.AnythingOfType("[]string")).Return(accountsMap, nil).Once()
	journal, err := suite.service.PersistJournal(ctx, req, userID)
	suite.Require().Error(err)
	suite.Nil(journal)
	suite.ErrorIs(err, services.ErrAccountNotFound)
	suite.Contains(err.Error(), accID2)
	suite.mockAccountRepo.AssertExpectations(suite.T())
	suite.mockJournalRepo.AssertNotCalled(suite.T(), "SaveJournal", mock.Anything, mock.Anything, mock.Anything)
}

func (suite *JournalServiceTestSuite) TestPersistJournal_AccountInactive() {
	ctx := context.Background()
	userID := uuid.NewString()
	accID1 := uuid.NewString()
	accID2 := uuid.NewString()
	req := createValidJournalDTO("EUR", accID1, accID2, decimal.NewFromInt(200))
	accountsMap := map[string]domain.Account{
		accID1: {AccountID: accID1, AccountType: domain.Asset, IsActive: true, CurrencyCode: "EUR"},
		accID2: {AccountID: accID2, AccountType: domain.Liability, IsActive: false, CurrencyCode: "EUR"},
	}
	suite.mockAccountRepo.On("FindAccountsByIDs", ctx, mock.AnythingOfType("[]string")).Return(accountsMap, nil).Once()
	journal, err := suite.service.PersistJournal(ctx, req, userID)
	suite.Require().Error(err)
	suite.Nil(journal)
	suite.Contains(err.Error(), "account")
	suite.Contains(err.Error(), accID2)
	suite.Contains(err.Error(), "is inactive")
	suite.mockAccountRepo.AssertExpectations(suite.T())
	suite.mockJournalRepo.AssertNotCalled(suite.T(), "SaveJournal", mock.Anything, mock.Anything, mock.Anything)
}

func (suite *JournalServiceTestSuite) TestPersistJournal_Unbalanced() {
	ctx := context.Background()
	userID := uuid.NewString()
	accID1 := uuid.NewString()
	accID2 := uuid.NewString()
	req := createValidJournalDTO("USD", accID1, accID2, decimal.NewFromInt(100))
	req.Transactions[1].Amount = decimal.NewFromInt(99)
	accountsMap := map[string]domain.Account{
		accID1: {AccountID: accID1, AccountType: domain.Asset, IsActive: true, CurrencyCode: "USD"},
		accID2: {AccountID: accID2, AccountType: domain.Liability, IsActive: true, CurrencyCode: "USD"},
	}
	suite.mockAccountRepo.On("FindAccountsByIDs", ctx, mock.AnythingOfType("[]string")).Return(accountsMap, nil).Once()
	journal, err := suite.service.PersistJournal(ctx, req, userID)
	suite.Require().Error(err)
	suite.Nil(journal)
	suite.ErrorIs(err, services.ErrJournalUnbalanced)
	suite.Contains(err.Error(), "sum is 1")
	suite.mockAccountRepo.AssertExpectations(suite.T())
	suite.mockJournalRepo.AssertNotCalled(suite.T(), "SaveJournal", mock.Anything, mock.Anything, mock.Anything)
}

func (suite *JournalServiceTestSuite) TestPersistJournal_FindAccountsError() {
	ctx := context.Background()
	userID := uuid.NewString()
	accID1 := uuid.NewString()
	accID2 := uuid.NewString()
	req := createValidJournalDTO("USD", accID1, accID2, decimal.NewFromInt(100))
	expectedErr := assert.AnError
	suite.mockAccountRepo.On("FindAccountsByIDs", ctx, mock.AnythingOfType("[]string")).Return(nil, expectedErr).Once()
	journal, err := suite.service.PersistJournal(ctx, req, userID)
	suite.Require().Error(err)
	suite.Nil(journal)
	suite.ErrorIs(err, expectedErr)
	suite.Contains(err.Error(), "failed to fetch accounts")
	suite.mockAccountRepo.AssertExpectations(suite.T())
	suite.mockJournalRepo.AssertNotCalled(suite.T(), "SaveJournal", mock.Anything, mock.Anything, mock.Anything)
}

func (suite *JournalServiceTestSuite) TestPersistJournal_SaveJournalError() {
	ctx := context.Background()
	userID := uuid.NewString()
	accID1 := uuid.NewString() // Asset
	accID2 := uuid.NewString() // << CHANGE: Make this an Asset too for balance
	req := createValidJournalDTO("USD", accID1, accID2, decimal.NewFromInt(100))
	expectedErr := assert.AnError

	// Mock account fetching (success)
	accountsMap := map[string]domain.Account{
		accID1: {AccountID: accID1, AccountType: domain.Asset, IsActive: true, CurrencyCode: "USD"},
		accID2: {AccountID: accID2, AccountType: domain.Asset, IsActive: true, CurrencyCode: "USD"}, // << CHANGE: Type is Asset
	}
	// Balance check: Debit Asset (+100), Credit Asset (-100) -> Sum = 0. OK.
	suite.mockAccountRepo.On("FindAccountsByIDs", ctx, mock.AnythingOfType("[]string")).Return(accountsMap, nil).Once() // Adjusted mock expectation

	// Mock journal saving returning an error
	suite.mockJournalRepo.On("SaveJournal", ctx, mock.AnythingOfType("domain.Journal"), mock.AnythingOfType("[]domain.Transaction")).Return(expectedErr).Once()

	journal, err := suite.service.PersistJournal(ctx, req, userID)

	suite.Require().Error(err)
	suite.Nil(journal)
	suite.ErrorIs(err, expectedErr) // Now we expect the save error
	suite.Contains(err.Error(), "failed to save journal")
	suite.mockAccountRepo.AssertExpectations(suite.T())
	suite.mockJournalRepo.AssertExpectations(suite.T())
}

// --- GetJournalWithTransactions Tests ---
func (suite *JournalServiceTestSuite) TestGetJournalWithTransactions_Success() {
	ctx := context.Background()
	journalID := uuid.NewString()
	expectedJournal := &domain.Journal{JournalID: journalID, Description: "Found Journal"}
	expectedTxns := []domain.Transaction{
		{TransactionID: uuid.NewString(), JournalID: journalID, Amount: decimal.NewFromInt(50)},
		{TransactionID: uuid.NewString(), JournalID: journalID, Amount: decimal.NewFromInt(50)},
	}
	suite.mockJournalRepo.On("FindJournalByID", ctx, journalID).Return(expectedJournal, nil).Once()
	suite.mockJournalRepo.On("FindTransactionsByJournalID", ctx, journalID).Return(expectedTxns, nil).Once()
	journal, txns, err := suite.service.GetJournalWithTransactions(ctx, journalID)
	suite.Require().NoError(err)
	suite.Equal(expectedJournal, journal)
	suite.Equal(expectedTxns, txns)
	suite.mockJournalRepo.AssertExpectations(suite.T())
}

func (suite *JournalServiceTestSuite) TestGetJournalWithTransactions_JournalNotFound() {
	ctx := context.Background()
	journalID := uuid.NewString()
	suite.mockJournalRepo.On("FindJournalByID", ctx, journalID).Return(nil, nil).Once()
	journal, txns, err := suite.service.GetJournalWithTransactions(ctx, journalID)
	suite.Require().Error(err)
	suite.Nil(journal)
	suite.Nil(txns)
	suite.Contains(err.Error(), "not found")
	suite.mockJournalRepo.AssertExpectations(suite.T())
	suite.mockJournalRepo.AssertNotCalled(suite.T(), "FindTransactionsByJournalID", mock.Anything)
}

func (suite *JournalServiceTestSuite) TestGetJournalWithTransactions_FindJournalError() {
	ctx := context.Background()
	journalID := uuid.NewString()
	expectedErr := assert.AnError
	suite.mockJournalRepo.On("FindJournalByID", ctx, journalID).Return(nil, expectedErr).Once()
	journal, txns, err := suite.service.GetJournalWithTransactions(ctx, journalID)
	suite.Require().Error(err)
	suite.Nil(journal)
	suite.Nil(txns)
	suite.ErrorIs(err, expectedErr)
	suite.Contains(err.Error(), "failed to find journal")
	suite.mockJournalRepo.AssertExpectations(suite.T())
	suite.mockJournalRepo.AssertNotCalled(suite.T(), "FindTransactionsByJournalID", mock.Anything)
}

func (suite *JournalServiceTestSuite) TestGetJournalWithTransactions_FindTransactionsError() {
	ctx := context.Background()
	journalID := uuid.NewString()
	expectedJournal := &domain.Journal{JournalID: journalID}
	expectedErr := assert.AnError
	suite.mockJournalRepo.On("FindJournalByID", ctx, journalID).Return(expectedJournal, nil).Once()
	suite.mockJournalRepo.On("FindTransactionsByJournalID", ctx, journalID).Return(nil, expectedErr).Once()
	journal, txns, err := suite.service.GetJournalWithTransactions(ctx, journalID)
	suite.Require().Error(err)
	suite.Nil(journal)
	suite.Nil(txns)
	suite.ErrorIs(err, expectedErr)
	suite.Contains(err.Error(), "failed to find transactions")
	suite.mockJournalRepo.AssertExpectations(suite.T())
}

// --- CalculateAccountBalance Tests ---
func (suite *JournalServiceTestSuite) TestCalculateAccountBalance_Success_Asset() {
	ctx := context.Background()
	accountID := uuid.NewString()
	account := &domain.Account{
		AccountID:   accountID,
		AccountType: domain.Asset,
		IsActive:    true,
	}
	transactions := []domain.Transaction{
		{AccountID: accountID, Amount: decimal.NewFromInt(100), TransactionType: domain.Debit},
		{AccountID: accountID, Amount: decimal.NewFromInt(30), TransactionType: domain.Credit},
		{AccountID: accountID, Amount: decimal.NewFromInt(50), TransactionType: domain.Debit},
	}
	expectedBalance := decimal.NewFromInt(120)
	suite.mockAccountRepo.On("FindAccountByID", ctx, accountID).Return(account, nil).Once()
	suite.mockJournalRepo.On("FindTransactionsByAccountID", ctx, accountID).Return(transactions, nil).Once()
	balance, err := suite.service.CalculateAccountBalance(ctx, accountID)
	suite.Require().NoError(err)
	suite.True(expectedBalance.Equal(balance), "Expected %s, got %s", expectedBalance, balance)
	suite.mockAccountRepo.AssertExpectations(suite.T())
	suite.mockJournalRepo.AssertExpectations(suite.T())
}

func (suite *JournalServiceTestSuite) TestCalculateAccountBalance_Success_LiabilityZero() {
	ctx := context.Background()
	accountID := uuid.NewString()
	account := &domain.Account{
		AccountID:   accountID,
		AccountType: domain.Liability,
		IsActive:    true,
	}
	transactions := []domain.Transaction{
		{AccountID: accountID, Amount: decimal.NewFromInt(200), TransactionType: domain.Credit},
		{AccountID: accountID, Amount: decimal.NewFromInt(200), TransactionType: domain.Debit},
	}
	expectedBalance := decimal.Zero
	suite.mockAccountRepo.On("FindAccountByID", ctx, accountID).Return(account, nil).Once()
	suite.mockJournalRepo.On("FindTransactionsByAccountID", ctx, accountID).Return(transactions, nil).Once()
	balance, err := suite.service.CalculateAccountBalance(ctx, accountID)
	suite.Require().NoError(err)
	suite.True(expectedBalance.Equal(balance), "Expected %s, got %s", expectedBalance, balance)
	suite.mockAccountRepo.AssertExpectations(suite.T())
	suite.mockJournalRepo.AssertExpectations(suite.T())
}

func (suite *JournalServiceTestSuite) TestCalculateAccountBalance_Success_EquityNegative() {
	ctx := context.Background()
	accountID := uuid.NewString()
	account := &domain.Account{
		AccountID:   accountID,
		AccountType: domain.Equity,
		IsActive:    true,
	}
	transactions := []domain.Transaction{
		{AccountID: accountID, Amount: decimal.NewFromInt(500), TransactionType: domain.Credit},
		{AccountID: accountID, Amount: decimal.NewFromInt(600), TransactionType: domain.Debit},
	}
	expectedBalance := decimal.NewFromInt(-100)
	suite.mockAccountRepo.On("FindAccountByID", ctx, accountID).Return(account, nil).Once()
	suite.mockJournalRepo.On("FindTransactionsByAccountID", ctx, accountID).Return(transactions, nil).Once()
	balance, err := suite.service.CalculateAccountBalance(ctx, accountID)
	suite.Require().NoError(err)
	suite.True(expectedBalance.Equal(balance), "Expected %s, got %s", expectedBalance, balance)
	suite.mockAccountRepo.AssertExpectations(suite.T())
	suite.mockJournalRepo.AssertExpectations(suite.T())
}

func (suite *JournalServiceTestSuite) TestCalculateAccountBalance_NoTransactions() {
	ctx := context.Background()
	accountID := uuid.NewString()
	account := &domain.Account{
		AccountID:   accountID,
		AccountType: domain.Asset,
		IsActive:    true,
	}
	var transactions []domain.Transaction
	suite.mockAccountRepo.On("FindAccountByID", ctx, accountID).Return(account, nil).Once()
	suite.mockJournalRepo.On("FindTransactionsByAccountID", ctx, accountID).Return(transactions, nil).Once()
	balance, err := suite.service.CalculateAccountBalance(ctx, accountID)
	suite.Require().NoError(err)
	suite.True(decimal.Zero.Equal(balance), "Expected 0, got %s", balance)
	suite.mockAccountRepo.AssertExpectations(suite.T())
	suite.mockJournalRepo.AssertExpectations(suite.T())
}

func (suite *JournalServiceTestSuite) TestCalculateAccountBalance_AccountNotFound() {
	ctx := context.Background()
	accountID := uuid.NewString()
	suite.mockAccountRepo.On("FindAccountByID", ctx, accountID).Return(nil, apperrors.ErrNotFound).Once()
	balance, err := suite.service.CalculateAccountBalance(ctx, accountID)
	suite.Require().Error(err)
	suite.ErrorIs(err, services.ErrAccountNotFound)
	suite.True(decimal.Zero.Equal(balance))
	suite.mockAccountRepo.AssertExpectations(suite.T())
	suite.mockJournalRepo.AssertNotCalled(suite.T(), "FindTransactionsByAccountID", mock.Anything)
}

func (suite *JournalServiceTestSuite) TestCalculateAccountBalance_AccountInactive() {
	ctx := context.Background()
	accountID := uuid.NewString()
	account := &domain.Account{
		AccountID:   accountID,
		AccountType: domain.Asset,
		IsActive:    false,
	}
	suite.mockAccountRepo.On("FindAccountByID", ctx, accountID).Return(account, nil).Once()
	balance, err := suite.service.CalculateAccountBalance(ctx, accountID)
	suite.Require().Error(err)
	suite.Contains(err.Error(), "account")
	suite.Contains(err.Error(), "is inactive")
	suite.True(decimal.Zero.Equal(balance))
	suite.mockAccountRepo.AssertExpectations(suite.T())
	suite.mockJournalRepo.AssertNotCalled(suite.T(), "FindTransactionsByAccountID", mock.Anything)
}

func (suite *JournalServiceTestSuite) TestCalculateAccountBalance_FindAccountError() {
	ctx := context.Background()
	accountID := uuid.NewString()
	expectedErr := assert.AnError
	suite.mockAccountRepo.On("FindAccountByID", ctx, accountID).Return(nil, expectedErr).Once()
	balance, err := suite.service.CalculateAccountBalance(ctx, accountID)
	suite.Require().Error(err)
	suite.ErrorIs(err, expectedErr)
	suite.Contains(err.Error(), "failed to find account")
	suite.True(decimal.Zero.Equal(balance))
	suite.mockAccountRepo.AssertExpectations(suite.T())
	suite.mockJournalRepo.AssertNotCalled(suite.T(), "FindTransactionsByAccountID", mock.Anything)
}

func (suite *JournalServiceTestSuite) TestCalculateAccountBalance_FindTransactionsError() {
	ctx := context.Background()
	accountID := uuid.NewString()
	account := &domain.Account{AccountID: accountID, AccountType: domain.Asset, IsActive: true}
	expectedErr := assert.AnError
	suite.mockAccountRepo.On("FindAccountByID", ctx, accountID).Return(account, nil).Once()
	suite.mockJournalRepo.On("FindTransactionsByAccountID", ctx, accountID).Return(nil, expectedErr).Once()
	balance, err := suite.service.CalculateAccountBalance(ctx, accountID)
	suite.Require().Error(err)
	suite.ErrorIs(err, expectedErr)
	suite.Contains(err.Error(), "failed to fetch transactions")
	suite.True(decimal.Zero.Equal(balance))
	suite.mockAccountRepo.AssertExpectations(suite.T())
	suite.mockJournalRepo.AssertExpectations(suite.T())
}

func (suite *JournalServiceTestSuite) TestCalculateAccountBalance_InvalidTransactionAmount() {
	ctx := context.Background()
	accountID := uuid.NewString()
	account := &domain.Account{
		AccountID:   accountID,
		AccountType: domain.Asset,
		IsActive:    true,
	}
	transactions := []domain.Transaction{
		{AccountID: accountID, Amount: decimal.NewFromInt(100), TransactionType: domain.Debit},
		{AccountID: accountID, Amount: decimal.NewFromInt(0), TransactionType: domain.Credit},
	}

	suite.mockAccountRepo.On("FindAccountByID", ctx, accountID).Return(account, nil).Once()
	suite.mockJournalRepo.On("FindTransactionsByAccountID", ctx, accountID).Return(transactions, nil).Once()

	balance, err := suite.service.CalculateAccountBalance(ctx, accountID)

	suite.Require().Error(err)
	suite.Contains(err.Error(), "invalid non-positive transaction amount")
	suite.True(decimal.Zero.Equal(balance))
	suite.mockAccountRepo.AssertExpectations(suite.T())
	suite.mockJournalRepo.AssertExpectations(suite.T())
}

// --- Run Suite ---
func TestJournalService(t *testing.T) {
	suite.Run(t, new(JournalServiceTestSuite))
}
