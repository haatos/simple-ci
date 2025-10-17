package service

import (
	"context"
	"math/rand"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/haatos/simple-ci/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockAPIKeyStore struct {
	mock.Mock
}

func (m *MockAPIKeyStore) CreateAPIKey(ctx context.Context, value string) (*store.APIKey, error) {
	args := m.Called(ctx, value)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.APIKey), nil
}

func (m *MockAPIKeyStore) ReadAPIKeyByID(ctx context.Context, id int64) (*store.APIKey, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.APIKey), nil
}

func (m *MockAPIKeyStore) ReadAPIKeyByValue(
	ctx context.Context,
	value string,
) (*store.APIKey, error) {
	args := m.Called(ctx, value)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.APIKey), nil
}

func (m *MockAPIKeyStore) DeleteAPIKey(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAPIKeyStore) ListAPIKeys(ctx context.Context) ([]*store.APIKey, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*store.APIKey), nil
}

type MockUUIDGenerator struct {
	mock.Mock
}

func (m *MockUUIDGenerator) GenerateUUID() string {
	args := m.Called()
	return args.Get(0).(string)
}

func TestAPIKeyService_CreateAPIKey(t *testing.T) {
	t.Run("success - api key is created", func(t *testing.T) {
		// arrange
		expectedAPIKey := generateAPIKey()
		ctx := context.Background()
		mockStore := new(MockAPIKeyStore)
		mockStore.On("CreateAPIKey", ctx, expectedAPIKey.Value).Return(expectedAPIKey, nil)
		mockUUIDGenerator := new(MockUUIDGenerator)
		mockUUIDGenerator.On("GenerateUUID").Return(expectedAPIKey.Value)
		apiKeyService := NewAPIKeyService(mockStore, mockUUIDGenerator)

		// act
		ak, err := apiKeyService.CreateAPIKey(ctx)

		// assert
		assert.NoError(t, err)
		assert.NotNil(t, ak)
		assert.Equal(t, expectedAPIKey.Value, ak.Value)
	})
}

func TestAPIKeyService_GetAPIKeyByID(t *testing.T) {
	t.Run("success - api key is found by id", func(t *testing.T) {
		// arrange
		expectedAPIKey := generateAPIKey()
		ctx := context.Background()
		mockStore := new(MockAPIKeyStore)
		mockStore.On("ReadAPIKeyByID", ctx, expectedAPIKey.ID).Return(expectedAPIKey, nil)
		apiKeyService := NewAPIKeyService(mockStore, nil)

		// act
		ak, err := apiKeyService.GetAPIKeyByID(ctx, expectedAPIKey.ID)

		// assert
		assert.NoError(t, err)
		assert.NotNil(t, ak)
		assert.Equal(t, expectedAPIKey.ID, ak.ID)
		assert.Equal(t, expectedAPIKey.Value, ak.Value)
		assert.Equal(t, expectedAPIKey.CreatedOn, ak.CreatedOn)
	})
}

func TestAPIKeyService_GetAPIKeyByValue(t *testing.T) {
	t.Run("success - api key is found by value", func(t *testing.T) {
		// arrange
		expectedAPIKey := generateAPIKey()
		ctx := context.Background()
		mockStore := new(MockAPIKeyStore)
		mockStore.On("ReadAPIKeyByValue", ctx, expectedAPIKey.Value).Return(expectedAPIKey, nil)
		apiKeyService := NewAPIKeyService(mockStore, nil)

		// act
		ak, err := apiKeyService.GetAPIKeyByValue(ctx, expectedAPIKey.Value)

		// assert
		assert.NoError(t, err)
		assert.NotNil(t, ak)
		assert.Equal(t, expectedAPIKey.ID, ak.ID)
		assert.Equal(t, expectedAPIKey.Value, ak.Value)
		assert.Equal(t, expectedAPIKey.CreatedOn, ak.CreatedOn)
	})
}

func TestAPIKeyService_DeleteAPIKey(t *testing.T) {
	t.Run("success - api key is deleted", func(t *testing.T) {
		// arrange
		ak := generateAPIKey()
		ctx := context.Background()
		mockStore := new(MockAPIKeyStore)
		mockStore.On("DeleteAPIKey", ctx, ak.ID).Return(nil)
		apiKeyService := NewAPIKeyService(mockStore, nil)

		// act
		err := apiKeyService.DeleteAPIKey(ctx, ak.ID)

		// assert
		assert.NoError(t, err)
	})
}

func TestAPIKeyService_ListAPIKeys(t *testing.T) {
	t.Run("success - api keys are found", func(t *testing.T) {
		// arrange
		expectedAPIKey := generateAPIKey()
		expectedAPIKeys := []*store.APIKey{expectedAPIKey}
		ctx := context.Background()
		mockStore := new(MockAPIKeyStore)
		mockStore.On("ListAPIKeys", ctx).Return(expectedAPIKeys, nil)
		apiKeyService := NewAPIKeyService(mockStore, nil)

		// act
		aks, err := apiKeyService.ListAPIKeys(ctx)

		// assert
		assert.NoError(t, err)
		assert.NotNil(t, aks)
		assert.Equal(t, len(expectedAPIKeys), len(aks))
		assert.True(t, slices.ContainsFunc(aks, func(ak *store.APIKey) bool {
			return ak.ID == expectedAPIKey.ID && ak.Value == expectedAPIKey.Value
		}))
	})
}

func generateAPIKey() *store.APIKey {
	return &store.APIKey{
		ID:        rand.Int63(),
		Value:     uuid.NewString(),
		CreatedOn: time.Now().UTC(),
	}
}
