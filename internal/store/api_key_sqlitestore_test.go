package store

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type apiKeySQLiteStoreSuite struct {
	apiKeyStore *APIKeySQLiteStore
	db          *sql.DB
	suite.Suite
}

func TestAPIKeySQLiteStore(t *testing.T) {
	suite.Run(t, new(apiKeySQLiteStoreSuite))
}

func (suite *apiKeySQLiteStoreSuite) SetupSuite() {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		log.Fatal(err)
	}
	suite.db = db
	_, err = db.Exec("PRAGMA foreign_keys = ON;")
	if err != nil {
		log.Fatal(err)
	}

	RunMigrations(db, "../../migrations")

	suite.apiKeyStore = NewAPIKeySQLiteStore(db, db)
}

func (suite *apiKeySQLiteStoreSuite) TearDownSuite() {
	_ = suite.db.Close()
}

func (suite *apiKeySQLiteStoreSuite) TestAPIKeySQLiteStore_CreateAPIKey() {
	suite.Run("success - api key is created", func() {
		// arrange
		value := uuid.NewString()

		// act
		ak, err := suite.apiKeyStore.CreateAPIKey(context.Background(), value)

		// assert
		suite.NoError(err)
		suite.NotNil(ak)
		suite.Equal(value, ak.Value)
	})
}

func (suite *apiKeySQLiteStoreSuite) TestAPIKeySQLiteStore_ReadAPIKeyByID() {
	suite.Run("success - key is found by id", func() {
		// arrange
		expectedKey := suite.createAPIKey()

		// act
		ak, err := suite.apiKeyStore.ReadAPIKeyByID(context.Background(), expectedKey.ID)

		// assert
		suite.NoError(err)
		suite.NotNil(ak)
		suite.Equal(expectedKey.ID, ak.ID)
		suite.Equal(expectedKey.Value, ak.Value)
	})
	suite.Run("failure - key is not found by id", func() {
		// arrange
		var id int64 = 432512

		// act
		ak, err := suite.apiKeyStore.ReadAPIKeyByID(context.Background(), id)

		// assert
		suite.Error(err)
		suite.True(errors.Is(err, sql.ErrNoRows))
		suite.Nil(ak)
	})
}

func (suite *apiKeySQLiteStoreSuite) TestAPIKeySQLiteStore_ReadAPIKeyByValue() {
	suite.Run("success - key is found by value", func() {
		// arrange
		expectedKey := suite.createAPIKey()

		// act
		ak, err := suite.apiKeyStore.ReadAPIKeyByValue(context.Background(), expectedKey.Value)

		// assert
		suite.NoError(err)
		suite.NotNil(ak)
		suite.Equal(expectedKey.ID, ak.ID)
		suite.Equal(expectedKey.Value, ak.Value)
	})
	suite.Run("failure - key is not found by value", func() {
		// arrange
		value := uuid.NewString()

		// act
		ak, err := suite.apiKeyStore.ReadAPIKeyByValue(context.Background(), value)

		// assert
		suite.Error(err)
		suite.True(errors.Is(err, sql.ErrNoRows))
		suite.Nil(ak)
	})
}

func (suite *apiKeySQLiteStoreSuite) TestAPIKeySQLiteStore_DeleteAPIKey() {
	suite.Run("success - key is found and deleted", func() {
		// arrange
		key := suite.createAPIKey()

		// act
		delErr := suite.apiKeyStore.DeleteAPIKey(context.Background(), key.ID)
		ak, readErr := suite.apiKeyStore.ReadAPIKeyByID(context.Background(), key.ID)

		// assert
		suite.NoError(delErr)
		suite.Error(readErr)
		suite.ErrorIs(readErr, sql.ErrNoRows)
		suite.Nil(ak)
	})
	suite.Run("failure - key is not found", func() {
		// arrange
		var id int64 = 3432535

		// act
		err := suite.apiKeyStore.DeleteAPIKey(context.Background(), id)

		// assert
		suite.Error(err)
		suite.ErrorIs(err, sql.ErrNoRows)
	})
}

func (suite *apiKeySQLiteStoreSuite) TestAPIKeySQLiteStore_ListAPIKeys() {
	// arrange
	key := suite.createAPIKey()

	// act
	keys, err := suite.apiKeyStore.ListAPIKeys(context.Background())

	// assert
	suite.NoError(err)
	suite.NotNil(keys)
	suite.True(slices.ContainsFunc(keys, func(k *APIKey) bool {
		return k.ID == key.ID
	}))
}

func (suite *apiKeySQLiteStoreSuite) createAPIKey() *APIKey {
	ak, err := suite.apiKeyStore.CreateAPIKey(context.Background(), uuid.NewString())
	suite.NoError(err)
	return ak
}
