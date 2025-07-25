package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/SscSPs/money_managemet_app/internal/apperrors"
	"github.com/SscSPs/money_managemet_app/internal/core/domain"
	portssvc "github.com/SscSPs/money_managemet_app/internal/core/ports/services"
	"github.com/SscSPs/money_managemet_app/internal/core/services"
	"github.com/SscSPs/money_managemet_app/internal/dto"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// --- Mock UserRepository (based on UserService usage) ---
type MockUserRepository struct {
	mock.Mock
	ListUsersFn                 func(ctx context.Context, limit, offset int) ([]domain.User, error)
	FindUserByIDFn              func(ctx context.Context, userID string) (*domain.User, error)
	UpdateUserFn                func(ctx context.Context, user domain.User) error
	CreateUserFn                func(ctx context.Context, user domain.User) error
	FindUserByUsernameFn        func(ctx context.Context, username string) (*domain.User, error)
	UpdateRefreshTokenFn        func(ctx context.Context, user *domain.User, refreshTokenHash string, refreshTokenExpiryTime time.Time) error
	ClearRefreshTokenFn         func(ctx context.Context, user *domain.User) error
	DeleteUserFn                func(ctx context.Context, user *domain.User) error
	GetUserByUsernameFn         func(ctx context.Context, username string) (*domain.User, error)
	FindUserByEmailFn           func(ctx context.Context, email string) (*domain.User, error)
	FindUserByProviderDetailsFn func(ctx context.Context, authProvider string, providerUserID string) (*domain.User, error)
	MarkUserDeletedFn           func(ctx context.Context, user *domain.User, deleterUserID string) error
	BeginFn                     func(ctx context.Context) (pgx.Tx, error)
	CommitFn                    func(ctx context.Context, tx pgx.Tx) error
	RollbackFn                  func(ctx context.Context, tx pgx.Tx) error
}

func (m *MockUserRepository) SaveUser(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

// CreateUser is an alias for SaveUser to maintain backward compatibility
func (m *MockUserRepository) CreateUser(ctx context.Context, user *domain.User) error {
	return m.SaveUser(ctx, user)
}

func (m *MockUserRepository) FindUserByID(ctx context.Context, userID string) (*domain.User, error) {
	if m.FindUserByIDFn != nil {
		return m.FindUserByIDFn(ctx, userID)
	}
	args := m.Called(ctx, userID)
	var user *domain.User
	if args.Get(0) != nil {
		user = args.Get(0).(*domain.User)
	}
	return user, args.Error(1)
}

func (m *MockUserRepository) FindUsers(ctx context.Context, limit, offset int) ([]domain.User, error) {
	if m.ListUsersFn != nil {
		return m.ListUsersFn(ctx, limit, offset)
	}
	args := m.Called(ctx, limit, offset)
	var users []domain.User
	if args.Get(0) != nil {
		users = args.Get(0).([]domain.User)
	}
	return users, args.Error(1)
}

func (m *MockUserRepository) UpdateUser(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) MarkUserDeleted(ctx context.Context, user *domain.User, deleterUserID string) error {
	args := m.Called(ctx, user, deleterUserID)
	return args.Error(0)
}

func (m *MockUserRepository) FindUserByUsername(ctx context.Context, username string) (*domain.User, error) {
	args := m.Called(ctx, username)
	var user *domain.User
	if args.Get(0) != nil {
		user = args.Get(0).(*domain.User)
	}
	return user, args.Error(1)
}

func (m *MockUserRepository) UpdateRefreshToken(ctx context.Context, user *domain.User, refreshTokenHash string, refreshTokenExpiryTime time.Time) error {
	args := m.Called(ctx, user, refreshTokenHash, refreshTokenExpiryTime)
	return args.Error(0)
}

func (m *MockUserRepository) ClearRefreshToken(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) DeleteUser(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) GetUserByUsername(ctx context.Context, username string) (*domain.User, error) {
	args := m.Called(ctx, username)
	var user *domain.User
	if args.Get(0) != nil {
		user = args.Get(0).(*domain.User)
	}
	return user, args.Error(1)
}

func (m *MockUserRepository) FindUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	if m.FindUserByEmailFn != nil {
		return m.FindUserByEmailFn(ctx, email)
	}
	args := m.Called(ctx, email)
	var user *domain.User
	if args.Get(0) != nil {
		user = args.Get(0).(*domain.User)
	}
	return user, args.Error(1)
}

func (m *MockUserRepository) FindUserByProviderDetails(ctx context.Context, authProvider string, providerUserID string) (*domain.User, error) {
	if m.FindUserByProviderDetailsFn != nil {
		return m.FindUserByProviderDetailsFn(ctx, authProvider, providerUserID)
	}
	args := m.Called(ctx, authProvider, providerUserID)
	var user *domain.User
	if args.Get(0) != nil {
		user = args.Get(0).(*domain.User)
	}
	return user, args.Error(1)
}

func (m *MockUserRepository) Begin(ctx context.Context) (pgx.Tx, error) {
	if m.BeginFn != nil {
		return m.BeginFn(ctx)
	}
	args := m.Called(ctx)
	var tx pgx.Tx
	if args.Get(0) != nil {
		tx = args.Get(0).(pgx.Tx)
	}
	return tx, args.Error(1)
}

func (m *MockUserRepository) Commit(ctx context.Context, tx pgx.Tx) error {
	if m.CommitFn != nil {
		return m.CommitFn(ctx, tx)
	}
	args := m.Called(ctx, tx)
	return args.Error(0)
}

func (m *MockUserRepository) Rollback(ctx context.Context, tx pgx.Tx) error {
	if m.RollbackFn != nil {
		return m.RollbackFn(ctx, tx)
	}
	args := m.Called(ctx, tx)
	return args.Error(0)
}

// --- Test Suite ---
type UserServiceTestSuite struct {
	suite.Suite
	mockUserRepo *MockUserRepository
	service      portssvc.UserSvcFacade
}

func (suite *UserServiceTestSuite) SetupTest() {
	suite.mockUserRepo = new(MockUserRepository)
	suite.service = services.NewUserService(suite.mockUserRepo)
}

// --- Test Cases ---

// --- CreateUser Tests ---
func (suite *UserServiceTestSuite) TestCreateUser_Success() {
	ctx := context.Background()
	username := "testuser"
	password := "password123"
	name := "Test User"

	createUserReq := dto.CreateUserRequest{
		Username: username,
		Password: password,
		Name:     name,
	}

	// Mock: FindUserByUsername should not find the user
	suite.mockUserRepo.On("FindUserByUsername", ctx, username).Return(nil, apperrors.ErrNotFound).Once()
	// Mock: SaveUser (called internally by service's CreateUser) should succeed
	// It receives a domain.User after the service maps the DTO and hashes password
	suite.mockUserRepo.On("SaveUser", ctx, mock.MatchedBy(func(user *domain.User) bool {
		return user.Username == username &&
			user.Name == name &&
			user.PasswordHash != nil &&
			*user.PasswordHash != password &&
			user.AuthProvider == domain.ProviderLocal
	})).Return(nil).Once()

	createdUser, err := suite.service.CreateUser(ctx, createUserReq)

	suite.Require().NoError(err)
	suite.Require().NotNil(createdUser)
	suite.Equal(username, createdUser.Username)
	suite.Equal(name, createdUser.Name)
	suite.NotEmpty(createdUser.UserID)
	println("-------------")
	println(createdUser.UserID)
	println(createdUser.PasswordHash)
	suite.Nil(createdUser.PasswordHash)
	// Check if AuthProvider is either empty string or ProviderLocal (both are valid for local auth)
	if createdUser.AuthProvider != "" {
		suite.Equal(domain.ProviderLocal, createdUser.AuthProvider)
	}
	suite.False(createdUser.IsVerified)

	suite.mockUserRepo.AssertExpectations(suite.T())
}

func (suite *UserServiceTestSuite) TestCreateUser_SaveError() {
	ctx := context.Background()
	username := "testuser-save-error"
	password := "password123"
	name := "Test User Save Error"

	createUserReq := dto.CreateUserRequest{
		Username: username,
		Password: password,
		Name:     name,
	}
	expectedErr := assert.AnError

	// Mock: FindUserByUsername should not find the user
	suite.mockUserRepo.On("FindUserByUsername", ctx, username).Return(nil, apperrors.ErrNotFound).Once()
	// Mock: SaveUser (called internally by service's CreateUser) should fail
	suite.mockUserRepo.On("SaveUser", ctx, mock.AnythingOfType("*domain.User")).Return(expectedErr).Once()

	createdUser, err := suite.service.CreateUser(ctx, createUserReq)

	suite.Require().Error(err)
	suite.Nil(createdUser)
	// The error from service should wrap the repo error or be specific
	// For now, let's check if it contains the original error, but ideally, it's a more specific app error.
	suite.Contains(err.Error(), expectedErr.Error()) // Or suite.ErrorIs(err, someSpecificAppError)

	suite.mockUserRepo.AssertExpectations(suite.T())
}

// --- GetUserByID Tests ---
func (suite *UserServiceTestSuite) TestGetUserByID_Success() {
	ctx := context.Background()
	userID := uuid.NewString()
	expectedUser := &domain.User{UserID: userID, Name: "Found User"}

	suite.mockUserRepo.On("FindUserByID", ctx, userID).Return(expectedUser, nil).Once()

	user, err := suite.service.GetUserByID(ctx, userID)

	suite.Require().NoError(err)
	suite.Equal(expectedUser, user)
	suite.mockUserRepo.AssertExpectations(suite.T())
}

func (suite *UserServiceTestSuite) TestGetUserByID_NotFound() {
	ctx := context.Background()
	userID := uuid.NewString()

	suite.mockUserRepo.On("FindUserByID", ctx, userID).Return(nil, apperrors.ErrNotFound).Once()

	user, err := suite.service.GetUserByID(ctx, userID)

	suite.Require().Error(err)
	suite.Nil(user)
	suite.ErrorIs(err, apperrors.ErrNotFound)
	suite.mockUserRepo.AssertExpectations(suite.T())
}

func (suite *UserServiceTestSuite) TestGetUserByID_RepoError() {
	ctx := context.Background()
	userID := uuid.NewString()
	expectedErr := assert.AnError

	suite.mockUserRepo.On("FindUserByID", ctx, userID).Return(nil, expectedErr).Once()

	user, err := suite.service.GetUserByID(ctx, userID)

	suite.Require().Error(err)
	suite.Nil(user)
	suite.ErrorIs(err, expectedErr)
	suite.mockUserRepo.AssertExpectations(suite.T())
}

// --- ListUsers Tests ---
func (suite *UserServiceTestSuite) TestListUsers_Success() {
	ctx := context.Background()
	limit, offset := 10, 0
	expectedUsers := []domain.User{{UserID: uuid.NewString()}, {UserID: uuid.NewString()}}

	suite.mockUserRepo.On("FindUsers", ctx, limit, offset).Return(expectedUsers, nil).Once()

	users, err := suite.service.ListUsers(ctx, limit, offset)

	suite.Require().NoError(err)
	suite.Require().NotNil(users)
	suite.Len(users, len(expectedUsers))
	for i := range expectedUsers {
		suite.Equal(expectedUsers[i].UserID, users[i].UserID)
	}
	suite.mockUserRepo.AssertExpectations(suite.T())
}

func (suite *UserServiceTestSuite) TestListUsers_Empty() {
	ctx := context.Background()
	limit, offset := 5, 10
	var expectedUsers []domain.User // Empty slice

	suite.mockUserRepo.On("FindUsers", ctx, limit, offset).Return(expectedUsers, nil).Once()

	users, err := suite.service.ListUsers(ctx, limit, offset)

	suite.Require().NoError(err)
	suite.Require().NotNil(users)
	suite.Empty(users)
	suite.mockUserRepo.AssertExpectations(suite.T())
}

func (suite *UserServiceTestSuite) TestListUsers_RepoError() {
	ctx := context.Background()
	expectedErr := assert.AnError

	suite.mockUserRepo.On("FindUsers", ctx, 10, 0).Return(nil, expectedErr).Once()

	users, err := suite.service.ListUsers(ctx, 10, 0)

	suite.Require().Error(err)
	suite.ErrorIs(err, expectedErr)
	suite.Nil(users)
	suite.mockUserRepo.AssertExpectations(suite.T())
}

// --- UpdateUser Tests ---
func (suite *UserServiceTestSuite) TestUpdateUser_Success() {
	ctx := context.Background()
	userID := uuid.NewString()
	updaterUserID := uuid.NewString()
	newName := "Updated Name"
	req := dto.UpdateUserRequest{Name: &newName}
	originalUser := &domain.User{
		UserID: userID,
		Name:   "Original Name",
		AuditFields: domain.AuditFields{
			LastUpdatedAt: time.Now().Add(-time.Hour),
			LastUpdatedBy: "somebodyElse",
		},
	}
	originalTimestamp := originalUser.LastUpdatedAt

	suite.mockUserRepo.On("FindUserByID", ctx, userID).Return(originalUser, nil).Once()
	suite.mockUserRepo.On("UpdateUser", ctx, mock.AnythingOfType("*domain.User")).Return(nil).Once().Run(func(args mock.Arguments) {
		userArg := args.Get(1).(*domain.User)
		suite.T().Logf("DEBUG: Comparing times in mock:")
		suite.T().Logf("  originalTimestamp: %v", originalTimestamp)
		suite.T().Logf("  userArg.LastUpdatedAt: %v", userArg.LastUpdatedAt)
		suite.Equal(userID, userArg.UserID)
		suite.Equal(newName, userArg.Name)
		suite.Equal(updaterUserID, userArg.LastUpdatedBy)
		suite.NotEqual(originalTimestamp, userArg.LastUpdatedAt)
		suite.True(userArg.LastUpdatedAt.After(originalTimestamp))
	})

	user, err := suite.service.UpdateUser(ctx, userID, req, updaterUserID)

	suite.Require().NoError(err)
	suite.Require().NotNil(user)
	suite.Equal(newName, user.Name)
	suite.Equal(updaterUserID, user.LastUpdatedBy)
	suite.NotEqual(originalTimestamp, user.LastUpdatedAt)
	suite.True(user.LastUpdatedAt.After(originalTimestamp))

	suite.mockUserRepo.AssertExpectations(suite.T())
}

func (suite *UserServiceTestSuite) TestUpdateUser_NoChange() {
	ctx := context.Background()
	userID := uuid.NewString()
	updaterUserID := uuid.NewString()
	originalName := "Original Name"
	originalUser := &domain.User{UserID: userID, Name: originalName, AuditFields: domain.AuditFields{LastUpdatedBy: "prevUpdater", LastUpdatedAt: time.Now().Add(-time.Hour)}}
	req := dto.UpdateUserRequest{Name: &originalName} // No actual change

	suite.mockUserRepo.On("FindUserByID", ctx, userID).Return(originalUser, nil).Once()
	// UpdateUser should NOT be called

	user, err := suite.service.UpdateUser(ctx, userID, req, updaterUserID)

	suite.Require().NoError(err)
	suite.Equal(originalUser, user) // Should return the original unchanged user
	suite.mockUserRepo.AssertExpectations(suite.T())
	suite.mockUserRepo.AssertNotCalled(suite.T(), "UpdateUser", mock.Anything, mock.Anything)
}

func (suite *UserServiceTestSuite) TestUpdateUser_NotFound() {
	ctx := context.Background()
	userID := uuid.NewString()
	updaterUserID := uuid.NewString()
	newName := "Updated Name"
	req := dto.UpdateUserRequest{Name: &newName}

	suite.mockUserRepo.On("FindUserByID", ctx, userID).Return(nil, apperrors.ErrNotFound).Once()
	// UpdateUser should NOT be called

	user, err := suite.service.UpdateUser(ctx, userID, req, updaterUserID)

	suite.Require().Error(err)
	suite.Nil(user)
	suite.ErrorIs(err, apperrors.ErrNotFound)
	suite.mockUserRepo.AssertExpectations(suite.T())
	suite.mockUserRepo.AssertNotCalled(suite.T(), "UpdateUser", mock.Anything, mock.Anything)
}

func (suite *UserServiceTestSuite) TestUpdateUser_UpdateError() {
	ctx := context.Background()
	userID := uuid.NewString()
	updaterUserID := uuid.NewString()
	newName := "Updated Name"
	req := dto.UpdateUserRequest{Name: &newName}
	originalUser := &domain.User{UserID: userID, Name: "Original Name"}
	expectedErr := assert.AnError

	suite.mockUserRepo.On("FindUserByID", ctx, userID).Return(originalUser, nil).Once()
	suite.mockUserRepo.On("UpdateUser", ctx, mock.AnythingOfType("*domain.User")).Return(expectedErr).Once()

	user, err := suite.service.UpdateUser(ctx, userID, req, updaterUserID)

	suite.Require().Error(err)
	suite.Nil(user)
	suite.ErrorIs(err, expectedErr)
	suite.mockUserRepo.AssertExpectations(suite.T())
}

// --- DeleteUser Tests ---
func (suite *UserServiceTestSuite) TestDeleteUser_Success() {
	ctx := context.Background()
	userID := uuid.NewString()
	deleterUserID := uuid.NewString()

	existingUser := &domain.User{
		UserID: userID,
	}

	suite.mockUserRepo.On("FindUserByID", ctx, userID).Return(existingUser, nil).Once()
	suite.mockUserRepo.On("MarkUserDeleted", ctx, existingUser, deleterUserID).Return(nil).Once()

	err := suite.service.DeleteUser(ctx, userID, deleterUserID)

	suite.Require().NoError(err)
	suite.mockUserRepo.AssertExpectations(suite.T())
}

func (suite *UserServiceTestSuite) TestDeleteUser_NotFound() {
	ctx := context.Background()
	userID := uuid.NewString()
	deleterUserID := uuid.NewString()

	suite.mockUserRepo.On("FindUserByID", ctx, userID).Return((*domain.User)(nil), apperrors.ErrNotFound).Once()

	err := suite.service.DeleteUser(ctx, userID, deleterUserID)

	suite.Require().Error(err)
	suite.ErrorIs(err, apperrors.ErrNotFound)
	suite.mockUserRepo.AssertExpectations(suite.T())
}

func (suite *UserServiceTestSuite) TestDeleteUser_RepoError() {
	ctx := context.Background()
	userID := uuid.NewString()
	deleterUserID := uuid.NewString()
	expectedErr := assert.AnError

	suite.mockUserRepo.On("FindUserByID", ctx, userID).Return((*domain.User)(nil), expectedErr).Once()

	err := suite.service.DeleteUser(ctx, userID, deleterUserID)

	suite.Require().Error(err)
	suite.Contains(err.Error(), "error finding user to delete")
	suite.mockUserRepo.AssertExpectations(suite.T())
}

// --- Run Suite ---
func TestUserService(t *testing.T) {
	suite.Run(t, new(UserServiceTestSuite))
}
