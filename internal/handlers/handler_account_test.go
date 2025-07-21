package handlers_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/SscSPs/money_managemet_app/internal/core/domain"                  // Renamed import
	portssvc "github.com/SscSPs/money_managemet_app/internal/core/ports/services" // Renamed import
	"github.com/SscSPs/money_managemet_app/internal/dto"
	"github.com/SscSPs/money_managemet_app/internal/handlers"   // Import handlers package
	"github.com/SscSPs/money_managemet_app/internal/middleware" // Needed for JWT secret
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5" // Added for JWT generation
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// --- Mock AccountService ---
type MockAccountService struct {
	mock.Mock
}

func (m *MockAccountService) CreateAccount(ctx context.Context, workplaceID string, req dto.CreateAccountRequest, userID string) (*domain.Account, error) {
	args := m.Called(ctx, workplaceID, req, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Account), args.Error(1)
}
func (m *MockAccountService) GetAccountByID(ctx context.Context, workplaceID string, accountID string, userID string) (*domain.Account, error) {
	args := m.Called(ctx, workplaceID, accountID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Account), args.Error(1)
}
func (m *MockAccountService) GetAccountByIDs(ctx context.Context, workplaceID string, accountIDs []string, userID string) (map[string]domain.Account, error) {
	args := m.Called(ctx, workplaceID, accountIDs, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]domain.Account), args.Error(1)
}
func (m *MockAccountService) ListAccounts(ctx context.Context, workplaceID string, limit int, offset int) ([]domain.Account, error) {
	args := m.Called(ctx, workplaceID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Account), args.Error(1)
}
func (m *MockAccountService) UpdateAccount(ctx context.Context, workplaceID string, accountID string, req dto.UpdateAccountRequest, userID string) (*domain.Account, error) {
	args := m.Called(ctx, workplaceID, accountID, req, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Account), args.Error(1)
}
func (m *MockAccountService) DeactivateAccount(ctx context.Context, workplaceID string, accountID string, userID string) error {
	args := m.Called(ctx, workplaceID, accountID, userID)
	return args.Error(0)
}
func (m *MockAccountService) CalculateAccountBalance(ctx context.Context, workplaceID string, accountID string, userID string) (decimal.Decimal, error) {
	args := m.Called(ctx, workplaceID, accountID, userID)
	return args.Get(0).(decimal.Decimal), args.Error(1)
}

func (m *MockAccountService) GetAccountByCFID(ctx context.Context, workplaceID string, cfid string, userID string) (*domain.Account, error) {
	args := m.Called(ctx, workplaceID, cfid, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Account), args.Error(1)
}

// Ensure mock implements the interface
var _ portssvc.AccountSvcFacade = (*MockAccountService)(nil)

// --- Mock JournalService ---
type MockJournalService struct {
	mock.Mock
}

func (m *MockJournalService) CreateJournal(ctx context.Context, workplaceID string, req dto.CreateJournalRequest, creatorUserID string) (*domain.Journal, error) {
	args := m.Called(ctx, workplaceID, req, creatorUserID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Journal), args.Error(1)
}
func (m *MockJournalService) GetJournalByID(ctx context.Context, workplaceID string, journalID string, requestingUserID string) (*domain.Journal, error) {
	args := m.Called(ctx, workplaceID, journalID, requestingUserID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Journal), args.Error(1)
}
func (m *MockJournalService) ListJournals(ctx context.Context, workplaceID string, userID string, params dto.ListJournalsParams) (*dto.ListJournalsResponse, error) {
	args := m.Called(ctx, workplaceID, userID, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.ListJournalsResponse), args.Error(1)
}
func (m *MockJournalService) UpdateJournal(ctx context.Context, workplaceID string, journalID string, req dto.UpdateJournalRequest, requestingUserID string) (*domain.Journal, error) {
	args := m.Called(ctx, workplaceID, journalID, req, requestingUserID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Journal), args.Error(1)
}
func (m *MockJournalService) DeactivateJournal(ctx context.Context, workplaceID string, journalID string, requestingUserID string) error {
	args := m.Called(ctx, workplaceID, journalID, requestingUserID)
	return args.Error(0)
}
func (m *MockJournalService) ListTransactionsByAccount(ctx context.Context, workplaceID string, accountID string, userID string, params dto.ListTransactionsParams) (*dto.ListTransactionsResponse, error) {
	args := m.Called(ctx, workplaceID, accountID, userID, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.ListTransactionsResponse), args.Error(1)
}
func (m *MockJournalService) CalculateAccountBalance(ctx context.Context, workplaceID string, accountID string, userID string) (decimal.Decimal, error) {
	args := m.Called(ctx, workplaceID, accountID, userID)
	if args.Get(0) == nil {
		return decimal.Decimal{}, args.Error(1)
	}
	return args.Get(0).(decimal.Decimal), args.Error(1)
}
func (m *MockJournalService) ReverseJournal(ctx context.Context, workplaceID string, journalID string, userID string) (*domain.Journal, error) {
	args := m.Called(ctx, workplaceID, journalID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Journal), args.Error(1)
}

// Ensure mock implements the interface
var _ portssvc.JournalSvcFacade = (*MockJournalService)(nil)

// --- Test Suite ---
type AccountHandlerTestSuite struct {
	suite.Suite
	router             *gin.Engine
	mockAccountService *MockAccountService
	mockJournalService *MockJournalService
	jwtSecret          string // Store JWT secret for token generation
	// No need for handler instance field, routes are registered once
}

// decimalPtr returns a pointer to the provided decimal.Decimal value.
func decimalPtr(d decimal.Decimal) *decimal.Decimal {
	return &d
}

// generateTestToken creates a dummy JWT for testing.
func (suite *AccountHandlerTestSuite) generateTestToken(userID string) string {
	claims := jwt.RegisteredClaims{
		Issuer:    "mma-test",
		Subject:   userID,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tsignedString, err := token.SignedString([]byte(suite.jwtSecret))
	if err != nil {
		suite.FailNow("Failed to sign test token", err.Error())
	}
	return tsignedString
}

func (suite *AccountHandlerTestSuite) SetupTest() {
	gin.SetMode(gin.TestMode)
	suite.router = gin.New()
	suite.jwtSecret = "test-secret-key-that-is-long-enough" // Use a test secret

	// Create dummy config for middleware - not needed for AuthMiddleware
	// dummyCfg := &config.Config{
	// 	JWTSecret: suite.jwtSecret,
	// 	// Add other config fields if middleware needs them
	// }

	// Use the actual AuthMiddleware
	suite.router.Use(middleware.AuthMiddleware(suite.jwtSecret))

	suite.mockAccountService = new(MockAccountService)
	suite.mockJournalService = new(MockJournalService)

	// Register routes - requires the actual registration function
	v1 := suite.router.Group("/api/v1/workplaces/:workplace_id")                           // Mimic grouping
	handlers.RegisterAccountRoutes(v1, suite.mockAccountService, suite.mockJournalService) // Use exported name
}

// --- Test Cases ---

func (suite *AccountHandlerTestSuite) TestListTransactionsByAccount_Success() {
	workplaceID := uuid.NewString()
	accountID := uuid.NewString()
	requestingUserID := uuid.NewString() // Use a UUID for the test user ID
	limit := 10

	// Prepare expected transactions
	usd := "USD"
	expectedTransactions := []dto.TransactionResponse{
		{
			TransactionID:   uuid.NewString(),
			JournalID:       uuid.NewString(),
			AccountID:       accountID,
			Amount:          decimal.NewFromInt(100),
			TransactionType: domain.Debit,
			OriginalAmount:  decimalPtr(decimal.NewFromInt(100)),
			OriginalCurrency: &usd,
			CreatedAt:       time.Now(),
		},
		{
			TransactionID:   uuid.NewString(),
			JournalID:       uuid.NewString(),
			AccountID:       accountID,
			Amount:          decimal.NewFromInt(50),
			TransactionType: domain.Credit,
			OriginalAmount:  decimalPtr(decimal.NewFromInt(50)),
			OriginalCurrency: &usd,
			CreatedAt:       time.Now().Add(-time.Hour),
		},
	}
	expectedResponse := &dto.ListTransactionsResponse{
		Transactions: expectedTransactions,
		NextToken:    nil, // No more pages in test
	}

	// Setup mock expectation with new signature
	suite.mockJournalService.On("ListTransactionsByAccount",
		mock.AnythingOfType("*context.valueCtx"), // Context will now have values from middleware
		workplaceID,
		accountID,
		requestingUserID, // Expect the user ID from the token
		mock.MatchedBy(func(p dto.ListTransactionsParams) bool {
			return p.Limit == limit
		}),
	).Return(expectedResponse, nil).Once()

	// Create request with updated query parameters
	url := fmt.Sprintf("/api/v1/workplaces/%s/accounts/%s/transactions?limit=%d", workplaceID, accountID, limit)
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	// Add the generated token to the Authorization header
	token := suite.generateTestToken(requestingUserID)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json") // Good practice

	// Create response recorder
	w := httptest.NewRecorder()

	// Serve request
	suite.router.ServeHTTP(w, req)

	// Assertions
	suite.Equal(http.StatusOK, w.Code, "Expected status OK")

	var responseBody dto.ListTransactionsResponse
	err := json.Unmarshal(w.Body.Bytes(), &responseBody)
	suite.NoError(err, "Failed to unmarshal response body")
	suite.Len(responseBody.Transactions, len(expectedTransactions))
	// Compare transaction details
	if len(responseBody.Transactions) == len(expectedTransactions) {
		suite.Equal(expectedTransactions[0].TransactionID, responseBody.Transactions[0].TransactionID)
		suite.Equal(expectedTransactions[1].TransactionID, responseBody.Transactions[1].TransactionID)
	}

	// Assert mock calls
	suite.mockJournalService.AssertExpectations(suite.T())
	suite.mockAccountService.AssertNotCalled(suite.T(), "ListAccounts") // Ensure unrelated service methods not called
}

// TODO: Add tests for other scenarios:
// - Service returns ErrNotFound
// - Service returns ErrForbidden
// - Service returns other error
// - Invalid query params (limit/offset)
// - Missing workplaceID/accountID path params

// --- Run Test Suite ---
func TestAccountHandler(t *testing.T) {
	suite.Run(t, new(AccountHandlerTestSuite))
}
