package testutil

import (
	"context"

	"github.com/haatos/simple-ci/internal/store"
	"github.com/stretchr/testify/mock"
)

type MockCredentialService struct {
	mock.Mock
}

func (m *MockCredentialService) CreateCredential(
	ctx context.Context,
	username, description, sshPrivateKeyHash string,
) (*store.Credential, error) {
	args := m.Called(ctx, username, description, sshPrivateKeyHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.Credential), args.Error(1)
}

func (m *MockCredentialService) GetCredentialByID(
	ctx context.Context,
	credentialID int64,
) (*store.Credential, error) {
	args := m.Called(ctx, credentialID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.Credential), args.Error(1)
}

func (m *MockCredentialService) ListCredentials(ctx context.Context) ([]*store.Credential, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*store.Credential), args.Error(1)
}

func (m *MockCredentialService) UpdateCredential(
	ctx context.Context,
	credentialID int64,
	username, description string,
) error {
	args := m.Called(ctx, credentialID, username, description)
	return args.Error(0)
}

func (m *MockCredentialService) DeleteCredential(ctx context.Context, credentialID int64) error {
	args := m.Called(ctx, credentialID)
	return args.Error(0)
}

func (m *MockCredentialService) DecryptAES(hash string) ([]byte, error) {
	args := m.Called(hash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), nil
}
