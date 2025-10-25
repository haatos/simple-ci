package testutil

import (
	"context"

	"github.com/haatos/simple-ci/internal/store"
	"github.com/stretchr/testify/mock"
)

type MockAPIKeyService struct {
	mock.Mock
}

func (m *MockAPIKeyService) CreateAPIKey(ctx context.Context) (*store.APIKey, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.APIKey), nil
}

func (m *MockAPIKeyService) GetAPIKeyByID(ctx context.Context, id int64) (*store.APIKey, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.APIKey), nil
}

func (m *MockAPIKeyService) GetAPIKeyByValue(
	ctx context.Context,
	value string,
) (*store.APIKey, error) {
	args := m.Called(ctx, value)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.APIKey), nil
}

func (m *MockAPIKeyService) DeleteAPIKey(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAPIKeyService) ListAPIKeys(ctx context.Context) ([]*store.APIKey, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*store.APIKey), nil
}
