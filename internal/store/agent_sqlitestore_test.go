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

func TestAgentSQLiteStore_CreateAgent(t *testing.T) {
	t.Run("success - agent created", func(t *testing.T) {
		// arrange
		c := generateCredential(t)
		name := "agent name"
		hostname := "localhost"
		description := "create agent success"
		workspace := "~/tmp"

		// act
		a, err := agentStore.CreateAgent(
			context.Background(),
			c.CredentialID,
			name, hostname, workspace, description,
		)

		// assert
		assert.NoError(t, err)
		assert.NotNil(t, c)
		assert.NotEqual(t, 0, a.AgentID)
		assert.Equal(t, name, a.Name)
		assert.Equal(t, hostname, a.Hostname)
		assert.Equal(t, workspace, a.Workspace)
		assert.Equal(t, description, a.Description)
	})
}

func TestAgentSQLiteStore_ReadAgentByID(t *testing.T) {
	t.Run("success - agent found", func(t *testing.T) {
		// arrange
		c := generateCredential(t)
		expectedAgent := generateAgent(t, c)
		// act
		a, err := agentStore.ReadAgentByID(
			context.Background(),
			expectedAgent.AgentID,
		)

		// assert
		assert.NoError(t, err)
		assert.NotNil(t, c)
		assert.Equal(t, expectedAgent.Name, a.Name)
		assert.Equal(t, expectedAgent.Hostname, a.Hostname)
		assert.Equal(t, expectedAgent.Workspace, a.Workspace)
		assert.Equal(t, expectedAgent.Description, a.Description)
	})
	t.Run("failure - agent not found", func(t *testing.T) {
		// arrange
		var id int64 = 43241

		// act
		c, err := agentStore.ReadAgentByID(context.Background(), id)

		// assert
		assert.Error(t, err)
		assert.True(t, errors.Is(err, sql.ErrNoRows))
		assert.Nil(t, c)
	})
}

func TestAgentSQLiteStore_UpdateAgent(t *testing.T) {
	t.Run("success - agent updates", func(t *testing.T) {
		// arrange
		c := generateCredential(t)
		expectedAgent := generateAgent(t, c)

		// act
		newName := "agent updated"
		newHostname := "192.168.1.11"
		newWorkspace := "/opt"
		newDescription := "agent description updated"
		updateErr := agentStore.UpdateAgent(
			context.Background(),
			expectedAgent.AgentID,
			c.CredentialID,
			newName, newHostname, newWorkspace, newDescription,
		)
		a, readErr := agentStore.ReadAgentByID(
			context.Background(),
			expectedAgent.AgentID,
		)

		// assert
		assert.NoError(t, updateErr)
		assert.NoError(t, readErr)
		assert.Equal(t, newName, a.Name)
		assert.Equal(t, newHostname, a.Hostname)
		assert.Equal(t, newWorkspace, a.Workspace)
		assert.Equal(t, newDescription, a.Description)
	})
}

func TestAgentSQLiteStore_DeleteAgent(t *testing.T) {
	t.Run("success - agent is deleted", func(t *testing.T) {
		// arrange
		c := generateCredential(t)
		expectedAgent := generateAgent(t, c)

		// act
		deleteErr := agentStore.DeleteAgent(
			context.Background(),
			expectedAgent.AgentID,
		)
		a, readErr := agentStore.ReadAgentByID(
			context.Background(),
			expectedAgent.AgentID,
		)

		// assert
		assert.NoError(t, deleteErr)
		assert.Error(t, readErr)
		assert.Nil(t, a)
	})
}

func TestAgentSQLiteStore_ListAgents(t *testing.T) {
	t.Run("success - agents found", func(t *testing.T) {
		// arrange
		c := generateCredential(t)
		expectedAgent := generateAgent(t, c)

		// act
		credentials, err := agentStore.ListAgents(context.Background())

		// assert
		assert.NoError(t, err)
		assert.True(t, len(credentials) >= 1)
		assert.True(t, slices.ContainsFunc(credentials, func(c *Agent) bool {
			return c.AgentID == expectedAgent.AgentID
		}))
	})
}

func generateAgent(t *testing.T, c *Credential) *Agent {
	a, err := agentStore.CreateAgent(
		context.Background(),
		c.CredentialID,
		"pipeline"+fmt.Sprintf("%d", time.Now().UnixNano()),
		"localhost",
		"pipeline description",
		"~/tmp",
	)
	assert.NoError(t, err)
	return a
}
