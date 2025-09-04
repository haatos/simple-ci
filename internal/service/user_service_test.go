package service

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/haatos/simple-ci/internal/store"
	"github.com/haatos/simple-ci/internal/types"
	"github.com/haatos/simple-ci/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

const testUserPassword string = "testpassword"

type MockUserStore struct {
	mock.Mock
}

func (m *MockUserStore) CreateUser(
	context.Context,
	types.Role,
	string,
	string,
) (*store.User, error) {
	panic("not implemented")
}
func (m *MockUserStore) ReadUserByID(ctx context.Context, userID int64) (*store.User, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.User), args.Error(1)
}

func (m *MockUserStore) ReadUserByUsername(
	ctx context.Context,
	username string,
) (*store.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.User), args.Error(1)
}

func (m *MockUserStore) ReadUserBySessionID(
	ctx context.Context,
	sessionID string,
) (*store.User, error) {
	args := m.Called(ctx, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.User), args.Error(1)
}

func (m *MockUserStore) UpdateUserRole(
	ctx context.Context,
	userID int64,
	role types.Role,
) error {
	args := m.Called(ctx, userID, role)
	return args.Error(0)
}

func (m *MockUserStore) UpdateUserPassword(
	ctx context.Context,
	userID int64,
	passwordHash string,
) error {
	args := m.Called(ctx, userID, passwordHash)
	return args.Error(0)
}

func (m *MockUserStore) UpdateUserPasswordChangedOn(
	ctx context.Context,
	userID int64,
	changedOn *time.Time,
) error {
	args := m.Called(ctx, userID, changedOn)
	return args.Error(0)
}

func (m *MockUserStore) DeleteUser(ctx context.Context, userID int64) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockUserStore) ListUsers(ctx context.Context) ([]*store.User, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*store.User), args.Error(1)
}

func (m *MockUserStore) ListSuperusers(ctx context.Context) ([]store.User, error) {
	args := m.Called(ctx)
	return args.Get(0).([]store.User), args.Error(1)
}

func (m *MockUserStore) CreateAuthSession(
	ctx context.Context,
	sessionID string,
	userID int64,
	expires time.Time,
) (*store.AuthSession, error) {
	args := m.Called(ctx, sessionID, userID, expires)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.AuthSession), nil
}

func (m *MockUserStore) DeleteAuthSessionsByUserID(ctx context.Context, userID int64) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func TestUserService_GetUserByID(t *testing.T) {
	t.Run("success - user is found", func(t *testing.T) {
		// arrange
		expectedUser := generateUser(types.Operator, nil, nil)
		mockStore := new(MockUserStore)
		mockStore.On(
			"ReadUserByID",
			context.Background(),
			expectedUser.UserID,
		).Return(expectedUser, nil)
		userService := NewUserService(mockStore)

		// act
		user, err := userService.GetUserByID(context.Background(), expectedUser.UserID)

		// assert
		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, expectedUser.UserRoleID, user.UserRoleID)
		assert.Equal(t, expectedUser.UserID, user.UserID)
		assert.Equal(t, expectedUser.Username, user.Username)
	})
}

func TestUserService_GetUserByUsername(t *testing.T) {
	t.Run("success - user is found", func(t *testing.T) {
		// arrange
		expectedUser := generateUser(types.Operator, nil, nil)
		password := "testuserpassword"
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		expectedUser.PasswordHash = string(hash)
		mockStore := new(MockUserStore)
		assert.NoError(t, err)
		mockStore.On(
			"ReadUserByUsername",
			context.Background(),
			expectedUser.Username,
		).Return(expectedUser, nil)
		userService := NewUserService(mockStore)

		// act
		user, err := userService.GetUserByUsernameAndPassword(
			context.Background(), expectedUser.Username, password,
		)

		// assert
		assert.NoError(t, err)
		assert.NotNil(t, user)
		err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
		assert.NoError(t, err)
		assert.Equal(t, expectedUser.UserRoleID, user.UserRoleID)
		assert.Equal(t, expectedUser.UserID, user.UserID)
		assert.Equal(t, expectedUser.Username, user.Username)
	})
	t.Run("failure - password and hash mismatch", func(t *testing.T) {
		// arrange
		mockStore := new(MockUserStore)
		username := "getuserbyusername"
		password := "testuserpassword"
		var expectedUser *store.User
		mockStore.On("ReadUserByUsername", context.Background(), username).
			Return(expectedUser, bcrypt.ErrMismatchedHashAndPassword)
		userService := NewUserService(mockStore)

		// act
		user, err := userService.GetUserByUsernameAndPassword(
			context.Background(),
			username,
			password,
		)

		// assert
		assert.Error(t, err)
		assert.ErrorIs(t, err, bcrypt.ErrMismatchedHashAndPassword)
		assert.Nil(t, user)
	})
}

func TestUserService_GetUserBySessionID(t *testing.T) {
	t.Run("success - user is found", func(t *testing.T) {
		// arrange
		expectedUser := generateUser(
			types.Operator,
			nil,
			util.AsPtr(time.Now().UTC().Add(30*time.Second)),
		)
		sessionID := "sessionid"
		mockStore := new(MockUserStore)
		mockStore.On(
			"ReadUserBySessionID",
			context.Background(),
			sessionID,
		).Return(expectedUser, nil)
		userService := NewUserService(mockStore)

		// act
		user, err := userService.GetUserBySessionID(context.Background(), sessionID)

		// assert
		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, expectedUser.UserRoleID, user.UserRoleID)
		assert.Equal(t, expectedUser.UserID, user.UserID)
		assert.Equal(t, expectedUser.Username, user.Username)
		assert.True(t, user.SessionExpires.Valid)
		assert.True(t, user.SessionExpires.Time.After(time.Now().UTC()))

	})
}

func TestUserService_UpdateUserRole(t *testing.T) {
	t.Run("success - user is found", func(t *testing.T) {
		// arrange
		var userID int64 = 1
		mockStore := new(MockUserStore)
		mockStore.On("UpdateUserRole", context.Background(), userID, types.Admin).Return(nil)
		userService := NewUserService(mockStore)

		// act
		err := userService.UpdateUserRole(context.Background(), userID, types.Admin)

		// assert
		assert.NoError(t, err)
	})
}

func TestUserService_DeleteUser(t *testing.T) {
	t.Run("success - user is found", func(t *testing.T) {
		// arrange
		var userID int64 = 1
		mockStore := new(MockUserStore)
		mockStore.On("DeleteUser", context.Background(), userID).Return(nil)
		userService := NewUserService(mockStore)

		// act
		err := userService.DeleteUser(context.Background(), &store.User{UserID: userID})

		// assert
		assert.NoError(t, err)
	})
}

func TestUserService_ListUsers(t *testing.T) {
	t.Run("success - users found", func(t *testing.T) {
		// arrange
		mockStore := new(MockUserStore)
		expectedUsers := []*store.User{
			generateUser(types.Operator, nil, nil),
			generateUser(types.Admin, nil, nil),
		}
		mockStore.On("ListUsers", context.Background()).Return(expectedUsers, nil)
		userService := NewUserService(mockStore)

		// act
		users, err := userService.ListUsers(context.Background())

		// assert
		assert.NoError(t, err)
		assert.Equal(t, len(expectedUsers), len(users))
	})
}

func TestUserService_ListSuperusers(t *testing.T) {
	t.Run("success - superusers found", func(t *testing.T) {
		// arrange
		mockStore := new(MockUserStore)
		expectedUsers := []store.User{
			*generateUser(types.Operator, nil, nil),
			*generateUser(types.Admin, nil, nil),
			*generateUser(types.Superuser, nil, nil),
		}
		mockStore.On(
			"ListSuperusers",
			context.Background(),
		).Return(expectedUsers, nil)
		userService := NewUserService(mockStore)

		// act
		users, err := userService.ListSuperusers(context.Background())

		// assert
		assert.NoError(t, err)
		assert.Equal(t, len(expectedUsers), len(users))
	})
}

func generateUser(
	role types.Role,
	passwordChangedOn *time.Time,
	sessionExpires *time.Time,
) *store.User {
	hash, _ := bcrypt.GenerateFromPassword([]byte(testUserPassword), bcrypt.DefaultCost)
	user := &store.User{
		UserID:            rand.Int63(),
		UserRoleID:        role,
		Username:          fmt.Sprintf("testuser%d", time.Now().UnixNano()),
		PasswordHash:      string(hash),
		PasswordChangedOn: passwordChangedOn,
	}
	if sessionExpires != nil {
		user.SessionExpires = sql.NullTime{Valid: true, Time: *sessionExpires}
	}
	return user
}
