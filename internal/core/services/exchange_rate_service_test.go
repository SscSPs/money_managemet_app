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

func (m *MockExchangeRateRepository) ListExchangeRates(ctx context.Context, fromCurrency, toCurrency *string, effectiveDate *time.Time, page, pageSize int) ([]domain.ExchangeRate, int, error) {
	args := m.Called(ctx, fromCurrency, toCurrency, effectiveDate, page, pageSize)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]domain.ExchangeRate), args.Int(1), args.Error(2)
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

func (m *MockExchangeRateRepository) FindExchangeRateByIDs(ctx context.Context, rateIDs []string) ([]domain.ExchangeRate, error) {
	args := m.Called(ctx, rateIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.ExchangeRate), args.Error(1)
}

// --- Test Suite ---
type ExchangeRateServiceTestSuite struct {
	suite.Suite
	mockRateRepo    *MockExchangeRateRepository
	mockCurrencySvc *MockCurrencyService
	service         portssvc.ExchangeRateSvcFacade
}

func (suite *ExchangeRateServiceTestSuite) SetupTest() {
	suite.mockRateRepo = new(MockExchangeRateRepository)
	suite.mockCurrencySvc = new(MockCurrencyService)
	// Inject mocks into the service
	suite.service = services.NewExchangeRateService(suite.mockRateRepo, suite.mockCurrencySvc)
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

	// Mock currency validation success
	suite.mockCurrencySvc.On("GetCurrencyByCode", ctx, fromCode).Return(&domain.Currency{CurrencyCode: fromCode}, nil).Once()
	suite.mockCurrencySvc.On("GetCurrencyByCode", ctx, toCode).Return(&domain.Currency{CurrencyCode: toCode}, nil).Once()

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
	suite.mockCurrencySvc.AssertExpectations(suite.T())
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
	req := dto.CreateExchangeRateRequest{FromCurrencyCode: fromCode, ToCurrencyCode: toCode, Rate: decimal.NewFromFloat(1)}

	suite.mockCurrencySvc.On("GetCurrencyByCode", ctx, fromCode).Return(nil, apperrors.ErrNotFound).Once()

	rate, err := suite.service.CreateExchangeRate(ctx, req, creatorUserID)

	suite.Require().Error(err)
	suite.Nil(rate)
	suite.ErrorIs(err, apperrors.ErrValidation)
	suite.Contains(err.Error(), "'from' currency code")
	suite.Contains(err.Error(), "not found")
	suite.mockCurrencySvc.AssertExpectations(suite.T())
	suite.mockRateRepo.AssertNotCalled(suite.T(), "SaveExchangeRate")
}

func (suite *ExchangeRateServiceTestSuite) TestCreateExchangeRate_ToCurrencyNotFound() {
	ctx := context.Background()
	creatorUserID := uuid.NewString()
	fromCode := "USD"
	toCode := "XXX"
	req := dto.CreateExchangeRateRequest{FromCurrencyCode: fromCode, ToCurrencyCode: toCode, Rate: decimal.NewFromFloat(1)}

	// Mock 'from' currency found but 'to' currency not found
	suite.mockCurrencySvc.On("GetCurrencyByCode", ctx, fromCode).Return(&domain.Currency{CurrencyCode: fromCode}, nil).Once()
	suite.mockCurrencySvc.On("GetCurrencyByCode", ctx, toCode).Return(nil, apperrors.ErrNotFound).Once()

	rate, err := suite.service.CreateExchangeRate(ctx, req, creatorUserID)

	suite.Require().Error(err)
	suite.Nil(rate)
	suite.ErrorIs(err, apperrors.ErrValidation)
	suite.Contains(err.Error(), "'to' currency code")
	suite.Contains(err.Error(), "not found")
	suite.mockCurrencySvc.AssertExpectations(suite.T())
	suite.mockRateRepo.AssertNotCalled(suite.T(), "SaveExchangeRate")
}

func (suite *ExchangeRateServiceTestSuite) TestCreateExchangeRate_SaveDuplicate() {
	ctx := context.Background()
	creatorUserID := uuid.NewString()
	fromCode := "USD"
	toCode := "EUR"
	req := dto.CreateExchangeRateRequest{FromCurrencyCode: fromCode, ToCurrencyCode: toCode, Rate: decimal.NewFromFloat(1)}
	duplicateErr := fmt.Errorf("%w: exchange rate exists", apperrors.ErrDuplicate)

	// Mock currency validation success
	suite.mockCurrencySvc.On("GetCurrencyByCode", ctx, fromCode).Return(&domain.Currency{CurrencyCode: fromCode}, nil).Once()
	suite.mockCurrencySvc.On("GetCurrencyByCode", ctx, toCode).Return(&domain.Currency{CurrencyCode: toCode}, nil).Once()

	// Mock rate save duplicate error
	suite.mockRateRepo.On("SaveExchangeRate", ctx, mock.AnythingOfType("domain.ExchangeRate")).Return(duplicateErr).Once()

	rate, err := suite.service.CreateExchangeRate(ctx, req, creatorUserID)

	suite.Require().Error(err)
	suite.Nil(rate)
	suite.ErrorIs(err, apperrors.ErrValidation) // Service maps duplicate to validation

	suite.mockRateRepo.AssertExpectations(suite.T())
	suite.mockCurrencySvc.AssertExpectations(suite.T())
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

// ConvertAmount method doesn't exist in the service interface, so we don't test it

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

	rate, err := suite.service.GetExchangeRateByID(ctx, rateID)

	suite.Require().NoError(err)
	suite.Equal(expectedRate, rate)
	suite.mockRateRepo.AssertExpectations(suite.T())
}

func (suite *ExchangeRateServiceTestSuite) TestGetExchangeRateByID_NotFound() {
	ctx := context.Background()
	rateID := "nonexistent_rate"

	suite.mockRateRepo.On("FindExchangeRateByID", ctx, rateID).
		Return(nil, apperrors.NewNotFoundError("exchange rate not found")).Once()

	rate, err := suite.service.GetExchangeRateByID(ctx, rateID)

	suite.Require().Error(err)
	suite.Nil(rate)
	// Service wraps not found error, check for wrapped error
	suite.Contains(err.Error(), "failed to get exchange rate in service")
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

func (suite *ExchangeRateServiceTestSuite) TestListExchangeRates_Success() {
	ctx := context.Background()
	expectedRates := []domain.ExchangeRate{
		{ExchangeRateID: "rate1", FromCurrencyCode: "USD", ToCurrencyCode: "EUR", Rate: decimal.NewFromFloat(0.85)},
		{ExchangeRateID: "rate2", FromCurrencyCode: "GBP", ToCurrencyCode: "USD", Rate: decimal.NewFromFloat(1.25)},
	}

	suite.mockRateRepo.On("ListExchangeRates", ctx, (*string)(nil), (*string)(nil), (*time.Time)(nil), 0, 0).Return(expectedRates, len(expectedRates), nil).Once()

	rates, err := suite.service.ListExchangeRates(ctx)

	suite.Require().NoError(err)
	suite.Equal(expectedRates, rates)
	suite.mockRateRepo.AssertExpectations(suite.T())
}

func (suite *ExchangeRateServiceTestSuite) TestListExchangeRates_RepositoryError() {
	ctx := context.Background()
	repoErr := fmt.Errorf("database error")

	suite.mockRateRepo.On("ListExchangeRates", ctx, (*string)(nil), (*string)(nil), (*time.Time)(nil), 0, 0).Return(nil, 0, repoErr).Once()

	rates, err := suite.service.ListExchangeRates(ctx)

	suite.Require().Error(err)
	suite.Nil(rates)
	suite.Contains(err.Error(), "failed to list exchange rates in service")
	suite.mockRateRepo.AssertExpectations(suite.T())
}

func (suite *ExchangeRateServiceTestSuite) TestListExchangeRatesByCurrency_Success() {
	ctx := context.Background()
	currencyCode := "USD"
	fromRates := []domain.ExchangeRate{
		{ExchangeRateID: "rate1", FromCurrencyCode: "USD", ToCurrencyCode: "EUR", Rate: decimal.NewFromFloat(0.85)},
	}
	toRates := []domain.ExchangeRate{
		{ExchangeRateID: "rate2", FromCurrencyCode: "GBP", ToCurrencyCode: "USD", Rate: decimal.NewFromFloat(1.25)},
	}

	suite.mockRateRepo.On("ListExchangeRates", ctx, &currencyCode, (*string)(nil), (*time.Time)(nil), 0, 0).Return(fromRates, len(fromRates), nil).Once()
	suite.mockRateRepo.On("ListExchangeRates", ctx, (*string)(nil), &currencyCode, (*time.Time)(nil), 0, 0).Return(toRates, len(toRates), nil).Once()

	rates, err := suite.service.ListExchangeRatesByCurrency(ctx, currencyCode)

	suite.Require().NoError(err)
	suite.Len(rates, 2) // Should have both from and to rates
	suite.mockRateRepo.AssertExpectations(suite.T())
}

func (suite *ExchangeRateServiceTestSuite) TestListExchangeRatesByCurrency_InvalidCode() {
	ctx := context.Background()
	invalidCode := "US" // Too short

	rates, err := suite.service.ListExchangeRatesByCurrency(ctx, invalidCode)

	suite.Require().Error(err)
	suite.Nil(rates)
	suite.ErrorIs(err, apperrors.ErrValidation)
	suite.Contains(err.Error(), "must be 3 letters")
}

// --- Run Suite ---
func TestExchangeRateService(t *testing.T) {
	suite.Run(t, new(ExchangeRateServiceTestSuite))
}
