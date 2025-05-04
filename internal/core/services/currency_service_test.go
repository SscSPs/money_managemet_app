package services_test

import (
	"context"
	"testing"

	"github.com/SscSPs/money_managemet_app/internal/apperrors"
	"github.com/SscSPs/money_managemet_app/internal/core/domain"
	portssvc "github.com/SscSPs/money_managemet_app/internal/core/ports/services"
	"github.com/SscSPs/money_managemet_app/internal/core/services"
	"github.com/SscSPs/money_managemet_app/internal/dto"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// --- Mock CurrencyRepository ---
type MockCurrencyRepository struct {
	mock.Mock
}

func (m *MockCurrencyRepository) SaveCurrency(ctx context.Context, currency domain.Currency) error {
	args := m.Called(ctx, currency)
	return args.Error(0)
}

func (m *MockCurrencyRepository) FindCurrencyByCode(ctx context.Context, currencyCode string) (*domain.Currency, error) {
	args := m.Called(ctx, currencyCode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Currency), args.Error(1)
}

func (m *MockCurrencyRepository) ListCurrencies(ctx context.Context) ([]domain.Currency, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Currency), args.Error(1)
}

// --- Test Suite ---
type CurrencyServiceTestSuite struct {
	suite.Suite
	mockRepo *MockCurrencyRepository
	service  portssvc.CurrencySvcFacade
}

func (suite *CurrencyServiceTestSuite) SetupTest() {
	suite.mockRepo = new(MockCurrencyRepository)
	suite.service = services.NewCurrencyService(suite.mockRepo)
}

// --- Test Cases ---

func (suite *CurrencyServiceTestSuite) TestCreateCurrency_Success() {
	ctx := context.Background()
	creatorUserID := uuid.NewString()
	req := dto.CreateCurrencyRequest{
		CurrencyCode: "TST",
		Symbol:       "T",
		Name:         "Test Currency",
	}

	suite.mockRepo.On("SaveCurrency", ctx, mock.MatchedBy(func(c domain.Currency) bool {
		return c.CurrencyCode == req.CurrencyCode && c.Symbol == req.Symbol && c.Name == req.Name && c.CreatedBy == creatorUserID && c.LastUpdatedBy == creatorUserID
	})).Return(nil).Once()

	currency, err := suite.service.CreateCurrency(ctx, req, creatorUserID)

	suite.Require().NoError(err)
	suite.Require().NotNil(currency)
	suite.Equal(req.CurrencyCode, currency.CurrencyCode)
	suite.Equal(req.Symbol, currency.Symbol)
	suite.Equal(req.Name, currency.Name)
	suite.Equal(creatorUserID, currency.CreatedBy)

	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *CurrencyServiceTestSuite) TestCreateCurrency_SaveError() {
	ctx := context.Background()
	creatorUserID := uuid.NewString()
	req := dto.CreateCurrencyRequest{
		CurrencyCode: "ERR",
		Symbol:       "E",
		Name:         "Error Currency",
	}
	expectedErr := assert.AnError

	suite.mockRepo.On("SaveCurrency", ctx, mock.AnythingOfType("domain.Currency")).Return(expectedErr).Once()

	currency, err := suite.service.CreateCurrency(ctx, req, creatorUserID)

	suite.Require().Error(err)
	suite.Nil(currency)
	suite.ErrorIs(err, expectedErr)
	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *CurrencyServiceTestSuite) TestGetCurrencyByCode_Success() {
	ctx := context.Background()
	code := "TST"
	expectedCurrency := &domain.Currency{CurrencyCode: code}

	suite.mockRepo.On("FindCurrencyByCode", ctx, code).Return(expectedCurrency, nil).Once()

	currency, err := suite.service.GetCurrencyByCode(ctx, code)

	suite.Require().NoError(err)
	suite.Equal(expectedCurrency, currency)
	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *CurrencyServiceTestSuite) TestGetCurrencyByCode_NotFound() {
	ctx := context.Background()
	code := "NTF"

	suite.mockRepo.On("FindCurrencyByCode", ctx, code).Return(nil, apperrors.ErrNotFound).Once()

	currency, err := suite.service.GetCurrencyByCode(ctx, code)

	suite.Require().Error(err)
	suite.Nil(currency)
	suite.ErrorIs(err, apperrors.ErrNotFound)
	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *CurrencyServiceTestSuite) TestGetCurrencyByCode_RepoError() {
	ctx := context.Background()
	code := "ERR"
	expectedErr := assert.AnError

	suite.mockRepo.On("FindCurrencyByCode", ctx, code).Return(nil, expectedErr).Once()

	currency, err := suite.service.GetCurrencyByCode(ctx, code)

	suite.Require().Error(err)
	suite.Nil(currency)
	suite.ErrorIs(err, expectedErr)
	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *CurrencyServiceTestSuite) TestListCurrencies_Success() {
	ctx := context.Background()
	expectedCurrencies := []domain.Currency{{CurrencyCode: "TST"}, {CurrencyCode: "CUR"}}

	suite.mockRepo.On("ListCurrencies", ctx).Return(expectedCurrencies, nil).Once()

	currencies, err := suite.service.ListCurrencies(ctx)

	suite.Require().NoError(err)
	suite.Equal(expectedCurrencies, currencies)
	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *CurrencyServiceTestSuite) TestListCurrencies_Empty() {
	ctx := context.Background()
	var expectedCurrencies []domain.Currency // Empty slice

	suite.mockRepo.On("ListCurrencies", ctx).Return(expectedCurrencies, nil).Once()

	currencies, err := suite.service.ListCurrencies(ctx)

	suite.Require().NoError(err)
	suite.Empty(currencies)
	suite.NotNil(currencies)
	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *CurrencyServiceTestSuite) TestListCurrencies_RepoError() {
	ctx := context.Background()
	expectedErr := assert.AnError

	suite.mockRepo.On("ListCurrencies", ctx).Return(nil, expectedErr).Once()

	currencies, err := suite.service.ListCurrencies(ctx)

	suite.Require().Error(err)
	suite.Nil(currencies)
	suite.ErrorIs(err, expectedErr)
	suite.mockRepo.AssertExpectations(suite.T())
}

// --- Run Suite ---
func TestCurrencyService(t *testing.T) {
	suite.Run(t, new(CurrencyServiceTestSuite))
}
