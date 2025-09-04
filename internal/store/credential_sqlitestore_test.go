package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCredentialSQLiteStore_CreateCredential(t *testing.T) {
	t.Run("success - credential created", func(t *testing.T) {
		// arrange
		username := "testuser"
		description := "create credential success"
		sshPrivateKeyHash := "sshprivatekeyhash"

		// act
		c, err := credentialStore.CreateCredential(
			context.Background(),
			username,
			description,
			sshPrivateKeyHash,
		)

		// assert
		assert.NoError(t, err)
		assert.NotNil(t, c)
		assert.NotEqual(t, 0, c.CredentialID)
		assert.Equal(t, username, c.Username)
		assert.Equal(t, description, c.Description)
		assert.Equal(t, username, c.Username)
		assert.Equal(t, sshPrivateKeyHash, c.SSHPrivateKeyHash)
	})
}

func TestCredentialSQLiteStore_ReadCredentialByID(t *testing.T) {
	t.Run("success - credential found", func(t *testing.T) {
		// arrange
		expectedCredential := generateCredential(t)

		// act
		c, err := credentialStore.ReadCredentialByID(
			context.Background(),
			expectedCredential.CredentialID,
		)

		// assert
		assert.NoError(t, err)
		assert.NotNil(t, c)
		assert.Equal(t, expectedCredential.Username, c.Username)
		assert.Equal(t, expectedCredential.Description, c.Description)
		assert.Equal(t, expectedCredential.SSHPrivateKeyHash, c.SSHPrivateKeyHash)
	})
	t.Run("failure - credential not found", func(t *testing.T) {
		// arrange
		var id int64 = 43241

		// act
		c, err := credentialStore.ReadCredentialByID(context.Background(), id)

		// assert
		assert.Error(t, err)
		assert.True(t, errors.Is(err, sql.ErrNoRows))
		assert.Nil(t, c)
	})
}

func TestCredentialSQLiteStore_UpdateCredential(t *testing.T) {
	t.Run("success - credential updates", func(t *testing.T) {
		// arrange
		expectedCredential := generateCredential(t)

		// act
		newUsername := "testuser.updated"
		newDescription := "testuser description updated"
		updateErr := credentialStore.UpdateCredential(
			context.Background(),
			expectedCredential.CredentialID,
			newUsername,
			newDescription,
		)
		c, readErr := credentialStore.ReadCredentialByID(
			context.Background(),
			expectedCredential.CredentialID,
		)

		// assert
		assert.NoError(t, updateErr)
		assert.NoError(t, readErr)
		assert.Equal(t, newUsername, c.Username)
		assert.Equal(t, newDescription, c.Description)
	})
}

func TestCredentialSQLiteStore_DeleteCredential(t *testing.T) {
	t.Run("success - credential is deleted", func(t *testing.T) {
		// arrange
		expectedCredential := generateCredential(t)

		// act
		deleteErr := credentialStore.DeleteCredential(
			context.Background(),
			expectedCredential.CredentialID,
		)
		c, readErr := credentialStore.ReadCredentialByID(
			context.Background(),
			expectedCredential.CredentialID,
		)

		// assert
		assert.NoError(t, deleteErr)
		assert.Error(t, readErr)
		assert.Nil(t, c)
	})
}

func TestCredentialSQLiteStore_ListCredentials(t *testing.T) {
	t.Run("success - credentials found", func(t *testing.T) {
		// arrange
		expectedCredential := generateCredential(t)

		// act
		credentials, err := credentialStore.ListCredentials(context.Background())

		// assert
		assert.NoError(t, err)
		assert.True(t, len(credentials) >= 1)
		assert.True(t, slices.ContainsFunc(credentials, func(c *Credential) bool {
			return c.CredentialID == expectedCredential.CredentialID
		}))
	})
}

func generateCredential(t *testing.T) *Credential {
	c, err := credentialStore.CreateCredential(
		context.Background(),
		"testuser"+fmt.Sprintf("%d", time.Now().UnixNano()),
		"credential "+fmt.Sprintf("%d", time.Now().UnixNano()),
		"hash",
	)
	assert.NoError(t, err)
	return c
}
