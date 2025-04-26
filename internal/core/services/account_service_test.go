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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// MockAccountRepository is a mock type for the AccountRepository interface
type MockAccountRepository struct {
	mock.Mock
}

// --- Implement mock methods for AccountRepository ---

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

// --- Test Suite Setup ---

type AccountServiceTestSuite struct {
	suite.Suite
	mockRepo *MockAccountRepository
	service  *services.AccountService
}

func (suite *AccountServiceTestSuite) SetupTest() {
	suite.mockRepo = new(MockAccountRepository)
	suite.service = services.NewAccountService(suite.mockRepo)
}

// --- Test Cases ---

func (suite *AccountServiceTestSuite) TestCreateAccount_Success() {
	ctx := context.Background()
	creatorUserID := uuid.NewString()
	req := dto.CreateAccountRequest{
		Name:         "Test Savings",
		AccountType:  domain.Asset,
		CurrencyCode: "USD",
		UserID:       creatorUserID, // Usually from context, but passed for audit
	}

	// Expect SaveAccount to be called once
	suite.mockRepo.On("SaveAccount", ctx, mock.AnythingOfType("domain.Account")).Return(nil).Once()

	// Call the service method
	createdAccount, err := suite.service.CreateAccount(ctx, req, creatorUserID)

	// Assertions
	suite.Require().NoError(err)
	suite.Require().NotNil(createdAccount)
	suite.NotEmpty(createdAccount.AccountID)
	suite.Equal(req.Name, createdAccount.Name)
	suite.Equal(req.AccountType, createdAccount.AccountType)
	suite.Equal(req.CurrencyCode, createdAccount.CurrencyCode)
	suite.True(createdAccount.IsActive)
	suite.Equal(creatorUserID, createdAccount.CreatedBy)
	suite.Equal(creatorUserID, createdAccount.LastUpdatedBy)
	suite.WithinDuration(time.Now(), createdAccount.CreatedAt, time.Second)
	suite.WithinDuration(time.Now(), createdAccount.LastUpdatedAt, time.Second)

	// Assert that the mock expectations were met
	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *AccountServiceTestSuite) TestCreateAccount_SaveError() {
	ctx := context.Background()
	creatorUserID := uuid.NewString()
	req := dto.CreateAccountRequest{
		Name:         "Test Error",
		AccountType:  domain.Asset,
		CurrencyCode: "EUR",
		UserID:       creatorUserID,
	}

	expectedErr := assert.AnError // Simulate a repository error

	// Expect SaveAccount to be called and return an error
	suite.mockRepo.On("SaveAccount", ctx, mock.AnythingOfType("domain.Account")).Return(expectedErr).Once()

	// Call the service method
	createdAccount, err := suite.service.CreateAccount(ctx, req, creatorUserID)

	// Assertions
	suite.Require().Error(err)
	suite.Nil(createdAccount)
	suite.ErrorIs(err, expectedErr) // Check if the underlying error matches

	// Assert that the mock expectations were met
	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *AccountServiceTestSuite) TestGetAccountByID_Success() {
	ctx := context.Background()
	testID := uuid.NewString()
	expectedAccount := &domain.Account{
		AccountID:    testID,
		Name:         "Found Account",
		AccountType:  domain.Liability,
		CurrencyCode: "CAD",
		IsActive:     true,
	}

	// Expect FindAccountByID to be called and return the account
	suite.mockRepo.On("FindAccountByID", ctx, testID).Return(expectedAccount, nil).Once()

	// Call the service method
	account, err := suite.service.GetAccountByID(ctx, testID)

	// Assertions
	suite.Require().NoError(err)
	suite.Require().NotNil(account)
	suite.Equal(expectedAccount, account)

	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *AccountServiceTestSuite) TestGetAccountByID_NotFound() {
	ctx := context.Background()
	testID := uuid.NewString()

	// Expect FindAccountByID to be called and return ErrNotFound
	suite.mockRepo.On("FindAccountByID", ctx, testID).Return(nil, apperrors.ErrNotFound).Once()

	// Call the service method
	account, err := suite.service.GetAccountByID(ctx, testID)

	// Assertions
	suite.Require().Error(err)
	suite.Nil(account)
	suite.ErrorIs(err, apperrors.ErrNotFound)

	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *AccountServiceTestSuite) TestGetAccountByID_RepoError() {
	ctx := context.Background()
	testID := uuid.NewString()
	expectedErr := assert.AnError

	// Expect FindAccountByID to be called and return a generic error
	suite.mockRepo.On("FindAccountByID", ctx, testID).Return(nil, expectedErr).Once()

	// Call the service method
	account, err := suite.service.GetAccountByID(ctx, testID)

	// Assertions
	suite.Require().Error(err)
	suite.Nil(account)
	suite.ErrorIs(err, expectedErr)

	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *AccountServiceTestSuite) TestListAccounts_Success() {
	ctx := context.Background()
	limit, offset := 10, 0
	expectedAccounts := []domain.Account{
		{AccountID: uuid.NewString(), Name: "List Acc 1", IsActive: true},
		{AccountID: uuid.NewString(), Name: "List Acc 2", IsActive: true},
	}

	// Expect ListAccounts to be called and return accounts
	suite.mockRepo.On("ListAccounts", ctx, limit, offset).Return(expectedAccounts, nil).Once()

	// Call service method
	accounts, err := suite.service.ListAccounts(ctx, limit, offset)

	// Assertions
	suite.Require().NoError(err)
	suite.Equal(expectedAccounts, accounts)

	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *AccountServiceTestSuite) TestListAccounts_Empty() {
	ctx := context.Background()
	limit, offset := 10, 0
	var expectedAccounts []domain.Account // Empty slice

	// Expect ListAccounts to be called and return empty slice
	suite.mockRepo.On("ListAccounts", ctx, limit, offset).Return(expectedAccounts, nil).Once()

	// Call service method
	accounts, err := suite.service.ListAccounts(ctx, limit, offset)

	// Assertions
	suite.Require().NoError(err)
	suite.Empty(accounts)
	suite.NotNil(accounts) // Should be an empty slice, not nil

	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *AccountServiceTestSuite) TestListAccounts_RepoError() {
	ctx := context.Background()
	limit, offset := 10, 0
	expectedErr := assert.AnError

	// Expect ListAccounts to be called and return an error
	suite.mockRepo.On("ListAccounts", ctx, limit, offset).Return(nil, expectedErr).Once()

	// Call service method
	accounts, err := suite.service.ListAccounts(ctx, limit, offset)

	// Assertions
	suite.Require().Error(err)
	suite.Nil(accounts)
	suite.ErrorIs(err, expectedErr)

	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *AccountServiceTestSuite) TestDeactivateAccount_Success() {
	ctx := context.Background()
	testID := uuid.NewString()
	userID := uuid.NewString()

	// Expect DeactivateAccount to be called and return nil
	suite.mockRepo.On("DeactivateAccount", ctx, testID, userID, mock.AnythingOfType("time.Time")).Return(nil).Once()

	// Call service method
	err := suite.service.DeactivateAccount(ctx, testID, userID)

	// Assertions
	suite.Require().NoError(err)

	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *AccountServiceTestSuite) TestDeactivateAccount_NotFound() {
	ctx := context.Background()
	testID := uuid.NewString()
	userID := uuid.NewString()

	// Expect DeactivateAccount to be called and return ErrNotFound
	suite.mockRepo.On("DeactivateAccount", ctx, testID, userID, mock.AnythingOfType("time.Time")).Return(apperrors.ErrNotFound).Once()

	// Call service method
	err := suite.service.DeactivateAccount(ctx, testID, userID)

	// Assertions
	suite.Require().Error(err)
	suite.ErrorIs(err, apperrors.ErrNotFound)

	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *AccountServiceTestSuite) TestDeactivateAccount_AlreadyInactive() {
	ctx := context.Background()
	testID := uuid.NewString()
	userID := uuid.NewString()
	validationErr := fmt.Errorf("%w: account %s already inactive", apperrors.ErrValidation, testID)

	// Expect DeactivateAccount to be called and return validation error
	suite.mockRepo.On("DeactivateAccount", ctx, testID, userID, mock.AnythingOfType("time.Time")).Return(validationErr).Once()

	// Call service method
	err := suite.service.DeactivateAccount(ctx, testID, userID)

	// Assertions
	suite.Require().Error(err)
	suite.ErrorIs(err, apperrors.ErrValidation)
	suite.EqualError(err, validationErr.Error())

	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *AccountServiceTestSuite) TestDeactivateAccount_RepoError() {
	ctx := context.Background()
	testID := uuid.NewString()
	userID := uuid.NewString()
	expectedErr := assert.AnError

	// Expect DeactivateAccount to be called and return a generic error
	suite.mockRepo.On("DeactivateAccount", ctx, testID, userID, mock.AnythingOfType("time.Time")).Return(expectedErr).Once()

	// Call service method
	err := suite.service.DeactivateAccount(ctx, testID, userID)

	// Assertions
	suite.Require().Error(err)
	suite.ErrorIs(err, expectedErr)

	suite.mockRepo.AssertExpectations(suite.T())
}

// --- Run Test Suite ---

func TestAccountService(t *testing.T) {
	suite.Run(t, new(AccountServiceTestSuite))
}
