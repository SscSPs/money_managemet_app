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
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
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

func (m *MockExchangeRateRepository) FindExchangeRateByID(ctx context.Context, rateID string) (*domain.ExchangeRate, error) {
	args := m.Called(ctx, rateID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ExchangeRate), args.Error(1)
}

// --- Test Suite ---
type ExchangeRateServiceTestSuite struct {
	suite.Suite
	mockRateRepo *MockExchangeRateRepository
	service     portssvc.ExchangeRateSvcFacade
}

func (suite *ExchangeRateServiceTestSuite) SetupTest() {
	suite.mockRateRepo = new(MockExchangeRateRepository)
	// Create a mock currency service
	mockCurrencySvc := new(MockCurrencyService)
	// Inject mocks into the service
	suite.service = services.NewExchangeRateService(suite.mockRateRepo, mockCurrencySvc)
}

// MockCurrencyService implements the CurrencySvcFacade interface
type MockCurrencyService struct {
	mock.Mock
}

func (m *MockCurrencyService) GetCurrencyByCode(ctx context.Context, code string) (*domain.Currency, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Currency), args.Error(1)
}

func (m *MockCurrencyService) ListCurrencies(ctx context.Context) ([]domain.Currency, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Currency), args.Error(1)
}

func (m *MockCurrencyService) CreateCurrency(ctx context.Context, req dto.CreateCurrencyRequest, creatorUserID string) (*domain.Currency, error) {
	args := m.Called(ctx, req, creatorUserID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Currency), args.Error(1)
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

	// Get the mock currency service from the test suite
	mockCurrencySvc, ok := suite.service.(interface{ GetCurrencyService() portssvc.CurrencySvcFacade }).GetCurrencyService().(*MockCurrencyService)
	suite.Require().True(ok, "Failed to get mock currency service")

	// Mock currency validation success
	mockCurrencySvc.On("GetCurrencyByCode", ctx, fromCode).Return(&domain.Currency{CurrencyCode: fromCode}, nil).Once()
	mockCurrencySvc.On("GetCurrencyByCode", ctx, toCode).Return(&domain.Currency{CurrencyCode: toCode}, nil).Once()

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
	mockCurrencySvc.AssertExpectations(suite.T())
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

	// Get the mock currency service from the test suite
	mockCurrencySvc, ok := suite.service.(interface{ GetCurrencyService() portssvc.CurrencySvcFacade }).GetCurrencyService().(*MockCurrencyService)
	suite.Require().True(ok, "Failed to get mock currency service")

	mockCurrencySvc.On("GetCurrencyByCode", ctx, fromCode).Return(nil, apperrors.ErrNotFound).Once()

	rate, err := suite.service.CreateExchangeRate(ctx, req, creatorUserID)

	suite.Require().Error(err)
	suite.Nil(rate)
	suite.ErrorIs(err, apperrors.ErrValidation)
	suite.Contains(err.Error(), "'from' currency code")
	suite.Contains(err.Error(), "not found")
	mockCurrencySvc.AssertExpectations(suite.T())
	suite.mockRateRepo.AssertNotCalled(suite.T(), "SaveExchangeRate")
}

func (suite *ExchangeRateServiceTestSuite) TestCreateExchangeRate_ToCurrencyNotFound() {
	ctx := context.Background()
	creatorUserID := uuid.NewString()
	fromCode := "USD"
	toCode := "XXX"
	req := dto.CreateExchangeRateRequest{FromCurrencyCode: fromCode, ToCurrencyCode: toCode, Rate: decimal.NewFromFloat(1)}

	// Get the mock currency service from the test suite
	mockCurrencySvc, ok := suite.service.(interface{ GetCurrencyService() portssvc.CurrencySvcFacade }).GetCurrencyService().(*MockCurrencyService)
	suite.Require().True(ok, "Failed to get mock currency service")

	// Mock 'from' currency found but 'to' currency not found
	mockCurrencySvc.On("GetCurrencyByCode", ctx, fromCode).Return(&domain.Currency{CurrencyCode: fromCode}, nil).Once()
	mockCurrencySvc.On("GetCurrencyByCode", ctx, toCode).Return(nil, apperrors.ErrNotFound).Once()

	rate, err := suite.service.CreateExchangeRate(ctx, req, creatorUserID)

	suite.Require().Error(err)
	suite.Nil(rate)
	suite.ErrorIs(err, apperrors.ErrValidation)
	suite.Contains(err.Error(), "'to' currency code")
	suite.Contains(err.Error(), "not found")
	mockCurrencySvc.AssertExpectations(suite.T())
	suite.mockRateRepo.AssertNotCalled(suite.T(), "SaveExchangeRate")
}

func (suite *ExchangeRateServiceTestSuite) TestCreateExchangeRate_SaveDuplicate() {
	ctx := context.Background()
	creatorUserID := uuid.NewString()
	fromCode := "USD"
	toCode := "EUR"
	req := dto.CreateExchangeRateRequest{FromCurrencyCode: fromCode, ToCurrencyCode: toCode, Rate: decimal.NewFromFloat(1)} // simplified
	duplicateErr := fmt.Errorf("%w: exchange rate exists", apperrors.ErrDuplicate)

	// Get the mock currency service from the test suite
	mockCurrencySvc, ok := suite.service.(interface{ GetCurrencyService() portssvc.CurrencySvcFacade }).GetCurrencyService().(*MockCurrencyService)
	suite.Require().True(ok, "Failed to get mock currency service")

	// Mock currency validation success
	mockCurrencySvc.On("GetCurrencyByCode", ctx, fromCode).Return(&domain.Currency{CurrencyCode: fromCode}, nil).Once()
	mockCurrencySvc.On("GetCurrencyByCode", ctx, toCode).Return(&domain.Currency{CurrencyCode: toCode}, nil).Once()

	// Mock rate save duplicate error
	suite.mockRateRepo.On("SaveExchangeRate", ctx, mock.AnythingOfType("domain.ExchangeRate")).Return(duplicateErr).Once()

	rate, err := suite.service.CreateExchangeRate(ctx, req, creatorUserID)

	suite.Require().Error(err)
	suite.Nil(rate)
	suite.ErrorIs(err, apperrors.ErrValidation) // Service maps duplicate to validation

	suite.mockRateRepo.AssertExpectations(suite.T())
	mockCurrencySvc.AssertExpectations(suite.T())
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

func (suite *ExchangeRateServiceTestSuite) TestConvertAmount_Success() {
	ctx := context.Background()
	fromAmount := decimal.NewFromFloat(100.00)
	fromCurrency := "USD"
	toCurrency := "EUR"
	exchangeRate := &domain.ExchangeRate{
		ExchangeRateID:   "rate_123",
		FromCurrencyCode: fromCurrency,
		ToCurrencyCode:   toCurrency,
		Rate:             decimal.NewFromFloat(0.85),
		DateEffective:    time.Now().Truncate(24 * time.Hour),
	}

	// Mock the exchange rate lookup
	suite.mockRateRepo.On("FindExchangeRate", ctx, fromCurrency, toCurrency).Return(exchangeRate, nil).Once()

	// Test conversion
	convertedAmount, rate, err := suite.service.(interface {
		ConvertAmount(ctx context.Context, fromAmount decimal.Decimal, fromCurrency, toCurrency string) (decimal.Decimal, *domain.ExchangeRate, error)
	}).ConvertAmount(ctx, fromAmount, fromCurrency, toCurrency)

	suite.Require().NoError(err)
	suite.Require().NotNil(rate)
	suite.True(convertedAmount.Equal(decimal.NewFromFloat(85.00)))
	suite.Equal(exchangeRate.ExchangeRateID, rate.ExchangeRateID)
	suite.mockRateRepo.AssertExpectations(suite.T())
}

func (suite *ExchangeRateServiceTestSuite) TestConvertAmount_InvalidCurrency() {
	ctx := context.Background()
	fromAmount := decimal.NewFromFloat(100.00)

	// Test with invalid from currency
	_, _, err := suite.service.(interface {
		ConvertAmount(ctx context.Context, fromAmount decimal.Decimal, fromCurrency, toCurrency string) (decimal.Decimal, *domain.ExchangeRate, error)
	}).ConvertAmount(ctx, fromAmount, "US", "EUR")

	suite.Require().Error(err)
	suite.ErrorIs(err, apperrors.ErrValidation)

	// Test with invalid to currency
	_, _, err = suite.service.(interface {
		ConvertAmount(ctx context.Context, fromAmount decimal.Decimal, fromCurrency, toCurrency string) (decimal.Decimal, *domain.ExchangeRate, error)
	}).ConvertAmount(ctx, fromAmount, "USD", "EU")

	suite.Require().Error(err)
	suite.ErrorIs(err, apperrors.ErrValidation)
}

func (suite *ExchangeRateServiceTestSuite) TestConvertAmount_RateNotFound() {
	ctx := context.Background()
	fromAmount := decimal.NewFromFloat(100.00)
	fromCurrency := "USD"
	toCurrency := "EUR"

	// Mock the exchange rate lookup to return not found
	suite.mockRateRepo.On("FindExchangeRate", ctx, fromCurrency, toCurrency).
		Return(nil, apperrors.NewNotFoundError("exchange rate not found")).Once()

	// Test conversion
	_, _, err := suite.service.(interface {
		ConvertAmount(ctx context.Context, fromAmount decimal.Decimal, fromCurrency, toCurrency string) (decimal.Decimal, *domain.ExchangeRate, error)
	}).ConvertAmount(ctx, fromAmount, fromCurrency, toCurrency)

	suite.Require().Error(err)
	suite.ErrorIs(err, apperrors.ErrNotFound)
	suite.mockRateRepo.AssertExpectations(suite.T())
}

func (suite *ExchangeRateServiceTestSuite) TestGetExchangeRateByID_Success() {
	ctx := context.Background()
	rateID := "rate_123"
	expectedRate := &domain.ExchangeRate{
		ExchangeRateID:   rateID,
		FromCurrencyCode: "USD",
		ToCurrencyCode:   "EUR",
		Rate:             decimal.NewFromFloat(0.85),
	}

	suite.mockRateRepo.On("FindExchangeRateByID", ctx, rateID).Return(expectedRate, nil).Once()

	rate, err := suite.service.(interface {
		GetExchangeRateByID(ctx context.Context, rateID string) (*domain.ExchangeRate, error)
	}).GetExchangeRateByID(ctx, rateID)

	suite.Require().NoError(err)
	suite.Equal(expectedRate, rate)
	suite.mockRateRepo.AssertExpectations(suite.T())
}

func (suite *ExchangeRateServiceTestSuite) TestGetExchangeRateByID_NotFound() {
	ctx := context.Background()
	rateID := "nonexistent_rate"

	suite.mockRateRepo.On("FindExchangeRateByID", ctx, rateID).
		Return(nil, apperrors.NewNotFoundError("exchange rate not found")).Once()

	rate, err := suite.service.(interface {
		GetExchangeRateByID(ctx context.Context, rateID string) (*domain.ExchangeRate, error)
	}).GetExchangeRateByID(ctx, rateID)

	suite.Require().Error(err)
	suite.Nil(rate)
	suite.ErrorIs(err, apperrors.ErrNotFound)
	suite.mockRateRepo.AssertExpectations(suite.T())
}

func TestNewExchangeRateService(t *testing.T) {
	// Create mock dependencies
	mockRateRepo := new(MockExchangeRateRepository)
	mockCurrencySvc := new(MockCurrencyService)
	
	// Create the service
	service := services.NewExchangeRateService(mockRateRepo, mockCurrencySvc)
	
	// Verify the service is created with the correct dependencies
	assert.NotNil(t, service)
	
	// Test that the service implements the correct interface
	var _ portssvc.ExchangeRateSvcFacade = service
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
