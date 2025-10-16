package store

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestAPIKeySQLiteStore_CreateAPIKey(t *testing.T) {
	t.Run("success - api key is created", func(t *testing.T) {
		// arrange
		value := uuid.NewString()

		// act
		ak, err := apiKeyStore.CreateAPIKey(context.Background(), value)

		// assert
		assert.NoError(t, err)
		assert.NotNil(t, ak)
		assert.Equal(t, value, ak.Value)
	})
}

func TestAPIKeySQLiteStore_ReadAPIKeyByID(t *testing.T) {
	t.Run("success - key is found by id", func(t *testing.T) {
		// arrange
		expectedKey := createAPIKey(t)

		// act
		ak, err := apiKeyStore.ReadAPIKeyByID(context.Background(), expectedKey.ID)

		// assert
		assert.NoError(t, err)
		assert.NotNil(t, ak)
		assert.Equal(t, expectedKey.ID, ak.ID)
		assert.Equal(t, expectedKey.Value, ak.Value)
	})
	t.Run("failure - key is not found by id", func(t *testing.T) {
		// arrange
		var id int64 = 432512

		// act
		ak, err := apiKeyStore.ReadAPIKeyByID(context.Background(), id)

		// assert
		assert.Error(t, err)
		assert.True(t, errors.Is(err, sql.ErrNoRows))
		assert.Nil(t, ak)
	})
}

func TestAPIKeySQLiteStore_ReadAPIKeyByValue(t *testing.T) {
	t.Run("success - key is found by value", func(t *testing.T) {
		// arrange
		expectedKey := createAPIKey(t)

		// act
		ak, err := apiKeyStore.ReadAPIKeyByValue(context.Background(), expectedKey.Value)

		// assert
		assert.NoError(t, err)
		assert.NotNil(t, ak)
		assert.Equal(t, expectedKey.ID, ak.ID)
		assert.Equal(t, expectedKey.Value, ak.Value)
	})
	t.Run("failure - key is not found by value", func(t *testing.T) {
		// arrange
		value := uuid.NewString()

		// act
		ak, err := apiKeyStore.ReadAPIKeyByValue(context.Background(), value)

		// assert
		assert.Error(t, err)
		assert.True(t, errors.Is(err, sql.ErrNoRows))
		assert.Nil(t, ak)
	})
}

func TestAPIKeySQLiteStore_DeleteAPIKey(t *testing.T) {
	t.Run("success - key is found and deleted", func(t *testing.T) {
		// arrange
		key := createAPIKey(t)

		// act
		delErr := apiKeyStore.DeleteAPIKey(context.Background(), key.ID)
		ak, readErr := apiKeyStore.ReadAPIKeyByID(context.Background(), key.ID)

		// assert
		assert.NoError(t, delErr)
		assert.Error(t, readErr)
		assert.ErrorIs(t, readErr, sql.ErrNoRows)
		assert.Nil(t, ak)
	})
	t.Run("failure - key is not found", func(t *testing.T) {
		// arrange
		var id int64 = 3432535

		// act
		err := apiKeyStore.DeleteAPIKey(context.Background(), id)

		// assert
		assert.Error(t, err)
		assert.ErrorIs(t, err, sql.ErrNoRows)
	})
}

func TestAPIKeySQLiteStore_ListAPIKeys(t *testing.T) {
	// arrange
	key := createAPIKey(t)

	// act
	keys, err := apiKeyStore.ListAPIKeys(context.Background())

	// assert
	assert.NoError(t, err)
	assert.NotNil(t, keys)
	assert.True(t, slices.ContainsFunc(keys, func(k *APIKey) bool {
		return k.ID == key.ID
	}))
}

func createAPIKey(t *testing.T) *APIKey {
	ak, err := apiKeyStore.CreateAPIKey(context.Background(), uuid.NewString())
	assert.NoError(t, err)
	return ak
}
