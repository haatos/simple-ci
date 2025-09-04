package testutil

import (
	"context"

	"github.com/haatos/simple-ci/internal/store"
	"github.com/haatos/simple-ci/internal/types"
	"github.com/stretchr/testify/mock"
)

type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) GetUserByID(ctx context.Context, userID int64) (*store.User, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.User), args.Error(1)
}

func (m *MockUserService) GetUserBySessionID(
	ctx context.Context,
	sessionID string,
) (*store.User, error) {
	args := m.Called(ctx, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.User), args.Error(1)
}

func (m *MockUserService) CreateAuthSession(
	ctx context.Context,
	userID int64,
) (*store.AuthSession, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.AuthSession), args.Error(1)
}

func (m *MockUserService) GetUserByUsernameAndPassword(
	ctx context.Context,
	username, password string,
) (*store.User, error) {
	args := m.Called(ctx, username, password)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.User), args.Error(1)
}

func (m *MockUserService) CreateUser(
	ctx context.Context,
	userRoleID types.Role,
	username, password string,
	updatePasswordChangedOn bool,
) (*store.User, error) {
	args := m.Called(ctx, userRoleID, username, password, updatePasswordChangedOn)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.User), args.Error(1)
}

func (m *MockUserService) ListUsers(ctx context.Context) ([]*store.User, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*store.User), args.Error(1)
}

func (m *MockUserService) ChangeUserPassword(
	ctx context.Context,
	userID int64,
	oldPassword, newPassword string,
) error {
	args := m.Called(ctx, userID, oldPassword, newPassword)
	return args.Error(0)
}

func (m *MockUserService) SetUserPassword(
	ctx context.Context,
	userID int64,
	newPassword string,
) error {
	args := m.Called(ctx, userID, newPassword)
	return args.Error(0)
}

func (m *MockUserService) ResetUserPassword(
	ctx context.Context,
	userID int64,
	newPassword string,
) error {
	args := m.Called(ctx, userID, newPassword)
	return args.Error(0)
}

func (m *MockUserService) DeleteUser(ctx context.Context, u *store.User) error {
	args := m.Called(ctx, u)
	return args.Error(0)
}

func (m *MockUserService) ListSuperusers(ctx context.Context) ([]store.User, error) {
	args := m.Called(ctx)
	users, err := args.Get(0).([]store.User), args.Error(1)
	return users, err
}

func (m *MockUserService) UpdateUserRole(ctx context.Context, userID int64, role types.Role) error {
	args := m.Called(ctx, userID, role)
	return args.Error(0)
}

func (m *MockUserService) InitializeSuperuser(ctx context.Context) {
	m.Called(ctx)
}
