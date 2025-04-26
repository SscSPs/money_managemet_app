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
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// --- Mock ExchangeRateRepository ---
type MockExchangeRateRepository struct {
	mock.Mock
}

func (m *MockExchangeRateRepository) SaveExchangeRate(ctx context.Context, rate domain.ExchangeRate) error {
	args := m.Called(ctx, rate)
	return args.Error(0)
}

func (m *MockExchangeRateRepository) FindExchangeRate(ctx context.Context, fromCode, toCode string) (*domain.ExchangeRate, error) {
	args := m.Called(ctx, fromCode, toCode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ExchangeRate), args.Error(1)
}

// --- Test Suite ---
type ExchangeRateServiceTestSuite struct {
	suite.Suite
	mockRateRepo     *MockExchangeRateRepository
	mockCurrencyRepo *MockCurrencyRepository // Need mock for currency validation
	service          *services.ExchangeRateService
	currencyService  *services.CurrencyService // Real currency service using mock repo
}

func (suite *ExchangeRateServiceTestSuite) SetupTest() {
	suite.mockRateRepo = new(MockExchangeRateRepository)
	suite.mockCurrencyRepo = new(MockCurrencyRepository)
	// Create real currency service with its mock repo for validation dependency
	suite.currencyService = services.NewCurrencyService(suite.mockCurrencyRepo)
	// Inject mock rate repo and the created currency service
	suite.service = services.NewExchangeRateService(suite.mockRateRepo, suite.currencyService)
}

// --- Test Cases ---

func (suite *ExchangeRateServiceTestSuite) TestCreateExchangeRate_Success() {
	ctx := context.Background()
	creatorUserID := uuid.NewString()
	fromCode := "USD"
	toCode := "EUR"
	req := dto.CreateExchangeRateRequest{
		FromCurrencyCode: fromCode,
		ToCurrencyCode:   toCode,
		Rate:             decimal.NewFromFloat(0.85),
		DateEffective:    time.Now().Truncate(24 * time.Hour),
	}

	// Mock currency validation success
	suite.mockCurrencyRepo.On("FindCurrencyByCode", ctx, fromCode).Return(&domain.Currency{CurrencyCode: fromCode}, nil).Once()
	suite.mockCurrencyRepo.On("FindCurrencyByCode", ctx, toCode).Return(&domain.Currency{CurrencyCode: toCode}, nil).Once()

	// Mock rate save success
	suite.mockRateRepo.On("SaveExchangeRate", ctx, mock.AnythingOfType("domain.ExchangeRate")).Return(nil).Once()

	rate, err := suite.service.CreateExchangeRate(ctx, req, creatorUserID)

	suite.Require().NoError(err)
	suite.Require().NotNil(rate)
	suite.NotEmpty(rate.ExchangeRateID)
	suite.Equal(req.FromCurrencyCode, rate.FromCurrencyCode)
	suite.Equal(req.ToCurrencyCode, rate.ToCurrencyCode)
	suite.True(req.Rate.Equal(rate.Rate))
	suite.Equal(req.DateEffective, rate.DateEffective)
	suite.Equal(creatorUserID, rate.CreatedBy)

	suite.mockRateRepo.AssertExpectations(suite.T())
	suite.mockCurrencyRepo.AssertExpectations(suite.T())
}

func (suite *ExchangeRateServiceTestSuite) TestCreateExchangeRate_InvalidRate() {
	ctx := context.Background()
	creatorUserID := uuid.NewString()
	req := dto.CreateExchangeRateRequest{
		FromCurrencyCode: "USD",
		ToCurrencyCode:   "EUR",
		Rate:             decimal.Zero, // Invalid rate
		DateEffective:    time.Now(),
	}

	rate, err := suite.service.CreateExchangeRate(ctx, req, creatorUserID)

	suite.Require().Error(err)
	suite.Nil(rate)
	suite.ErrorIs(err, apperrors.ErrValidation)
	suite.Contains(err.Error(), "must be positive")
}

func (suite *ExchangeRateServiceTestSuite) TestCreateExchangeRate_SameCurrency() {
	ctx := context.Background()
	creatorUserID := uuid.NewString()
	req := dto.CreateExchangeRateRequest{
		FromCurrencyCode: "USD",
		ToCurrencyCode:   "USD", // Same currency
		Rate:             decimal.NewFromInt(1),
		DateEffective:    time.Now(),
	}

	rate, err := suite.service.CreateExchangeRate(ctx, req, creatorUserID)

	suite.Require().Error(err)
	suite.Nil(rate)
	suite.ErrorIs(err, apperrors.ErrValidation)
	suite.Contains(err.Error(), "cannot be the same")
}

func (suite *ExchangeRateServiceTestSuite) TestCreateExchangeRate_FromCurrencyNotFound() {
	ctx := context.Background()
	creatorUserID := uuid.NewString()
	fromCode := "XXX"
	toCode := "EUR"
	req := dto.CreateExchangeRateRequest{FromCurrencyCode: fromCode, ToCurrencyCode: toCode, Rate: decimal.NewFromFloat(1)} // simplified

	suite.mockCurrencyRepo.On("FindCurrencyByCode", ctx, fromCode).Return(nil, apperrors.ErrNotFound).Once()

	rate, err := suite.service.CreateExchangeRate(ctx, req, creatorUserID)

	suite.Require().Error(err)
	suite.Nil(rate)
	suite.ErrorIs(err, apperrors.ErrValidation)
	suite.Contains(err.Error(), "'from' currency code")
	suite.Contains(err.Error(), "not found")
	suite.mockCurrencyRepo.AssertExpectations(suite.T())
	suite.mockRateRepo.AssertNotCalled(suite.T(), "SaveExchangeRate")
}

// Add similar test for ToCurrencyNotFound

func (suite *ExchangeRateServiceTestSuite) TestCreateExchangeRate_SaveDuplicate() {
	ctx := context.Background()
	creatorUserID := uuid.NewString()
	fromCode := "USD"
	toCode := "EUR"
	req := dto.CreateExchangeRateRequest{FromCurrencyCode: fromCode, ToCurrencyCode: toCode, Rate: decimal.NewFromFloat(1)} // simplified
	duplicateErr := fmt.Errorf("%w: exchange rate exists", apperrors.ErrDuplicate)

	// Mock currency validation success
	suite.mockCurrencyRepo.On("FindCurrencyByCode", ctx, fromCode).Return(&domain.Currency{}, nil).Once()
	suite.mockCurrencyRepo.On("FindCurrencyByCode", ctx, toCode).Return(&domain.Currency{}, nil).Once()
	// Mock rate save duplicate error
	suite.mockRateRepo.On("SaveExchangeRate", ctx, mock.AnythingOfType("domain.ExchangeRate")).Return(duplicateErr).Once()

	rate, err := suite.service.CreateExchangeRate(ctx, req, creatorUserID)

	suite.Require().Error(err)
	suite.Nil(rate)
	suite.ErrorIs(err, apperrors.ErrValidation) // Service maps duplicate to validation
	suite.mockRateRepo.AssertExpectations(suite.T())
	suite.mockCurrencyRepo.AssertExpectations(suite.T())
}

func (suite *ExchangeRateServiceTestSuite) TestGetExchangeRate_Success() {
	ctx := context.Background()
	fromCode := "USD"
	toCode := "EUR"
	expectedRate := &domain.ExchangeRate{FromCurrencyCode: fromCode, ToCurrencyCode: toCode}

	suite.mockRateRepo.On("FindExchangeRate", ctx, fromCode, toCode).Return(expectedRate, nil).Once()

	rate, err := suite.service.GetExchangeRate(ctx, fromCode, toCode)

	suite.Require().NoError(err)
	suite.Equal(expectedRate, rate)
	suite.mockRateRepo.AssertExpectations(suite.T())
}

func (suite *ExchangeRateServiceTestSuite) TestGetExchangeRate_InvalidCode() {
	ctx := context.Background()
	rate, err := suite.service.GetExchangeRate(ctx, "US", "EUR") // Invalid from code
	suite.Require().Error(err)
	suite.Nil(rate)
	suite.ErrorIs(err, apperrors.ErrValidation)

	rate, err = suite.service.GetExchangeRate(ctx, "USD", "EU") // Invalid to code
	suite.Require().Error(err)
	suite.Nil(rate)
	suite.ErrorIs(err, apperrors.ErrValidation)
}

func (suite *ExchangeRateServiceTestSuite) TestGetExchangeRate_NotFound() {
	ctx := context.Background()
	fromCode := "USD"
	toCode := "XXX"

	suite.mockRateRepo.On("FindExchangeRate", ctx, fromCode, toCode).Return(nil, apperrors.ErrNotFound).Once()

	rate, err := suite.service.GetExchangeRate(ctx, fromCode, toCode)

	suite.Require().Error(err)
	suite.Nil(rate)
	// Service wraps not found error, so check the message content
	suite.Contains(err.Error(), "failed to get exchange rate in service")

	suite.mockRateRepo.AssertExpectations(suite.T())
}

// --- Run Suite ---
func TestExchangeRateService(t *testing.T) {
	suite.Run(t, new(ExchangeRateServiceTestSuite))
}
