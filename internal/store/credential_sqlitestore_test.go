package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type credentialSQLiteStoreSuite struct {
	credentialStore *CredentialSQLiteStore
	db              *sql.DB
	suite.Suite
}

func TestCredentialSQLiteStore(t *testing.T) {
	suite.Run(t, new(credentialSQLiteStoreSuite))
}

func (suite *credentialSQLiteStoreSuite) SetupSuite() {
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

	suite.credentialStore = NewCredentialSQLiteStore(db, db)
}

func (suite *credentialSQLiteStoreSuite) TearDownSuite() {
	_ = suite.db.Close()
}

func (suite *credentialSQLiteStoreSuite) TestCredentialSQLiteStore_CreateCredential() {
	suite.Run("success - credential created", func() {
		// arrange
		expectedCredential := suite.createCredential()

		// act
		c, err := suite.credentialStore.CreateCredential(
			context.Background(),
			expectedCredential.Username,
			expectedCredential.Description,
			expectedCredential.SSHPrivateKeyHash,
		)

		// assert
		suite.NoError(err)
		suite.NotNil(c)
		suite.NotEqual(0, c.CredentialID)
		suite.Equal(expectedCredential.Username, c.Username)
		suite.Equal(expectedCredential.Description, c.Description)
		suite.Equal(expectedCredential.SSHPrivateKeyHash, c.SSHPrivateKeyHash)
	})
}

func (suite *credentialSQLiteStoreSuite) TestCredentialSQLiteStore_ReadCredentialByID() {
	suite.Run("success - credential found", func() {
		// arrange
		expectedCredential := suite.createCredential()

		// act
		c, err := suite.credentialStore.ReadCredentialByID(
			context.Background(),
			expectedCredential.CredentialID,
		)

		// assert
		suite.NoError(err)
		suite.NotNil(c)
		suite.Equal(expectedCredential.Username, c.Username)
		suite.Equal(expectedCredential.Description, c.Description)
		suite.Equal(expectedCredential.SSHPrivateKeyHash, c.SSHPrivateKeyHash)
	})
	suite.Run("failure - credential not found", func() {
		// arrange
		var id int64 = 43241

		// act
		c, err := suite.credentialStore.ReadCredentialByID(context.Background(), id)

		// assert
		suite.Error(err)
		suite.True(errors.Is(err, sql.ErrNoRows))
		suite.Nil(c)
	})
}

func (suite *credentialSQLiteStoreSuite) TestCredentialSQLiteStore_UpdateCredential() {
	suite.Run("success - credential updates", func() {
		// arrange
		credential := suite.createCredential()
		username := "updated username"
		description := "updated description"

		// act
		updateErr := suite.credentialStore.UpdateCredential(
			context.Background(),
			credential.CredentialID,
			username,
			description,
		)
		c, readErr := suite.credentialStore.ReadCredentialByID(
			context.Background(),
			credential.CredentialID,
		)

		// assert
		suite.NoError(updateErr)
		suite.NoError(readErr)
		suite.Equal(username, c.Username)
		suite.Equal(description, c.Description)
	})
}

func (suite *credentialSQLiteStoreSuite) TestCredentialSQLiteStore_DeleteCredential() {
	suite.Run("success - credential is deleted", func() {
		// arrange
		expectedCredential := suite.createCredential()

		// act
		deleteErr := suite.credentialStore.DeleteCredential(
			context.Background(),
			expectedCredential.CredentialID,
		)
		c, readErr := suite.credentialStore.ReadCredentialByID(
			context.Background(),
			expectedCredential.CredentialID,
		)

		// assert
		suite.NoError(deleteErr)
		suite.Error(readErr)
		suite.Nil(c)
	})
}

func (suite *credentialSQLiteStoreSuite) TestCredentialSQLiteStore_ListCredentials() {
	suite.Run("success - credentials found", func() {
		// arrange
		expectedCredential := suite.createCredential()

		// act
		credentials, err := suite.credentialStore.ListCredentials(context.Background())

		// assert
		suite.NoError(err)
		suite.True(len(credentials) >= 1)
		suite.True(slices.ContainsFunc(credentials, func(c *Credential) bool {
			return c.CredentialID == expectedCredential.CredentialID
		}))
	})
}

func (suite *credentialSQLiteStoreSuite) createCredential() *Credential {
	c, err := suite.credentialStore.CreateCredential(
		context.Background(),
		fmt.Sprintf("testuser%d", time.Now().UTC().UnixNano()),
		fmt.Sprintf("credential%d", time.Now().UTC().UnixNano()),
		"hash",
	)
	suite.NoError(err)
	return c
}
