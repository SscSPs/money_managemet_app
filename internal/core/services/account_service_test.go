package services_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/SscSPs/money_managemet_app/internal/apperrors"
	"github.com/SscSPs/money_managemet_app/internal/core/domain"
	portssvc "github.com/SscSPs/money_managemet_app/internal/core/ports/services"
	"github.com/SscSPs/money_managemet_app/internal/core/services"
	"github.com/SscSPs/money_managemet_app/internal/dto"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// MockAccountRepositoryFacade is a mock type for the AccountRepositoryFacade interface
// --- Mock AccountRepositoryFacade ---
type MockAccountRepositoryFacade struct {
	mock.Mock
}

func (m *MockAccountRepositoryFacade) SaveAccount(ctx context.Context, account domain.Account) error {
	args := m.Called(ctx, account)
	return args.Error(0)
}

func (m *MockAccountRepositoryFacade) FindAccountByID(ctx context.Context, accountID string) (*domain.Account, error) {
	args := m.Called(ctx, accountID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Account), args.Error(1)
}

func (m *MockAccountRepositoryFacade) FindAccountByCFID(ctx context.Context, cfid string, workplaceID string) (*domain.Account, error) {
	args := m.Called(ctx, cfid, workplaceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Account), args.Error(1)
}

func (m *MockAccountRepositoryFacade) FindAccountsByIDs(ctx context.Context, accountIDs []string) (map[string]domain.Account, error) {
	args := m.Called(ctx, accountIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]domain.Account), args.Error(1)
}

// Mock for FindAccountsByIDsForUpdate
func (m *MockAccountRepositoryFacade) FindAccountsByIDsForUpdate(ctx context.Context, tx pgx.Tx, accountIDs []string) (map[string]domain.Account, error) {
	// Note: Mocking the pgx.Tx might be tricky or unnecessary depending on test focus.
	// Often, you might assert this is called but not deeply mock the transaction itself.
	// Passing mock.Anything for tx might be suitable in many cases.
	args := m.Called(ctx, tx, accountIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]domain.Account), args.Error(1)
}

func (m *MockAccountRepositoryFacade) ListAccounts(ctx context.Context, workplaceID string, limit int, offset int) ([]domain.Account, error) {
	args := m.Called(ctx, workplaceID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Account), args.Error(1)
}

func (m *MockAccountRepositoryFacade) UpdateAccount(ctx context.Context, account domain.Account) error {
	args := m.Called(ctx, account)
	return args.Error(0)
}

// Mock for UpdateAccountBalancesInTx
func (m *MockAccountRepositoryFacade) UpdateAccountBalancesInTx(ctx context.Context, tx pgx.Tx, balanceChanges map[string]decimal.Decimal, userID string, now time.Time) error {
	args := m.Called(ctx, tx, balanceChanges, userID, now)
	return args.Error(0)
}

func (m *MockAccountRepositoryFacade) DeactivateAccount(ctx context.Context, accountID string, userID string, now time.Time) error {
	args := m.Called(ctx, accountID, userID, now)
	return args.Error(0)
}

// --- Test Suite Setup ---

type AccountServiceTestSuite struct {
	suite.Suite
	mockRepo *MockAccountRepositoryFacade
	service  portssvc.AccountSvcFacade
}

func (suite *AccountServiceTestSuite) SetupTest() {
	suite.mockRepo = new(MockAccountRepositoryFacade)
	suite.service = services.NewAccountService(suite.mockRepo)
}

// --- Test Cases ---

func (suite *AccountServiceTestSuite) TestGetAccountByCFID_Success() {
	// Setup
	ctx := context.Background()
	expectedAccount := domain.Account{
		AccountID:    "test-account-id",
		WorkplaceID:  "test-workplace-id",
		CFID:         "CUST-123",
		Name:         "Test Account",
		AccountType:  domain.Asset,
		CurrencyCode: "USD",
		IsActive:     true,
	}

	suite.mockRepo.On("FindAccountByCFID", ctx, "CUST-123", "test-workplace-id").
		Return(&expectedAccount, nil)

	// Execute
	account, err := suite.service.GetAccountByCFID(ctx, "test-workplace-id", "CUST-123", "userid")

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), account)
	assert.Equal(suite.T(), "CUST-123", account.CFID)
	assert.Equal(suite.T(), "test-account-id", account.AccountID)
	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *AccountServiceTestSuite) TestGetAccountByCFID_NotFound() {
	// Setup
	ctx := context.Background()

	suite.mockRepo.On("FindAccountByCFID", ctx, "NON-EXISTENT", "test-workplace-id").
		Return((*domain.Account)(nil), fmt.Errorf("account not found"))

	// Execute
	account, err := suite.service.GetAccountByCFID(ctx, "test-workplace-id", "NON-EXISTENT", "userid")

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), account)
	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *AccountServiceTestSuite) TestGetAccountByCFID_EmptyCFID() {
	// Setup
	ctx := context.Background()

	// Execute - Note: We don't set up any mock expectations since the method should fail before making any repository calls
	account, err := suite.service.GetAccountByCFID(ctx, "test-workplace-id", "", "userid")

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), account)
	assert.Contains(suite.T(), err.Error(), "CFID cannot be empty")
	suite.mockRepo.AssertExpectations(suite.T()) // Verify no unexpected calls were made
}

func (suite *AccountServiceTestSuite) TestCreateAccount_Success() {
	ctx := context.Background()
	creatorUserID := uuid.NewString()
	dummyWorkplaceID := uuid.NewString()
	req := dto.CreateAccountRequest{
		Name:         "Test Savings",
		AccountType:  domain.Asset,
		CurrencyCode: "USD",
	}

	suite.mockRepo.On("SaveAccount", ctx, mock.MatchedBy(func(acc domain.Account) bool {
		return acc.WorkplaceID == dummyWorkplaceID &&
			acc.CreatedBy == creatorUserID
	})).Return(nil).Once()

	createdAccount, err := suite.service.CreateAccount(ctx, dummyWorkplaceID, req, creatorUserID)

	suite.Require().NoError(err)
	suite.Require().NotNil(createdAccount)
	suite.NotEmpty(createdAccount.AccountID)
	suite.Equal(dummyWorkplaceID, createdAccount.WorkplaceID)
	suite.Equal(req.Name, createdAccount.Name)
	suite.Equal(req.AccountType, createdAccount.AccountType)
	suite.Equal(req.CurrencyCode, createdAccount.CurrencyCode)
	suite.True(createdAccount.IsActive)
	suite.Equal(creatorUserID, createdAccount.CreatedBy)
	suite.Equal(creatorUserID, createdAccount.LastUpdatedBy)
	suite.WithinDuration(time.Now(), createdAccount.CreatedAt, time.Second)
	suite.WithinDuration(time.Now(), createdAccount.LastUpdatedAt, time.Second)

	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *AccountServiceTestSuite) TestCreateAccount_SaveError() {
	ctx := context.Background()
	creatorUserID := uuid.NewString()
	dummyWorkplaceID := uuid.NewString()
	req := dto.CreateAccountRequest{
		Name:         "Test Error",
		AccountType:  domain.Asset,
		CurrencyCode: "EUR",
	}

	expectedErr := assert.AnError

	suite.mockRepo.On("SaveAccount", ctx, mock.MatchedBy(func(acc domain.Account) bool {
		return acc.WorkplaceID == dummyWorkplaceID
	})).Return(expectedErr).Once()

	createdAccount, err := suite.service.CreateAccount(ctx, dummyWorkplaceID, req, creatorUserID)

	suite.Require().Error(err)
	suite.Nil(createdAccount)
	suite.ErrorIs(err, expectedErr)

	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *AccountServiceTestSuite) TestGetAccountByID_Success() {
	ctx := context.Background()
	testID := uuid.NewString()
	dummyWorkplaceID := uuid.NewString()
	expectedAccount := &domain.Account{
		AccountID:    testID,
		Name:         "Found Account",
		AccountType:  domain.Liability,
		CurrencyCode: "CAD",
		IsActive:     true,
		WorkplaceID:  dummyWorkplaceID,
	}

	suite.mockRepo.On("FindAccountByID", ctx, testID).Return(expectedAccount, nil).Once()

	account, err := suite.service.GetAccountByID(ctx, dummyWorkplaceID, testID, "userid")

	suite.Require().NoError(err)
	suite.Require().NotNil(account)
	suite.Equal(expectedAccount, account)

	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *AccountServiceTestSuite) TestGetAccountByID_WrongWorkplace() {
	ctx := context.Background()
	testID := uuid.NewString()
	correctWorkplaceID := uuid.NewString()
	incorrectWorkplaceID := uuid.NewString()
	accountFromRepo := &domain.Account{
		AccountID:   testID,
		Name:        "Found Account, Wrong WP",
		WorkplaceID: incorrectWorkplaceID,
		IsActive:    true,
	}

	suite.mockRepo.On("FindAccountByID", ctx, testID).Return(accountFromRepo, nil).Once()

	account, err := suite.service.GetAccountByID(ctx, correctWorkplaceID, testID, "userid")

	suite.Require().Error(err)
	suite.Nil(account)
	suite.ErrorIs(err, apperrors.ErrNotFound)

	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *AccountServiceTestSuite) TestGetAccountByID_NotFound() {
	ctx := context.Background()
	testID := uuid.NewString()
	dummyWorkplaceID := uuid.NewString()

	suite.mockRepo.On("FindAccountByID", ctx, testID).Return(nil, apperrors.ErrNotFound).Once()

	account, err := suite.service.GetAccountByID(ctx, dummyWorkplaceID, testID, "userid")

	suite.Require().Error(err)
	suite.Nil(account)
	suite.ErrorIs(err, apperrors.ErrNotFound)

	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *AccountServiceTestSuite) TestGetAccountByID_RepoError() {
	ctx := context.Background()
	testID := uuid.NewString()
	dummyWorkplaceID := uuid.NewString()
	expectedErr := assert.AnError

	suite.mockRepo.On("FindAccountByID", ctx, testID).Return(nil, expectedErr).Once()

	account, err := suite.service.GetAccountByID(ctx, dummyWorkplaceID, testID, "userid")

	suite.Require().Error(err)
	suite.Nil(account)
	suite.ErrorIs(err, expectedErr)

	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *AccountServiceTestSuite) TestListAccounts_Success() {
	ctx := context.Background()
	limit, offset := 10, 0
	dummyWorkplaceID := uuid.NewString()
	expectedAccounts := []domain.Account{
		{AccountID: uuid.NewString(), Name: "List Acc 1", IsActive: true, WorkplaceID: dummyWorkplaceID},
		{AccountID: uuid.NewString(), Name: "List Acc 2", IsActive: true, WorkplaceID: dummyWorkplaceID},
	}

	suite.mockRepo.On("ListAccounts", ctx, dummyWorkplaceID, limit, offset).Return(expectedAccounts, nil).Once()

	accounts, err := suite.service.ListAccounts(ctx, dummyWorkplaceID, limit, offset)

	suite.Require().NoError(err)
	suite.Equal(expectedAccounts, accounts)

	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *AccountServiceTestSuite) TestListAccounts_Empty() {
	ctx := context.Background()
	limit, offset := 10, 0
	dummyWorkplaceID := uuid.NewString()
	var expectedAccounts []domain.Account

	suite.mockRepo.On("ListAccounts", ctx, dummyWorkplaceID, limit, offset).Return(expectedAccounts, nil).Once()

	accounts, err := suite.service.ListAccounts(ctx, dummyWorkplaceID, limit, offset)

	suite.Require().NoError(err)
	suite.Empty(accounts)
	suite.NotNil(accounts)

	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *AccountServiceTestSuite) TestListAccounts_RepoError() {
	ctx := context.Background()
	limit, offset := 10, 0
	dummyWorkplaceID := uuid.NewString()
	expectedErr := assert.AnError

	suite.mockRepo.On("ListAccounts", ctx, dummyWorkplaceID, limit, offset).Return(nil, expectedErr).Once()

	accounts, err := suite.service.ListAccounts(ctx, dummyWorkplaceID, limit, offset)

	suite.Require().Error(err)
	suite.Nil(accounts)
	suite.ErrorIs(err, expectedErr)

	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *AccountServiceTestSuite) TestDeactivateAccount_Success() {
	ctx := context.Background()
	testID := uuid.NewString()
	userID := uuid.NewString()
	dummyWorkplaceID := uuid.NewString()

	originalAccount := &domain.Account{
		AccountID:   testID,
		WorkplaceID: dummyWorkplaceID,
		IsActive:    true,
	}
	suite.mockRepo.On("FindAccountByID", ctx, testID).Return(originalAccount, nil).Once()

	suite.mockRepo.On("DeactivateAccount", ctx, testID, userID, mock.AnythingOfType("time.Time")).Return(nil).Once()

	err := suite.service.DeactivateAccount(ctx, dummyWorkplaceID, testID, userID)

	suite.Require().NoError(err)

	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *AccountServiceTestSuite) TestDeactivateAccount_WrongWorkplace() {
	ctx := context.Background()
	testID := uuid.NewString()
	userID := uuid.NewString()
	correctWorkplaceID := uuid.NewString()
	incorrectWorkplaceID := uuid.NewString()

	originalAccount := &domain.Account{
		AccountID:   testID,
		WorkplaceID: incorrectWorkplaceID,
		IsActive:    true,
	}
	suite.mockRepo.On("FindAccountByID", ctx, testID).Return(originalAccount, nil).Once()

	err := suite.service.DeactivateAccount(ctx, correctWorkplaceID, testID, userID)

	suite.Require().Error(err)
	suite.ErrorIs(err, apperrors.ErrNotFound)

	suite.mockRepo.AssertExpectations(suite.T())
	suite.mockRepo.AssertNotCalled(suite.T(), "DeactivateAccount", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func (suite *AccountServiceTestSuite) TestDeactivateAccount_NotFound_FindFails() {
	ctx := context.Background()
	testID := uuid.NewString()
	userID := uuid.NewString()
	dummyWorkplaceID := uuid.NewString()

	suite.mockRepo.On("FindAccountByID", ctx, testID).Return(nil, apperrors.ErrNotFound).Once()

	err := suite.service.DeactivateAccount(ctx, dummyWorkplaceID, testID, userID)

	suite.Require().Error(err)
	suite.ErrorIs(err, apperrors.ErrNotFound)

	suite.mockRepo.AssertExpectations(suite.T())
	suite.mockRepo.AssertNotCalled(suite.T(), "DeactivateAccount", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func (suite *AccountServiceTestSuite) TestDeactivateAccount_AlreadyInactive() {
	ctx := context.Background()
	testID := uuid.NewString()
	userID := uuid.NewString()
	dummyWorkplaceID := uuid.NewString()
	validationErr := fmt.Errorf("%w: account %s already inactive", apperrors.ErrValidation, testID)

	originalAccount := &domain.Account{
		AccountID:   testID,
		WorkplaceID: dummyWorkplaceID,
		IsActive:    false,
	}
	suite.mockRepo.On("FindAccountByID", ctx, testID).Return(originalAccount, nil).Once()

	suite.mockRepo.On("DeactivateAccount", ctx, testID, userID, mock.AnythingOfType("time.Time")).Return(validationErr).Once()

	err := suite.service.DeactivateAccount(ctx, dummyWorkplaceID, testID, userID)

	suite.Require().Error(err)
	suite.ErrorIs(err, apperrors.ErrValidation)
	suite.EqualError(err, validationErr.Error())

	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *AccountServiceTestSuite) TestDeactivateAccount_RepoError_FindFails() {
	ctx := context.Background()
	testID := uuid.NewString()
	userID := uuid.NewString()
	dummyWorkplaceID := uuid.NewString()
	expectedErr := assert.AnError

	suite.mockRepo.On("FindAccountByID", ctx, testID).Return(nil, expectedErr).Once()

	err := suite.service.DeactivateAccount(ctx, dummyWorkplaceID, testID, userID)

	suite.Require().Error(err)
	suite.ErrorIs(err, expectedErr)

	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *AccountServiceTestSuite) TestDeactivateAccount_RepoError_DeactivateFails() {
	ctx := context.Background()
	testID := uuid.NewString()
	userID := uuid.NewString()
	dummyWorkplaceID := uuid.NewString()
	expectedErr := assert.AnError

	originalAccount := &domain.Account{
		AccountID:   testID,
		WorkplaceID: dummyWorkplaceID,
		IsActive:    true,
	}
	suite.mockRepo.On("FindAccountByID", ctx, testID).Return(originalAccount, nil).Once()

	suite.mockRepo.On("DeactivateAccount", ctx, testID, userID, mock.AnythingOfType("time.Time")).Return(expectedErr).Once()

	err := suite.service.DeactivateAccount(ctx, dummyWorkplaceID, testID, userID)

	suite.Require().Error(err)
	suite.ErrorIs(err, expectedErr)

	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *AccountServiceTestSuite) TestUpdateAccount_Success_NameAndDescription() {
	ctx := context.Background()
	testID := uuid.NewString()
	updaterUserID := uuid.NewString()
	dummyWorkplaceID := uuid.NewString()
	initialTime := time.Now().Add(-time.Hour)

	originalAccount := &domain.Account{
		AccountID:    testID,
		WorkplaceID:  dummyWorkplaceID,
		Name:         "Original Name",
		Description:  "Original Desc",
		AccountType:  domain.Asset,
		CurrencyCode: "USD",
		IsActive:     true,
		AuditFields: domain.AuditFields{
			CreatedBy:     "creator",
			LastUpdatedBy: "creator",
			CreatedAt:     initialTime,
			LastUpdatedAt: initialTime,
		},
	}

	newName := "Updated Name"
	newDesc := "Updated Desc"
	req := dto.UpdateAccountRequest{
		Name:        &newName,
		Description: &newDesc,
	}

	suite.mockRepo.On("FindAccountByID", ctx, testID).Return(originalAccount, nil).Once()

	suite.mockRepo.On("UpdateAccount", ctx, mock.MatchedBy(func(acc domain.Account) bool {
		return acc.AccountID == testID &&
			acc.WorkplaceID == dummyWorkplaceID &&
			acc.Name == newName &&
			acc.Description == newDesc &&
			acc.IsActive == originalAccount.IsActive &&
			acc.LastUpdatedBy == updaterUserID &&
			acc.LastUpdatedAt.After(initialTime)
	})).Return(nil).Once()

	updatedAccount, err := suite.service.UpdateAccount(ctx, dummyWorkplaceID, testID, req, updaterUserID)

	suite.Require().NoError(err)
	suite.Require().NotNil(updatedAccount)
	suite.Equal(testID, updatedAccount.AccountID)
	suite.Equal(dummyWorkplaceID, updatedAccount.WorkplaceID)
	suite.Equal(newName, updatedAccount.Name)
	suite.Equal(newDesc, updatedAccount.Description)
	suite.True(updatedAccount.IsActive)
	suite.Equal(updaterUserID, updatedAccount.LastUpdatedBy)
	suite.True(updatedAccount.LastUpdatedAt.After(initialTime))

	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *AccountServiceTestSuite) TestUpdateAccount_Success_IsActive() {
	ctx := context.Background()
	testID := uuid.NewString()
	updaterUserID := uuid.NewString()
	dummyWorkplaceID := uuid.NewString()

	originalAccount := &domain.Account{
		AccountID:    testID,
		WorkplaceID:  dummyWorkplaceID,
		Name:         "To Deactivate",
		AccountType:  domain.Liability,
		CurrencyCode: "GBP",
		IsActive:     true,
	}

	newIsActive := false
	req := dto.UpdateAccountRequest{
		IsActive: &newIsActive,
	}

	suite.mockRepo.On("FindAccountByID", ctx, testID).Return(originalAccount, nil).Once()

	suite.mockRepo.On("UpdateAccount", ctx, mock.MatchedBy(func(acc domain.Account) bool {
		return acc.AccountID == testID && !acc.IsActive && acc.LastUpdatedBy == updaterUserID && acc.WorkplaceID == dummyWorkplaceID
	})).Return(nil).Once()

	updatedAccount, err := suite.service.UpdateAccount(ctx, dummyWorkplaceID, testID, req, updaterUserID)

	suite.Require().NoError(err)
	suite.Require().NotNil(updatedAccount)
	suite.False(updatedAccount.IsActive)
	suite.Equal(updaterUserID, updatedAccount.LastUpdatedBy)

	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *AccountServiceTestSuite) TestUpdateAccount_WrongWorkplace() {
	ctx := context.Background()
	testID := uuid.NewString()
	updaterUserID := uuid.NewString()
	correctWorkplaceID := uuid.NewString()
	incorrectWorkplaceID := uuid.NewString()

	originalAccount := &domain.Account{
		AccountID:   testID,
		WorkplaceID: incorrectWorkplaceID,
		Name:        "Original Name",
		IsActive:    true,
	}

	newName := "Updated Name"
	req := dto.UpdateAccountRequest{
		Name: &newName,
	}

	suite.mockRepo.On("FindAccountByID", ctx, testID).Return(originalAccount, nil).Once()

	updatedAccount, err := suite.service.UpdateAccount(ctx, correctWorkplaceID, testID, req, updaterUserID)

	suite.Require().Error(err)
	suite.Nil(updatedAccount)
	suite.ErrorIs(err, apperrors.ErrNotFound)

	suite.mockRepo.AssertExpectations(suite.T())
	suite.mockRepo.AssertNotCalled(suite.T(), "UpdateAccount", mock.Anything, mock.Anything)
}

func (suite *AccountServiceTestSuite) TestUpdateAccount_NoChanges() {
	ctx := context.Background()
	testID := uuid.NewString()
	updaterUserID := uuid.NewString()
	dummyWorkplaceID := uuid.NewString()

	originalAccount := &domain.Account{
		AccountID:   testID,
		WorkplaceID: dummyWorkplaceID,
		Name:        "No Change",
		IsActive:    true,
	}

	req := dto.UpdateAccountRequest{}

	suite.mockRepo.On("FindAccountByID", ctx, testID).Return(originalAccount, nil).Once()

	updatedAccount, err := suite.service.UpdateAccount(ctx, dummyWorkplaceID, testID, req, updaterUserID)

	suite.Require().NoError(err)
	suite.Require().NotNil(updatedAccount)
	suite.Equal(originalAccount, updatedAccount)

	suite.mockRepo.AssertExpectations(suite.T())
	suite.mockRepo.AssertNotCalled(suite.T(), "UpdateAccount", mock.Anything, mock.Anything)
}

func (suite *AccountServiceTestSuite) TestUpdateAccount_NotFound() {
	ctx := context.Background()
	testID := uuid.NewString()
	updaterUserID := uuid.NewString()
	dummyWorkplaceID := uuid.NewString()
	newName := "Doesn't matter"
	req := dto.UpdateAccountRequest{Name: &newName}

	suite.mockRepo.On("FindAccountByID", ctx, testID).Return(nil, apperrors.ErrNotFound).Once()

	updatedAccount, err := suite.service.UpdateAccount(ctx, dummyWorkplaceID, testID, req, updaterUserID)

	suite.Require().Error(err)
	suite.Nil(updatedAccount)
	suite.ErrorIs(err, apperrors.ErrNotFound)

	suite.mockRepo.AssertExpectations(suite.T())
	suite.mockRepo.AssertNotCalled(suite.T(), "UpdateAccount", mock.Anything, mock.Anything)
}

func (suite *AccountServiceTestSuite) TestUpdateAccount_FindError() {
	ctx := context.Background()
	testID := uuid.NewString()
	updaterUserID := uuid.NewString()
	dummyWorkplaceID := uuid.NewString()
	newName := "Doesn't matter"
	req := dto.UpdateAccountRequest{Name: &newName}
	expectedErr := assert.AnError

	suite.mockRepo.On("FindAccountByID", ctx, testID).Return(nil, expectedErr).Once()

	updatedAccount, err := suite.service.UpdateAccount(ctx, dummyWorkplaceID, testID, req, updaterUserID)

	suite.Require().Error(err)
	suite.Nil(updatedAccount)
	suite.ErrorIs(err, expectedErr)

	suite.mockRepo.AssertExpectations(suite.T())
	suite.mockRepo.AssertNotCalled(suite.T(), "UpdateAccount", mock.Anything, mock.Anything)
}

func (suite *AccountServiceTestSuite) TestUpdateAccount_UpdateError() {
	ctx := context.Background()
	testID := uuid.NewString()
	updaterUserID := uuid.NewString()
	dummyWorkplaceID := uuid.NewString()

	originalAccount := &domain.Account{
		AccountID:   testID,
		WorkplaceID: dummyWorkplaceID,
		Name:        "Update Fail",
		IsActive:    true,
	}

	newName := "Will Fail"
	req := dto.UpdateAccountRequest{Name: &newName}
	expectedErr := assert.AnError

	suite.mockRepo.On("FindAccountByID", ctx, testID).Return(originalAccount, nil).Once()

	suite.mockRepo.On("UpdateAccount", ctx, mock.AnythingOfType("domain.Account")).Return(expectedErr).Once()

	updatedAccount, err := suite.service.UpdateAccount(ctx, dummyWorkplaceID, testID, req, updaterUserID)

	suite.Require().Error(err)
	suite.Nil(updatedAccount)
	suite.ErrorIs(err, expectedErr)

	suite.mockRepo.AssertExpectations(suite.T())
}

// --- Run Test Suite ---

func TestAccountService(t *testing.T) {
	suite.Run(t, new(AccountServiceTestSuite))
}
