package services_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/SscSPs/money_managemet_app/internal/apperrors"
	"github.com/SscSPs/money_managemet_app/internal/core/domain"
	"github.com/SscSPs/money_managemet_app/internal/core/services"
	"github.com/SscSPs/money_managemet_app/internal/dto"
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

func (m *MockJournalRepository) FindTransactionsByAccountID(ctx context.Context, workplaceID, accountID string) ([]domain.Transaction, error) {
	args := m.Called(ctx, workplaceID, accountID)
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

// --- Mock WorkplaceService ---
type MockWorkplaceService struct {
	mock.Mock
}

func (m *MockWorkplaceService) AuthorizeUserAction(ctx context.Context, userID, workplaceID string, requiredRole domain.UserWorkplaceRole) error {
	args := m.Called(ctx, userID, workplaceID, requiredRole)
	return args.Error(0)
}

// --- Placeholder implementations for other interface methods ---

func (m *MockWorkplaceService) CreateWorkplace(ctx context.Context, name, description, creatorUserID string) (*domain.Workplace, error) {
	args := m.Called(ctx, name, description, creatorUserID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Workplace), args.Error(1)
}

func (m *MockWorkplaceService) AddUserToWorkplace(ctx context.Context, addingUserID, targetUserID, workplaceID string, role domain.UserWorkplaceRole) error {
	args := m.Called(ctx, addingUserID, targetUserID, workplaceID, role)
	return args.Error(0)
}

func (m *MockWorkplaceService) ListUserWorkplaces(ctx context.Context, userID string) ([]domain.Workplace, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Workplace), args.Error(1)
}

func (m *MockWorkplaceService) FindWorkplaceByID(ctx context.Context, workplaceID string) (*domain.Workplace, error) {
	args := m.Called(ctx, workplaceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Workplace), args.Error(1)
}

// --- End Placeholder implementations ---

// --- Test Suite ---
type JournalServiceTestSuite struct {
	suite.Suite
	mockJournalRepo  *MockJournalRepository
	mockAccountRepo  *MockAccountRepository
	mockWorkplaceSvc *MockWorkplaceService // Added mock workplace service
	service          *services.JournalService
}

func (suite *JournalServiceTestSuite) SetupTest() {
	suite.mockJournalRepo = new(MockJournalRepository)
	suite.mockAccountRepo = new(MockAccountRepository)
	suite.mockWorkplaceSvc = new(MockWorkplaceService) // Initialize mock workplace service
	// Pass all required mocks to the service constructor
	suite.service = services.NewJournalService(suite.mockAccountRepo, suite.mockJournalRepo, suite.mockWorkplaceSvc)
}

// --- Test Cases ---

// Helper function to create a basic valid DTO for CreateJournal tests
func createValidJournalCreateDTO(currency string, accID1, accID2 string, amount decimal.Decimal) dto.CreateJournalRequest {
	return dto.CreateJournalRequest{
		Date:         time.Now(),
		Description:  "Test Journal Create",
		CurrencyCode: currency,
		Transactions: []dto.CreateTransactionRequest{
			{
				AccountID:       accID1,
				Amount:          amount,
				TransactionType: domain.Debit,
			},
			{
				AccountID:       accID2,
				Amount:          amount,
				TransactionType: domain.Credit,
			},
		},
	}
}

// --- CreateJournal Tests (renamed from PersistJournal) ---
func (suite *JournalServiceTestSuite) TestCreateJournal_Success() {
	ctx := context.Background()
	userID := uuid.NewString()
	dummyWorkplaceID := uuid.NewString()
	accID1 := uuid.NewString()
	accID2 := uuid.NewString()
	currency := "USD"
	amount := decimal.NewFromInt(100)
	req := createValidJournalCreateDTO(currency, accID1, accID2, amount)

	// >>> Added: Expect AuthorizeUserAction call (success)
	suite.mockWorkplaceSvc.On("AuthorizeUserAction", ctx, userID, dummyWorkplaceID, domain.RoleMember).Return(nil).Once()

	// Mock account fetching
	accountsMap := map[string]domain.Account{
		accID1: {AccountID: accID1, AccountType: domain.Asset, IsActive: true, CurrencyCode: currency, WorkplaceID: dummyWorkplaceID},
		accID2: {AccountID: accID2, AccountType: domain.Asset, IsActive: true, CurrencyCode: currency, WorkplaceID: dummyWorkplaceID},
	}
	suite.mockAccountRepo.On("FindAccountsByIDs", ctx, mock.AnythingOfType("[]string")).Return(accountsMap, nil).Once()

	// Mock journal saving
	suite.mockJournalRepo.On("SaveJournal", ctx, mock.AnythingOfType("domain.Journal"), mock.AnythingOfType("[]domain.Transaction")).Return(nil).Once().Run(func(args mock.Arguments) {
		jnl := args.Get(1).(domain.Journal)
		txns := args.Get(2).([]domain.Transaction)

		suite.Equal(req.Description, jnl.Description)
		suite.Equal(req.CurrencyCode, jnl.CurrencyCode)
		suite.Equal(domain.Posted, jnl.Status)
		suite.NotEmpty(jnl.JournalID)
		suite.Equal(userID, jnl.CreatedBy)
		suite.Equal(dummyWorkplaceID, jnl.WorkplaceID) // Check workplace ID
		suite.Len(txns, 2)
		for _, txn := range txns {
			suite.Equal(jnl.JournalID, txn.JournalID)
			suite.NotEmpty(txn.TransactionID)
			suite.Equal(amount, txn.Amount)
			suite.Equal(currency, txn.CurrencyCode)
			suite.Equal(userID, txn.CreatedBy)
		}
	})

	// Call CreateJournal
	journal, err := suite.service.CreateJournal(ctx, dummyWorkplaceID, req, userID)

	suite.Require().NoError(err)
	suite.Require().NotNil(journal)
	suite.NotEmpty(journal.JournalID)
	suite.Equal(dummyWorkplaceID, journal.WorkplaceID)
	suite.Equal(req.CurrencyCode, journal.CurrencyCode)
	suite.Equal(domain.Posted, journal.Status)

	suite.mockWorkplaceSvc.AssertExpectations(suite.T())
	suite.mockAccountRepo.AssertExpectations(suite.T())
	suite.mockJournalRepo.AssertExpectations(suite.T())
}

func (suite *JournalServiceTestSuite) TestCreateJournal_LessThanTwoTransactions() {
	ctx := context.Background()
	userID := uuid.NewString()
	dummyWorkplaceID := uuid.NewString()
	req := createValidJournalCreateDTO("USD", "acc1", "acc2", decimal.NewFromInt(10))
	req.Transactions = req.Transactions[:1]

	// >>> Added: Expect AuthorizeUserAction call (success)
	suite.mockWorkplaceSvc.On("AuthorizeUserAction", ctx, userID, dummyWorkplaceID, domain.RoleMember).Return(nil).Once()

	journal, err := suite.service.CreateJournal(ctx, dummyWorkplaceID, req, userID)

	suite.Require().Error(err)
	suite.Nil(journal)
	suite.ErrorIs(err, services.ErrJournalMinEntries)
	suite.mockWorkplaceSvc.AssertExpectations(suite.T())
	suite.mockAccountRepo.AssertNotCalled(suite.T(), "FindAccountsByIDs", mock.Anything, mock.Anything)
}

func (suite *JournalServiceTestSuite) TestCreateJournal_AccountCurrencyMismatch() {
	ctx := context.Background()
	userID := uuid.NewString()
	dummyWorkplaceID := uuid.NewString()
	accID1 := uuid.NewString()
	accID2 := uuid.NewString()
	journalCurrency := "USD"
	account2Currency := "EUR"
	req := createValidJournalCreateDTO(journalCurrency, accID1, accID2, decimal.NewFromInt(10))

	// >>> Added: Expect AuthorizeUserAction call (success)
	suite.mockWorkplaceSvc.On("AuthorizeUserAction", ctx, userID, dummyWorkplaceID, domain.RoleMember).Return(nil).Once()

	accountsMap := map[string]domain.Account{
		accID1: {AccountID: accID1, AccountType: domain.Asset, IsActive: true, CurrencyCode: journalCurrency, WorkplaceID: dummyWorkplaceID},
		accID2: {AccountID: accID2, AccountType: domain.Liability, IsActive: true, CurrencyCode: account2Currency, WorkplaceID: dummyWorkplaceID}, // Mismatched currency
	}
	suite.mockAccountRepo.On("FindAccountsByIDs", ctx, mock.AnythingOfType("[]string")).Return(accountsMap, nil).Once()

	journal, err := suite.service.CreateJournal(ctx, dummyWorkplaceID, req, userID)

	suite.Require().Error(err)
	suite.Nil(journal)
	suite.ErrorIs(err, services.ErrCurrencyMismatch)
	suite.Contains(err.Error(), fmt.Sprintf("account currency %s does not match journal currency %s", account2Currency, journalCurrency))
	suite.mockWorkplaceSvc.AssertExpectations(suite.T())
	suite.mockAccountRepo.AssertExpectations(suite.T())
	suite.mockJournalRepo.AssertNotCalled(suite.T(), "SaveJournal", mock.Anything, mock.Anything, mock.Anything)
}

func (suite *JournalServiceTestSuite) TestCreateJournal_AccountNotFound() {
	ctx := context.Background()
	userID := uuid.NewString()
	dummyWorkplaceID := uuid.NewString()
	accID1 := uuid.NewString()
	accID2 := uuid.NewString()
	req := createValidJournalCreateDTO("CAD", accID1, accID2, decimal.NewFromInt(50))

	// >>> Added: Expect AuthorizeUserAction call (success)
	suite.mockWorkplaceSvc.On("AuthorizeUserAction", ctx, userID, dummyWorkplaceID, domain.RoleMember).Return(nil).Once()

	accountsMap := map[string]domain.Account{
		accID1: {AccountID: accID1, AccountType: domain.Asset, IsActive: true, CurrencyCode: "CAD", WorkplaceID: dummyWorkplaceID},
	}
	suite.mockAccountRepo.On("FindAccountsByIDs", ctx, mock.AnythingOfType("[]string")).Return(accountsMap, nil).Once()

	journal, err := suite.service.CreateJournal(ctx, dummyWorkplaceID, req, userID)

	suite.Require().Error(err)
	suite.Nil(journal)
	suite.ErrorIs(err, services.ErrAccountNotFound)
	suite.Contains(err.Error(), accID2)
	suite.mockWorkplaceSvc.AssertExpectations(suite.T())
	suite.mockAccountRepo.AssertExpectations(suite.T())
	suite.mockJournalRepo.AssertNotCalled(suite.T(), "SaveJournal", mock.Anything, mock.Anything, mock.Anything)
}

func (suite *JournalServiceTestSuite) TestCreateJournal_AccountInactive() {
	ctx := context.Background()
	userID := uuid.NewString()
	dummyWorkplaceID := uuid.NewString()
	accID1 := uuid.NewString()
	accID2 := uuid.NewString()
	req := createValidJournalCreateDTO("EUR", accID1, accID2, decimal.NewFromInt(200))

	// >>> Added: Expect AuthorizeUserAction call (success)
	suite.mockWorkplaceSvc.On("AuthorizeUserAction", ctx, userID, dummyWorkplaceID, domain.RoleMember).Return(nil).Once()

	accountsMap := map[string]domain.Account{
		accID1: {AccountID: accID1, AccountType: domain.Asset, IsActive: true, CurrencyCode: "EUR", WorkplaceID: dummyWorkplaceID},
		accID2: {AccountID: accID2, AccountType: domain.Liability, IsActive: false, CurrencyCode: "EUR", WorkplaceID: dummyWorkplaceID}, // Inactive account
	}
	suite.mockAccountRepo.On("FindAccountsByIDs", ctx, mock.AnythingOfType("[]string")).Return(accountsMap, nil).Once()

	journal, err := suite.service.CreateJournal(ctx, dummyWorkplaceID, req, userID)

	suite.Require().Error(err)
	suite.Nil(journal)
	suite.Contains(err.Error(), "account")
	suite.Contains(err.Error(), accID2)
	suite.Contains(err.Error(), "is inactive")
	suite.mockWorkplaceSvc.AssertExpectations(suite.T())
	suite.mockAccountRepo.AssertExpectations(suite.T())
	suite.mockJournalRepo.AssertNotCalled(suite.T(), "SaveJournal", mock.Anything, mock.Anything, mock.Anything)
}

func (suite *JournalServiceTestSuite) TestCreateJournal_AccountWrongWorkplace() {
	ctx := context.Background()
	userID := uuid.NewString()
	dummyWorkplaceID := uuid.NewString()
	otherWorkplaceID := uuid.NewString()
	accID1 := uuid.NewString()
	accID2 := uuid.NewString()
	req := createValidJournalCreateDTO("GBP", accID1, accID2, decimal.NewFromInt(300))

	// >>> Added: Expect AuthorizeUserAction call (success)
	suite.mockWorkplaceSvc.On("AuthorizeUserAction", ctx, userID, dummyWorkplaceID, domain.RoleMember).Return(nil).Once()

	accountsMap := map[string]domain.Account{
		accID1: {AccountID: accID1, AccountType: domain.Asset, IsActive: true, CurrencyCode: "GBP", WorkplaceID: dummyWorkplaceID},
		accID2: {AccountID: accID2, AccountType: domain.Expense, IsActive: true, CurrencyCode: "GBP", WorkplaceID: otherWorkplaceID}, // Wrong workplace
	}
	suite.mockAccountRepo.On("FindAccountsByIDs", ctx, mock.AnythingOfType("[]string")).Return(accountsMap, nil).Once()

	journal, err := suite.service.CreateJournal(ctx, dummyWorkplaceID, req, userID)

	suite.Require().Error(err)
	suite.Nil(journal)
	suite.ErrorIs(err, services.ErrAccountNotFound) // Treat as not found for security
	suite.Contains(err.Error(), accID2)
	suite.mockWorkplaceSvc.AssertExpectations(suite.T())
	suite.mockAccountRepo.AssertExpectations(suite.T())
	suite.mockJournalRepo.AssertNotCalled(suite.T(), "SaveJournal", mock.Anything, mock.Anything, mock.Anything)
}

func (suite *JournalServiceTestSuite) TestCreateJournal_Unbalanced() {
	ctx := context.Background()
	userID := uuid.NewString()
	dummyWorkplaceID := uuid.NewString()
	accID1 := uuid.NewString()
	accID2 := uuid.NewString()
	req := createValidJournalCreateDTO("USD", accID1, accID2, decimal.NewFromInt(100))
	req.Transactions[1].Amount = decimal.NewFromInt(99)

	// >>> Added: Expect AuthorizeUserAction call (success)
	suite.mockWorkplaceSvc.On("AuthorizeUserAction", ctx, userID, dummyWorkplaceID, domain.RoleMember).Return(nil).Once()

	accountsMap := map[string]domain.Account{
		accID1: {AccountID: accID1, AccountType: domain.Asset, IsActive: true, CurrencyCode: "USD", WorkplaceID: dummyWorkplaceID},
		accID2: {AccountID: accID2, AccountType: domain.Liability, IsActive: true, CurrencyCode: "USD", WorkplaceID: dummyWorkplaceID},
	}
	suite.mockAccountRepo.On("FindAccountsByIDs", ctx, mock.AnythingOfType("[]string")).Return(accountsMap, nil).Once()

	journal, err := suite.service.CreateJournal(ctx, dummyWorkplaceID, req, userID)

	suite.Require().Error(err)
	suite.Nil(journal)
	suite.ErrorIs(err, services.ErrJournalUnbalanced)
	suite.Contains(err.Error(), "sum is 1")
	suite.mockWorkplaceSvc.AssertExpectations(suite.T())
	suite.mockAccountRepo.AssertExpectations(suite.T())
	suite.mockJournalRepo.AssertNotCalled(suite.T(), "SaveJournal", mock.Anything, mock.Anything, mock.Anything)
}

func (suite *JournalServiceTestSuite) TestCreateJournal_FindAccountsError() {
	ctx := context.Background()
	userID := uuid.NewString()
	dummyWorkplaceID := uuid.NewString()
	accID1 := uuid.NewString()
	accID2 := uuid.NewString()
	req := createValidJournalCreateDTO("USD", accID1, accID2, decimal.NewFromInt(100))
	expectedErr := assert.AnError

	// >>> Added: Expect AuthorizeUserAction call (success)
	suite.mockWorkplaceSvc.On("AuthorizeUserAction", ctx, userID, dummyWorkplaceID, domain.RoleMember).Return(nil).Once()

	suite.mockAccountRepo.On("FindAccountsByIDs", ctx, mock.AnythingOfType("[]string")).Return(nil, expectedErr).Once()

	journal, err := suite.service.CreateJournal(ctx, dummyWorkplaceID, req, userID)

	suite.Require().Error(err)
	suite.Nil(journal)
	suite.ErrorIs(err, expectedErr)
	suite.Contains(err.Error(), "failed to fetch accounts")
	suite.mockWorkplaceSvc.AssertExpectations(suite.T())
	suite.mockAccountRepo.AssertExpectations(suite.T())
	suite.mockJournalRepo.AssertNotCalled(suite.T(), "SaveJournal", mock.Anything, mock.Anything, mock.Anything)
}

func (suite *JournalServiceTestSuite) TestCreateJournal_SaveJournalError() {
	ctx := context.Background()
	userID := uuid.NewString()
	dummyWorkplaceID := uuid.NewString()
	accID1 := uuid.NewString()
	accID2 := uuid.NewString()
	req := createValidJournalCreateDTO("USD", accID1, accID2, decimal.NewFromInt(100))
	expectedErr := assert.AnError

	// >>> Added: Expect AuthorizeUserAction call (success)
	suite.mockWorkplaceSvc.On("AuthorizeUserAction", ctx, userID, dummyWorkplaceID, domain.RoleMember).Return(nil).Once()

	accountsMap := map[string]domain.Account{
		accID1: {AccountID: accID1, AccountType: domain.Asset, IsActive: true, CurrencyCode: "USD", WorkplaceID: dummyWorkplaceID},
		accID2: {AccountID: accID2, AccountType: domain.Asset, IsActive: true, CurrencyCode: "USD", WorkplaceID: dummyWorkplaceID},
	}
	suite.mockAccountRepo.On("FindAccountsByIDs", ctx, mock.AnythingOfType("[]string")).Return(accountsMap, nil).Once()

	suite.mockJournalRepo.On("SaveJournal", ctx, mock.AnythingOfType("domain.Journal"), mock.AnythingOfType("[]domain.Transaction")).Return(expectedErr).Once()

	journal, err := suite.service.CreateJournal(ctx, dummyWorkplaceID, req, userID)

	suite.Require().Error(err)
	suite.Nil(journal)
	suite.ErrorIs(err, expectedErr)
	suite.Contains(err.Error(), "failed to save journal")
	suite.mockWorkplaceSvc.AssertExpectations(suite.T())
	suite.mockAccountRepo.AssertExpectations(suite.T())
	suite.mockJournalRepo.AssertExpectations(suite.T())
}

// --- GetJournalByID Tests (renamed from GetJournalWithTransactions) ---
func (suite *JournalServiceTestSuite) TestGetJournalByID_Success() {
	ctx := context.Background()
	journalID := uuid.NewString()
	dummyWorkplaceID := uuid.NewString()
	dummyUserID := uuid.NewString() // Added requesting user ID
	expectedJournal := &domain.Journal{
		JournalID:   journalID,
		WorkplaceID: dummyWorkplaceID,
		Description: "Found Journal",
	}

	// Expect AuthorizeUserAction call (assuming success)
	suite.mockWorkplaceSvc.On("AuthorizeUserAction", ctx, dummyUserID, dummyWorkplaceID, domain.RoleMember).Return(nil).Once()

	// Expect FindJournalByID repo call
	suite.mockJournalRepo.On("FindJournalByID", ctx, journalID).Return(expectedJournal, nil).Once()

	// Call GetJournalByID with workplaceID and requestingUserID
	journal, err := suite.service.GetJournalByID(ctx, dummyWorkplaceID, journalID, dummyUserID)

	suite.Require().NoError(err)
	suite.Equal(expectedJournal, journal)
	suite.mockWorkplaceSvc.AssertExpectations(suite.T())
	suite.mockJournalRepo.AssertExpectations(suite.T())
	suite.mockJournalRepo.AssertNotCalled(suite.T(), "FindTransactionsByJournalID", mock.Anything, mock.Anything)
}

func (suite *JournalServiceTestSuite) TestGetJournalByID_AuthFailed() {
	ctx := context.Background()
	journalID := uuid.NewString()
	dummyWorkplaceID := uuid.NewString()
	dummyUserID := uuid.NewString()
	authErr := apperrors.ErrForbidden // Simulate auth failure

	// Expect AuthorizeUserAction call returning an error
	suite.mockWorkplaceSvc.On("AuthorizeUserAction", ctx, dummyUserID, dummyWorkplaceID, domain.RoleMember).Return(authErr).Once()

	// Call GetJournalByID
	journal, err := suite.service.GetJournalByID(ctx, dummyWorkplaceID, journalID, dummyUserID)

	suite.Require().Error(err)
	suite.Nil(journal)
	suite.ErrorIs(err, authErr)
	suite.mockWorkplaceSvc.AssertExpectations(suite.T())
	// JournalRepo should not be called if auth fails
	suite.mockJournalRepo.AssertNotCalled(suite.T(), "FindJournalByID", mock.Anything, mock.Anything)
}

func (suite *JournalServiceTestSuite) TestGetJournalByID_WrongWorkplace() {
	ctx := context.Background()
	journalID := uuid.NewString()
	correctWorkplaceID := uuid.NewString()
	incorrectWorkplaceID := uuid.NewString()
	dummyUserID := uuid.NewString() // Added requesting user ID
	journalFromRepo := &domain.Journal{
		JournalID:   journalID,
		WorkplaceID: incorrectWorkplaceID, // Belongs to wrong workplace
		Description: "Found Journal, Wrong WP",
	}

	// Expect AuthorizeUserAction call (assuming success for the correct workplace ID)
	suite.mockWorkplaceSvc.On("AuthorizeUserAction", ctx, dummyUserID, correctWorkplaceID, domain.RoleMember).Return(nil).Once()

	// Expect FindJournalByID repo call (returns journal from the *incorrect* workplace)
	suite.mockJournalRepo.On("FindJournalByID", ctx, journalID).Return(journalFromRepo, nil).Once()

	// Call GetJournalByID asking for the *correct* workplaceID and user
	journal, err := suite.service.GetJournalByID(ctx, correctWorkplaceID, journalID, dummyUserID)

	// Assertions: Expect ErrNotFound because the service should filter by workplace mismatch
	suite.Require().Error(err)
	suite.Nil(journal)
	suite.ErrorIs(err, apperrors.ErrNotFound)
	suite.mockWorkplaceSvc.AssertExpectations(suite.T())
	suite.mockJournalRepo.AssertExpectations(suite.T())
	suite.mockJournalRepo.AssertNotCalled(suite.T(), "FindTransactionsByJournalID", mock.Anything, mock.Anything)
}

func (suite *JournalServiceTestSuite) TestGetJournalByID_NotFound() {
	ctx := context.Background()
	journalID := uuid.NewString()
	dummyWorkplaceID := uuid.NewString()
	dummyUserID := uuid.NewString() // Added requesting user ID

	// Expect AuthorizeUserAction call (assuming success)
	suite.mockWorkplaceSvc.On("AuthorizeUserAction", ctx, dummyUserID, dummyWorkplaceID, domain.RoleMember).Return(nil).Once()

	// Expect FindJournalByID repo call returning NotFound
	suite.mockJournalRepo.On("FindJournalByID", ctx, journalID).Return(nil, apperrors.ErrNotFound).Once()

	// Call GetJournalByID with workplaceID and requestingUserID
	journal, err := suite.service.GetJournalByID(ctx, dummyWorkplaceID, journalID, dummyUserID)

	suite.Require().Error(err)
	suite.Nil(journal)
	suite.ErrorIs(err, apperrors.ErrNotFound)
	suite.mockWorkplaceSvc.AssertExpectations(suite.T())
	suite.mockJournalRepo.AssertExpectations(suite.T())
	suite.mockJournalRepo.AssertNotCalled(suite.T(), "FindTransactionsByJournalID", mock.Anything, mock.Anything)
}

func (suite *JournalServiceTestSuite) TestGetJournalByID_FindJournalError() {
	ctx := context.Background()
	journalID := uuid.NewString()
	dummyWorkplaceID := uuid.NewString()
	dummyUserID := uuid.NewString() // Added requesting user ID
	expectedErr := assert.AnError

	// Expect AuthorizeUserAction call (assuming success)
	suite.mockWorkplaceSvc.On("AuthorizeUserAction", ctx, dummyUserID, dummyWorkplaceID, domain.RoleMember).Return(nil).Once()

	// Expect FindJournalByID repo call returning a generic error
	suite.mockJournalRepo.On("FindJournalByID", ctx, journalID).Return(nil, expectedErr).Once()

	// Call GetJournalByID with workplaceID and requestingUserID
	journal, err := suite.service.GetJournalByID(ctx, dummyWorkplaceID, journalID, dummyUserID)

	suite.Require().Error(err)
	suite.Nil(journal)
	suite.ErrorIs(err, expectedErr)
	suite.Contains(err.Error(), "failed to find journal by ID")
	suite.mockWorkplaceSvc.AssertExpectations(suite.T())
	suite.mockJournalRepo.AssertExpectations(suite.T())
	suite.mockJournalRepo.AssertNotCalled(suite.T(), "FindTransactionsByJournalID", mock.Anything, mock.Anything)
}

// --- CalculateAccountBalance Tests (already used workplaceID) ---
// ... (tests remain largely the same, just ensure FindAccountByID mocks include WorkplaceID)

func (suite *JournalServiceTestSuite) TestCalculateAccountBalance_Success_Asset() {
	ctx := context.Background()
	accID := uuid.NewString()
	dummyWorkplaceID := uuid.NewString()
	account := &domain.Account{
		AccountID:   accID,
		AccountType: domain.Asset,
		IsActive:    true,
		WorkplaceID: dummyWorkplaceID, // Ensure WorkplaceID is set in mock
	}
	transactions := []domain.Transaction{
		{AccountID: accID, Amount: decimal.NewFromInt(100), TransactionType: domain.Debit},
		{AccountID: accID, Amount: decimal.NewFromInt(50), TransactionType: domain.Credit},
		{AccountID: accID, Amount: decimal.NewFromInt(25), TransactionType: domain.Debit},
	}

	suite.mockAccountRepo.On("FindAccountByID", ctx, accID).Return(account, nil).Once()
	suite.mockJournalRepo.On("FindTransactionsByAccountID", ctx, dummyWorkplaceID, accID).Return(transactions, nil).Once()

	balance, err := suite.service.CalculateAccountBalance(ctx, dummyWorkplaceID, accID)

	suite.Require().NoError(err)
	suite.Equal("75", balance.String())
	suite.mockAccountRepo.AssertExpectations(suite.T())
	suite.mockJournalRepo.AssertExpectations(suite.T())
}

func (suite *JournalServiceTestSuite) TestCalculateAccountBalance_Success_LiabilityZero() {
	ctx := context.Background()
	accID := uuid.NewString()
	dummyWorkplaceID := uuid.NewString()
	account := &domain.Account{
		AccountID:   accID,
		AccountType: domain.Liability,
		IsActive:    true,
		WorkplaceID: dummyWorkplaceID, // Ensure WorkplaceID is set in mock
	}
	transactions := []domain.Transaction{
		{AccountID: accID, Amount: decimal.NewFromInt(200), TransactionType: domain.Credit},
		{AccountID: accID, Amount: decimal.NewFromInt(200), TransactionType: domain.Debit},
	}

	suite.mockAccountRepo.On("FindAccountByID", ctx, accID).Return(account, nil).Once()
	suite.mockJournalRepo.On("FindTransactionsByAccountID", ctx, dummyWorkplaceID, accID).Return(transactions, nil).Once()

	balance, err := suite.service.CalculateAccountBalance(ctx, dummyWorkplaceID, accID)

	suite.Require().NoError(err)
	suite.True(balance.IsZero())
	suite.mockAccountRepo.AssertExpectations(suite.T())
	suite.mockJournalRepo.AssertExpectations(suite.T())
}

func (suite *JournalServiceTestSuite) TestCalculateAccountBalance_Success_EquityNegative() {
	ctx := context.Background()
	accID := uuid.NewString()
	dummyWorkplaceID := uuid.NewString()
	account := &domain.Account{
		AccountID:   accID,
		AccountType: domain.Equity,
		IsActive:    true,
		WorkplaceID: dummyWorkplaceID, // Ensure WorkplaceID is set in mock
	}
	transactions := []domain.Transaction{
		{AccountID: accID, Amount: decimal.NewFromInt(500), TransactionType: domain.Debit},
		{AccountID: accID, Amount: decimal.NewFromInt(100), TransactionType: domain.Credit},
	}

	suite.mockAccountRepo.On("FindAccountByID", ctx, accID).Return(account, nil).Once()
	suite.mockJournalRepo.On("FindTransactionsByAccountID", ctx, dummyWorkplaceID, accID).Return(transactions, nil).Once()

	balance, err := suite.service.CalculateAccountBalance(ctx, dummyWorkplaceID, accID)

	suite.Require().NoError(err)
	suite.Equal("-400", balance.String())
	suite.mockAccountRepo.AssertExpectations(suite.T())
	suite.mockJournalRepo.AssertExpectations(suite.T())
}

func (suite *JournalServiceTestSuite) TestCalculateAccountBalance_NoTransactions() {
	ctx := context.Background()
	accID := uuid.NewString()
	dummyWorkplaceID := uuid.NewString()
	account := &domain.Account{AccountID: accID, AccountType: domain.Asset, IsActive: true, WorkplaceID: dummyWorkplaceID}
	transactions := []domain.Transaction{}

	suite.mockAccountRepo.On("FindAccountByID", ctx, accID).Return(account, nil).Once()
	suite.mockJournalRepo.On("FindTransactionsByAccountID", ctx, dummyWorkplaceID, accID).Return(transactions, nil).Once()

	balance, err := suite.service.CalculateAccountBalance(ctx, dummyWorkplaceID, accID)

	suite.Require().NoError(err)
	suite.True(balance.IsZero())
	suite.mockAccountRepo.AssertExpectations(suite.T())
	suite.mockJournalRepo.AssertExpectations(suite.T())
}

func (suite *JournalServiceTestSuite) TestCalculateAccountBalance_AccountNotFound() {
	ctx := context.Background()
	accID := uuid.NewString()
	dummyWorkplaceID := uuid.NewString()

	suite.mockAccountRepo.On("FindAccountByID", ctx, accID).Return(nil, apperrors.ErrNotFound).Once()

	balance, err := suite.service.CalculateAccountBalance(ctx, dummyWorkplaceID, accID)

	suite.Require().Error(err)
	suite.ErrorIs(err, services.ErrAccountNotFound)
	suite.True(balance.IsZero())
	suite.mockAccountRepo.AssertExpectations(suite.T())
	suite.mockJournalRepo.AssertNotCalled(suite.T(), "FindTransactionsByAccountID", mock.Anything, mock.Anything, mock.Anything)
}

func (suite *JournalServiceTestSuite) TestCalculateAccountBalance_AccountWrongWorkplace() {
	ctx := context.Background()
	accID := uuid.NewString()
	correctWorkplaceID := uuid.NewString()
	incorrectWorkplaceID := uuid.NewString()
	account := &domain.Account{
		AccountID:   accID,
		AccountType: domain.Asset,
		IsActive:    true,
		WorkplaceID: incorrectWorkplaceID, // Account belongs to wrong workplace
	}

	// FindAccountByID returns the account (even if wrong workplace)
	suite.mockAccountRepo.On("FindAccountByID", ctx, accID).Return(account, nil).Once()

	// Call CalculateAccountBalance asking for the *correct* workplace
	balance, err := suite.service.CalculateAccountBalance(ctx, correctWorkplaceID, accID)

	// Service should detect the mismatch and return an error (e.g., NotFound or Forbidden)
	suite.Require().Error(err)
	suite.True(balance.IsZero())
	suite.Contains(err.Error(), "not found in workplace") // Or check for ErrForbidden if that's the chosen behavior
	suite.mockAccountRepo.AssertExpectations(suite.T())
	// Should not attempt to fetch transactions if account workplace doesn't match
	suite.mockJournalRepo.AssertNotCalled(suite.T(), "FindTransactionsByAccountID", mock.Anything, mock.Anything, mock.Anything)
}

func (suite *JournalServiceTestSuite) TestCalculateAccountBalance_AccountInactive() {
	ctx := context.Background()
	accID := uuid.NewString()
	dummyWorkplaceID := uuid.NewString()
	account := &domain.Account{AccountID: accID, AccountType: domain.Asset, IsActive: false, WorkplaceID: dummyWorkplaceID}

	suite.mockAccountRepo.On("FindAccountByID", ctx, accID).Return(account, nil).Once()

	balance, err := suite.service.CalculateAccountBalance(ctx, dummyWorkplaceID, accID)

	suite.Require().Error(err)
	suite.Contains(err.Error(), "inactive")
	suite.True(balance.IsZero())
	suite.mockAccountRepo.AssertExpectations(suite.T())
	suite.mockJournalRepo.AssertNotCalled(suite.T(), "FindTransactionsByAccountID", mock.Anything, mock.Anything, mock.Anything)
}

func (suite *JournalServiceTestSuite) TestCalculateAccountBalance_FindAccountError() {
	ctx := context.Background()
	accID := uuid.NewString()
	dummyWorkplaceID := uuid.NewString()
	expectedErr := assert.AnError

	suite.mockAccountRepo.On("FindAccountByID", ctx, accID).Return(nil, expectedErr).Once()

	balance, err := suite.service.CalculateAccountBalance(ctx, dummyWorkplaceID, accID)

	suite.Require().Error(err)
	suite.ErrorIs(err, expectedErr)
	suite.True(balance.IsZero())
	suite.mockAccountRepo.AssertExpectations(suite.T())
	suite.mockJournalRepo.AssertNotCalled(suite.T(), "FindTransactionsByAccountID", mock.Anything, mock.Anything, mock.Anything)
}

func (suite *JournalServiceTestSuite) TestCalculateAccountBalance_FindTransactionsError() {
	ctx := context.Background()
	accID := uuid.NewString()
	dummyWorkplaceID := uuid.NewString()
	account := &domain.Account{AccountID: accID, AccountType: domain.Asset, IsActive: true, WorkplaceID: dummyWorkplaceID}
	expectedErr := assert.AnError

	suite.mockAccountRepo.On("FindAccountByID", ctx, accID).Return(account, nil).Once()
	suite.mockJournalRepo.On("FindTransactionsByAccountID", ctx, dummyWorkplaceID, accID).Return(nil, expectedErr).Once()

	balance, err := suite.service.CalculateAccountBalance(ctx, dummyWorkplaceID, accID)

	suite.Require().Error(err)
	suite.ErrorIs(err, expectedErr)
	suite.True(balance.IsZero())
	suite.mockAccountRepo.AssertExpectations(suite.T())
	suite.mockJournalRepo.AssertExpectations(suite.T())
}

func (suite *JournalServiceTestSuite) TestCalculateAccountBalance_InvalidTransactionAmount() {
	ctx := context.Background()
	accID := uuid.NewString()
	dummyWorkplaceID := uuid.NewString()
	account := &domain.Account{AccountID: accID, AccountType: domain.Asset, IsActive: true, WorkplaceID: dummyWorkplaceID}
	transactions := []domain.Transaction{
		{AccountID: accID, Amount: decimal.NewFromInt(-10), TransactionType: domain.Debit},
	}

	suite.mockAccountRepo.On("FindAccountByID", ctx, accID).Return(account, nil).Once()
	suite.mockJournalRepo.On("FindTransactionsByAccountID", ctx, dummyWorkplaceID, accID).Return(transactions, nil).Once()

	balance, err := suite.service.CalculateAccountBalance(ctx, dummyWorkplaceID, accID)

	suite.Require().Error(err)
	suite.Contains(err.Error(), "invalid non-positive transaction amount")
	suite.True(balance.IsZero())
	suite.mockAccountRepo.AssertExpectations(suite.T())
	suite.mockJournalRepo.AssertExpectations(suite.T())
}

// --- Run Test Suite ---
func TestJournalService(t *testing.T) {
	suite.Run(t, new(JournalServiceTestSuite))
}
