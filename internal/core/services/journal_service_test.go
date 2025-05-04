package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/SscSPs/money_managemet_app/internal/apperrors"
	"github.com/SscSPs/money_managemet_app/internal/core/domain"
	portsrepo "github.com/SscSPs/money_managemet_app/internal/core/ports/repositories"
	portssvc "github.com/SscSPs/money_managemet_app/internal/core/ports/services"
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

// Ensure MockJournalRepository implements portsrepo.JournalRepositoryFacade
var _ portsrepo.JournalRepositoryFacade = (*MockJournalRepository)(nil)

func (m *MockJournalRepository) SaveJournal(ctx context.Context, journal domain.Journal, transactions []domain.Transaction, balanceChanges map[string]decimal.Decimal) error {
	args := m.Called(ctx, journal, transactions, balanceChanges)
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

func (m *MockJournalRepository) ListJournalsByWorkplace(ctx context.Context, workplaceID string, limit int, nextToken *string, includeReversals bool) ([]domain.Journal, *string, error) {
	args := m.Called(ctx, workplaceID, limit, nextToken, includeReversals)
	if args.Get(0) == nil {
		return nil, nil, args.Error(2)
	}
	var returnedNextToken *string
	if args.Get(1) != nil {
		tokenVal := args.Get(1).(string)
		returnedNextToken = &tokenVal
	}
	return args.Get(0).([]domain.Journal), returnedNextToken, args.Error(2)
}

func (m *MockJournalRepository) FindTransactionsByJournalIDs(ctx context.Context, journalIDs []string) (map[string][]domain.Transaction, error) {
	args := m.Called(ctx, journalIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string][]domain.Transaction), args.Error(1)
}

func (m *MockJournalRepository) UpdateJournal(ctx context.Context, journal domain.Journal) error {
	args := m.Called(ctx, journal)
	return args.Error(0)
}

func (m *MockJournalRepository) UpdateJournalStatusAndLinks(ctx context.Context, journalID string, status domain.JournalStatus, reversingJournalID *string, originalJournalID *string, updatedByUserID string, updatedAt time.Time) error {
	args := m.Called(ctx, journalID, status, reversingJournalID, originalJournalID, updatedByUserID, updatedAt)
	return args.Error(0)
}

func (m *MockJournalRepository) ListTransactionsByAccountID(ctx context.Context, workplaceID, accountID string, limit int, nextToken *string) ([]domain.Transaction, *string, error) {
	args := m.Called(ctx, workplaceID, accountID, limit, nextToken)
	if args.Get(0) == nil {
		return nil, nil, args.Error(2)
	}
	var returnedNextToken *string
	if args.Get(1) != nil {
		tokenVal := args.Get(1).(string)
		returnedNextToken = &tokenVal
	}
	return args.Get(0).([]domain.Transaction), returnedNextToken, args.Error(2)
}

// --- Mock AccountService (as used by JournalService) ---
type MockAccountService2 struct {
	mock.Mock
}

var _ portssvc.AccountService = (*MockAccountService2)(nil)

func (m *MockAccountService2) CreateAccount(ctx context.Context, workplaceID string, req dto.CreateAccountRequest, userID string) (*domain.Account, error) {
	args := m.Called(ctx, workplaceID, req, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Account), args.Error(1)
}

func (m *MockAccountService2) GetAccountByID(ctx context.Context, workplaceID string, accountID string) (*domain.Account, error) {
	args := m.Called(ctx, workplaceID, accountID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Account), args.Error(1)
}

func (m *MockAccountService2) GetAccountByIDs(ctx context.Context, workplaceID string, accountIDs []string) (map[string]domain.Account, error) {
	args := m.Called(ctx, workplaceID, accountIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]domain.Account), args.Error(1)
}

func (m *MockAccountService2) ListAccounts(ctx context.Context, workplaceID string, limit int, offset int) ([]domain.Account, error) {
	args := m.Called(ctx, workplaceID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Account), args.Error(1)
}

func (m *MockAccountService2) UpdateAccount(ctx context.Context, workplaceID string, accountID string, req dto.UpdateAccountRequest, userID string) (*domain.Account, error) {
	args := m.Called(ctx, workplaceID, accountID, req, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Account), args.Error(1)
}

func (m *MockAccountService2) DeactivateAccount(ctx context.Context, workplaceID string, accountID string, userID string) error {
	args := m.Called(ctx, workplaceID, accountID, userID)
	return args.Error(0)
}

func (m *MockAccountService2) CalculateAccountBalance(ctx context.Context, workplaceID string, accountID string) (decimal.Decimal, error) {
	args := m.Called(ctx, workplaceID, accountID)
	if args.Get(0) == nil {
		return decimal.Zero, args.Error(1)
	}
	return args.Get(0).(decimal.Decimal), args.Error(1)
}

// --- Mock WorkplaceService ---
type MockWorkplaceService struct {
	mock.Mock
}

// Ensure MockWorkplaceService implements the full interface
var _ portssvc.WorkplaceService = (*MockWorkplaceService)(nil)

func (m *MockWorkplaceService) CreateWorkplace(ctx context.Context, name, description, defaultCurrencyCode, creatorUserID string) (*domain.Workplace, error) {
	args := m.Called(ctx, name, description, defaultCurrencyCode, creatorUserID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Workplace), args.Error(1)
}

func (m *MockWorkplaceService) AddUserToWorkplace(ctx context.Context, addingUserID, targetUserID, workplaceID string, role domain.UserWorkplaceRole) error {
	args := m.Called(ctx, addingUserID, targetUserID, workplaceID, role)
	return args.Error(0)
}

func (m *MockWorkplaceService) ListUserWorkplaces(ctx context.Context, userID string, includeDisabled bool) ([]domain.Workplace, error) {
	args := m.Called(ctx, userID, includeDisabled)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Workplace), args.Error(1)
}

// Updated AuthorizeUserAction mock signature
func (m *MockWorkplaceService) AuthorizeUserAction(ctx context.Context, userID string, workplaceID string, requiredRole domain.UserWorkplaceRole) error {
	args := m.Called(ctx, userID, workplaceID, requiredRole)
	return args.Error(0)
}

func (m *MockWorkplaceService) FindWorkplaceByID(ctx context.Context, workplaceID string) (*domain.Workplace, error) {
	args := m.Called(ctx, workplaceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Workplace), args.Error(1)
}

func (m *MockWorkplaceService) DeactivateWorkplace(ctx context.Context, workplaceID string, requestingUserID string) error {
	args := m.Called(ctx, workplaceID, requestingUserID)
	return args.Error(0)
}

func (m *MockWorkplaceService) ActivateWorkplace(ctx context.Context, workplaceID string, requestingUserID string) error {
	args := m.Called(ctx, workplaceID, requestingUserID)
	return args.Error(0)
}

// Add ListWorkplaceUsers method to the mock
func (m *MockWorkplaceService) ListWorkplaceUsers(ctx context.Context, workplaceID string, requestingUserID string) ([]domain.UserWorkplace, error) {
	args := m.Called(ctx, workplaceID, requestingUserID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.UserWorkplace), args.Error(1)
}

// Add RemoveUserFromWorkplace method to the mock
func (m *MockWorkplaceService) RemoveUserFromWorkplace(ctx context.Context, requestingUserID, targetUserID, workplaceID string) error {
	args := m.Called(ctx, requestingUserID, targetUserID, workplaceID)
	return args.Error(0)
}

// Add UpdateUserWorkplaceRole method to the mock
func (m *MockWorkplaceService) UpdateUserWorkplaceRole(ctx context.Context, requestingUserID, targetUserID, workplaceID string, newRole domain.UserWorkplaceRole) error {
	args := m.Called(ctx, requestingUserID, targetUserID, workplaceID, newRole)
	return args.Error(0)
}

// --- Test Suite Setup ---
type JournalServiceTestSuite struct {
	suite.Suite
	mockJournalRepo  *MockJournalRepository
	mockAccountSvc   *MockAccountService2
	mockWorkplaceSvc *MockWorkplaceService
	service          portssvc.JournalService
	assetAccount     domain.Account
	liabilityAccount domain.Account
	incomeAccount    domain.Account
	expenseAccount   domain.Account
	workplaceID      string
	userID           string
}

func (suite *JournalServiceTestSuite) SetupTest() {
	suite.mockJournalRepo = new(MockJournalRepository)
	suite.mockAccountSvc = new(MockAccountService2)
	suite.mockWorkplaceSvc = new(MockWorkplaceService)
	suite.service = services.NewJournalService(suite.mockJournalRepo, suite.mockAccountSvc, suite.mockWorkplaceSvc)

	suite.workplaceID = uuid.NewString()
	suite.userID = uuid.NewString()

	suite.assetAccount = domain.Account{
		AccountID:    uuid.NewString(),
		WorkplaceID:  suite.workplaceID,
		AccountType:  domain.Asset,
		CurrencyCode: "USD",
		IsActive:     true,
	}
	suite.liabilityAccount = domain.Account{
		AccountID:    uuid.NewString(),
		WorkplaceID:  suite.workplaceID,
		AccountType:  domain.Liability,
		CurrencyCode: "USD",
		IsActive:     true,
	}
	suite.incomeAccount = domain.Account{
		AccountID:    uuid.NewString(),
		WorkplaceID:  suite.workplaceID,
		AccountType:  domain.Revenue,
		CurrencyCode: "USD",
		IsActive:     true,
	}
	suite.expenseAccount = domain.Account{
		AccountID:    uuid.NewString(),
		WorkplaceID:  suite.workplaceID,
		AccountType:  domain.Expense,
		CurrencyCode: "USD",
		IsActive:     true,
	}
}

// --- Test Cases ---

func (suite *JournalServiceTestSuite) TestCreateJournal_Success() {
	ctx := context.Background()
	req := dto.CreateJournalRequest{
		Date:         time.Now(),
		Description:  "Test Journal Success",
		CurrencyCode: "USD",
		Transactions: []dto.CreateTransactionRequest{
			{AccountID: suite.assetAccount.AccountID, Amount: decimal.NewFromInt(100), TransactionType: domain.Debit},      // Debit Asset
			{AccountID: suite.liabilityAccount.AccountID, Amount: decimal.NewFromInt(100), TransactionType: domain.Credit}, // Credit Liability
		},
	}

	// Mock authorization
	suite.mockWorkplaceSvc.On("AuthorizeUserAction", ctx, suite.userID, suite.workplaceID, domain.RoleMember).Return(nil).Once()

	// Mock finding accounts
	accountsMap := map[string]domain.Account{
		suite.assetAccount.AccountID:     suite.assetAccount,
		suite.liabilityAccount.AccountID: suite.liabilityAccount, // Use liability account
	}
	suite.mockAccountSvc.On("GetAccountByIDs", ctx, suite.workplaceID, []string{suite.assetAccount.AccountID, suite.liabilityAccount.AccountID}).Return(accountsMap, nil).Once()

	// Mock saving journal
	suite.mockJournalRepo.On("SaveJournal", ctx, mock.AnythingOfType("domain.Journal"), mock.AnythingOfType("[]domain.Transaction"), mock.AnythingOfType("map[string]decimal.Decimal")).Return(nil).Once()

	createdJournal, err := suite.service.CreateJournal(ctx, suite.workplaceID, req, suite.userID)

	suite.Require().NoError(err)
	suite.Require().NotNil(createdJournal)
	suite.NotEmpty(createdJournal.JournalID)
	suite.Equal(suite.workplaceID, createdJournal.WorkplaceID)
	suite.Equal(req.Description, createdJournal.Description)
	suite.Equal(domain.Posted, createdJournal.Status)
	suite.Equal(suite.userID, createdJournal.CreatedBy)
	suite.Nil(createdJournal.Transactions) // Service should return journal without transactions populated

	suite.mockWorkplaceSvc.AssertExpectations(suite.T())
	suite.mockAccountSvc.AssertExpectations(suite.T())
	suite.mockJournalRepo.AssertExpectations(suite.T())
}

func (suite *JournalServiceTestSuite) TestCreateJournal_AuthorizationFail() {
	ctx := context.Background()
	req := dto.CreateJournalRequest{ /* ... */ }
	authErr := apperrors.ErrForbidden

	suite.mockWorkplaceSvc.On("AuthorizeUserAction", ctx, suite.userID, suite.workplaceID, domain.RoleMember).Return(authErr).Once()

	_, err := suite.service.CreateJournal(ctx, suite.workplaceID, req, suite.userID)

	suite.Require().Error(err)
	suite.ErrorIs(err, authErr)
	suite.mockWorkplaceSvc.AssertExpectations(suite.T())
	suite.mockAccountSvc.AssertNotCalled(suite.T(), "GetAccountByIDs", mock.Anything, mock.Anything, mock.Anything)
	suite.mockJournalRepo.AssertNotCalled(suite.T(), "SaveJournal", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func (suite *JournalServiceTestSuite) TestCreateJournal_LessThanTwoTransactions() {
	ctx := context.Background()
	req := dto.CreateJournalRequest{
		Transactions: []dto.CreateTransactionRequest{
			{AccountID: suite.assetAccount.AccountID, Amount: decimal.NewFromInt(100), TransactionType: domain.Debit},
		},
	}
	suite.mockWorkplaceSvc.On("AuthorizeUserAction", ctx, suite.userID, suite.workplaceID, domain.RoleMember).Return(nil).Once()

	_, err := suite.service.CreateJournal(ctx, suite.workplaceID, req, suite.userID)

	suite.Require().Error(err)
	suite.ErrorIs(err, services.ErrJournalMinEntries)
	suite.mockWorkplaceSvc.AssertExpectations(suite.T())
}

func (suite *JournalServiceTestSuite) TestCreateJournal_NonPositiveAmount() {
	ctx := context.Background()
	req := dto.CreateJournalRequest{
		Transactions: []dto.CreateTransactionRequest{
			{AccountID: suite.assetAccount.AccountID, Amount: decimal.NewFromInt(100), TransactionType: domain.Debit},
			{AccountID: suite.incomeAccount.AccountID, Amount: decimal.NewFromInt(0), TransactionType: domain.Credit}, // Zero amount
		},
	}
	suite.mockWorkplaceSvc.On("AuthorizeUserAction", ctx, suite.userID, suite.workplaceID, domain.RoleMember).Return(nil).Once()

	_, err := suite.service.CreateJournal(ctx, suite.workplaceID, req, suite.userID)

	suite.Require().Error(err)
	suite.ErrorIs(err, apperrors.ErrValidation) // Should be validation error
	suite.mockWorkplaceSvc.AssertExpectations(suite.T())
}

func (suite *JournalServiceTestSuite) TestCreateJournal_FindAccountsError() {
	ctx := context.Background()
	req := dto.CreateJournalRequest{
		Transactions: []dto.CreateTransactionRequest{
			{AccountID: suite.assetAccount.AccountID, Amount: decimal.NewFromInt(100), TransactionType: domain.Debit},
			{AccountID: suite.incomeAccount.AccountID, Amount: decimal.NewFromInt(100), TransactionType: domain.Credit},
		},
	}
	repoErr := assert.AnError
	suite.mockWorkplaceSvc.On("AuthorizeUserAction", ctx, suite.userID, suite.workplaceID, domain.RoleMember).Return(nil).Once()
	suite.mockAccountSvc.On("GetAccountByIDs", ctx, suite.workplaceID, mock.Anything).Return(nil, repoErr).Once()

	_, err := suite.service.CreateJournal(ctx, suite.workplaceID, req, suite.userID)

	suite.Require().Error(err)
	suite.Contains(err.Error(), repoErr.Error())
	suite.mockWorkplaceSvc.AssertExpectations(suite.T())
	suite.mockAccountSvc.AssertExpectations(suite.T())
}

func (suite *JournalServiceTestSuite) TestCreateJournal_AccountNotFound() {
	ctx := context.Background()
	unknownAccountID := uuid.NewString()
	req := dto.CreateJournalRequest{
		CurrencyCode: "USD",
		Transactions: []dto.CreateTransactionRequest{
			{AccountID: suite.assetAccount.AccountID, Amount: decimal.NewFromInt(100), TransactionType: domain.Debit},
			{AccountID: unknownAccountID, Amount: decimal.NewFromInt(100), TransactionType: domain.Credit},
		},
	}
	accountsMap := map[string]domain.Account{
		suite.assetAccount.AccountID: suite.assetAccount,
		// unknownAccountID is missing
	}
	suite.mockWorkplaceSvc.On("AuthorizeUserAction", ctx, suite.userID, suite.workplaceID, domain.RoleMember).Return(nil).Once()
	suite.mockAccountSvc.On("GetAccountByIDs", ctx, suite.workplaceID, mock.Anything).Return(accountsMap, nil).Once()

	_, err := suite.service.CreateJournal(ctx, suite.workplaceID, req, suite.userID)

	suite.Require().Error(err)
	suite.ErrorIs(err, services.ErrAccountNotFound)
	suite.mockWorkplaceSvc.AssertExpectations(suite.T())
	suite.mockAccountSvc.AssertExpectations(suite.T())
}

func (suite *JournalServiceTestSuite) TestCreateJournal_AccountWrongWorkplace() {
	ctx := context.Background()
	wrongWorkplaceAccount := domain.Account{
		AccountID:    uuid.NewString(),
		WorkplaceID:  uuid.NewString(), // Different workplace
		AccountType:  domain.Expense,
		CurrencyCode: "USD",
		IsActive:     true,
	}
	req := dto.CreateJournalRequest{
		CurrencyCode: "USD",
		Transactions: []dto.CreateTransactionRequest{
			{AccountID: suite.assetAccount.AccountID, Amount: decimal.NewFromInt(50), TransactionType: domain.Debit},
			{AccountID: wrongWorkplaceAccount.AccountID, Amount: decimal.NewFromInt(50), TransactionType: domain.Credit},
		},
	}
	accountsMap := map[string]domain.Account{
		suite.assetAccount.AccountID:    suite.assetAccount,
		wrongWorkplaceAccount.AccountID: wrongWorkplaceAccount,
	}
	suite.mockWorkplaceSvc.On("AuthorizeUserAction", ctx, suite.userID, suite.workplaceID, domain.RoleMember).Return(nil).Once()
	suite.mockAccountSvc.On("GetAccountByIDs", ctx, suite.workplaceID, mock.Anything).Return(accountsMap, nil).Once()

	_, err := suite.service.CreateJournal(ctx, suite.workplaceID, req, suite.userID)

	suite.Require().Error(err)
	suite.ErrorIs(err, services.ErrAccountNotFound) // Should be treated as not found in this workplace
	suite.mockWorkplaceSvc.AssertExpectations(suite.T())
	suite.mockAccountSvc.AssertExpectations(suite.T())
}

func (suite *JournalServiceTestSuite) TestCreateJournal_AccountInactive() {
	ctx := context.Background()
	inactiveAccount := domain.Account{
		AccountID:    uuid.NewString(),
		WorkplaceID:  suite.workplaceID,
		AccountType:  domain.Expense,
		CurrencyCode: "USD",
		IsActive:     false, // Inactive
	}
	req := dto.CreateJournalRequest{
		CurrencyCode: "USD",
		Transactions: []dto.CreateTransactionRequest{
			{AccountID: suite.assetAccount.AccountID, Amount: decimal.NewFromInt(50), TransactionType: domain.Debit},
			{AccountID: inactiveAccount.AccountID, Amount: decimal.NewFromInt(50), TransactionType: domain.Credit},
		},
	}
	accountsMap := map[string]domain.Account{
		suite.assetAccount.AccountID: suite.assetAccount,
		inactiveAccount.AccountID:    inactiveAccount,
	}
	suite.mockWorkplaceSvc.On("AuthorizeUserAction", ctx, suite.userID, suite.workplaceID, domain.RoleMember).Return(nil).Once()
	suite.mockAccountSvc.On("GetAccountByIDs", ctx, suite.workplaceID, mock.Anything).Return(accountsMap, nil).Once()

	_, err := suite.service.CreateJournal(ctx, suite.workplaceID, req, suite.userID)

	suite.Require().Error(err)
	suite.ErrorIs(err, apperrors.ErrValidation)
	suite.mockWorkplaceSvc.AssertExpectations(suite.T())
	suite.mockAccountSvc.AssertExpectations(suite.T())
}

func (suite *JournalServiceTestSuite) TestCreateJournal_CurrencyMismatch() {
	ctx := context.Background()
	mismatchCurrencyAccount := domain.Account{
		AccountID:    uuid.NewString(),
		WorkplaceID:  suite.workplaceID,
		AccountType:  domain.Expense,
		CurrencyCode: "EUR", // Different currency
		IsActive:     true,
	}
	req := dto.CreateJournalRequest{
		CurrencyCode: "USD", // Journal currency
		Transactions: []dto.CreateTransactionRequest{
			{AccountID: suite.assetAccount.AccountID, Amount: decimal.NewFromInt(50), TransactionType: domain.Debit},
			{AccountID: mismatchCurrencyAccount.AccountID, Amount: decimal.NewFromInt(50), TransactionType: domain.Credit},
		},
	}
	accountsMap := map[string]domain.Account{
		suite.assetAccount.AccountID:      suite.assetAccount,
		mismatchCurrencyAccount.AccountID: mismatchCurrencyAccount,
	}
	suite.mockWorkplaceSvc.On("AuthorizeUserAction", ctx, suite.userID, suite.workplaceID, domain.RoleMember).Return(nil).Once()
	suite.mockAccountSvc.On("GetAccountByIDs", ctx, suite.workplaceID, mock.Anything).Return(accountsMap, nil).Once()

	_, err := suite.service.CreateJournal(ctx, suite.workplaceID, req, suite.userID)

	suite.Require().Error(err)
	suite.ErrorIs(err, services.ErrCurrencyMismatch)
	suite.mockWorkplaceSvc.AssertExpectations(suite.T())
	suite.mockAccountSvc.AssertExpectations(suite.T())
}

func (suite *JournalServiceTestSuite) TestCreateJournal_Unbalanced() {
	ctx := context.Background()
	req := dto.CreateJournalRequest{
		CurrencyCode: "USD",
		Transactions: []dto.CreateTransactionRequest{
			{AccountID: suite.assetAccount.AccountID, Amount: decimal.NewFromInt(100), TransactionType: domain.Debit},
			{AccountID: suite.incomeAccount.AccountID, Amount: decimal.NewFromInt(99), TransactionType: domain.Credit}, // Unbalanced
		},
	}
	accountsMap := map[string]domain.Account{
		suite.assetAccount.AccountID:  suite.assetAccount,
		suite.incomeAccount.AccountID: suite.incomeAccount,
	}
	suite.mockWorkplaceSvc.On("AuthorizeUserAction", ctx, suite.userID, suite.workplaceID, domain.RoleMember).Return(nil).Once()
	suite.mockAccountSvc.On("GetAccountByIDs", ctx, suite.workplaceID, mock.Anything).Return(accountsMap, nil).Once()

	_, err := suite.service.CreateJournal(ctx, suite.workplaceID, req, suite.userID)

	suite.Require().Error(err)
	suite.ErrorIs(err, services.ErrJournalUnbalanced)
	suite.mockWorkplaceSvc.AssertExpectations(suite.T())
	suite.mockAccountSvc.AssertExpectations(suite.T())
}

func (suite *JournalServiceTestSuite) TestCreateJournal_SaveError() {
	ctx := context.Background()
	req := dto.CreateJournalRequest{
		CurrencyCode: "USD",
		Date:         time.Now(),
		Transactions: []dto.CreateTransactionRequest{
			{AccountID: suite.assetAccount.AccountID, Amount: decimal.NewFromInt(100), TransactionType: domain.Debit},
			{AccountID: suite.liabilityAccount.AccountID, Amount: decimal.NewFromInt(100), TransactionType: domain.Credit},
		},
	}
	accountsMap := map[string]domain.Account{
		suite.assetAccount.AccountID:     suite.assetAccount,
		suite.liabilityAccount.AccountID: suite.liabilityAccount,
	}
	repoErr := assert.AnError
	suite.mockWorkplaceSvc.On("AuthorizeUserAction", ctx, suite.userID, suite.workplaceID, domain.RoleMember).Return(nil).Once()
	suite.mockAccountSvc.On("GetAccountByIDs", ctx, suite.workplaceID, mock.Anything).Return(accountsMap, nil).Once()
	// Expect SaveJournal AFTER successful validation
	suite.mockJournalRepo.On("SaveJournal", ctx, mock.Anything, mock.Anything, mock.Anything).Return(repoErr).Once()

	_, err := suite.service.CreateJournal(ctx, suite.workplaceID, req, suite.userID)

	suite.Require().Error(err)
	// Now the error should be the one returned by SaveJournal
	suite.Contains(err.Error(), repoErr.Error())
	suite.mockWorkplaceSvc.AssertExpectations(suite.T())
	suite.mockAccountSvc.AssertExpectations(suite.T())
	suite.mockJournalRepo.AssertExpectations(suite.T())
}

// TODO: Add tests for GetJournalByID, ListJournals, UpdateJournal, DeactivateJournal, ListTransactionsByAccount, CalculateAccountBalance

// --- Run Test Suite ---
func TestJournalService(t *testing.T) {
	suite.Run(t, new(JournalServiceTestSuite))
}
