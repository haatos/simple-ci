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
		c := createCredential(t)
		name := "testuser agent"
		hostname := "localhost"
		workspace := "/tmp"
		description := "description"
		osType := "unix"

		// act
		a, err := agentStore.CreateAgent(
			context.Background(),
			c.CredentialID,
			name,
			hostname,
			workspace,
			description,
			osType,
		)

		// assert
		assert.NoError(t, err)
		assert.NotNil(t, c)
		assert.NotEqual(t, 0, a.AgentID)
		assert.Equal(t, name, a.Name)
		assert.Equal(t, hostname, a.Hostname)
		assert.Equal(t, workspace, a.Workspace)
		assert.Equal(t, description, a.Description)
		assert.Equal(t, osType, a.OSType)
	})
}

func TestAgentSQLiteStore_ReadAgentByID(t *testing.T) {
	t.Run("success - agent found", func(t *testing.T) {
		// arrange
		c := createCredential(t)
		expectedAgent := createAgent(t, c)
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
		c := createCredential(t)
		agent := createAgent(t, c)
		name := "updated name"
		hostname := "updated.host"
		workspace := "/updated/workspace"
		description := "updated description"
		osType := "windows"

		// act
		updateErr := agentStore.UpdateAgent(
			context.Background(),
			agent.AgentID,
			c.CredentialID,
			name,
			hostname,
			workspace,
			description,
			osType,
		)
		a, readErr := agentStore.ReadAgentByID(context.Background(), agent.AgentID)

		// assert
		assert.NoError(t, updateErr)
		assert.NoError(t, readErr)
		assert.Equal(t, name, a.Name)
		assert.Equal(t, hostname, a.Hostname)
		assert.Equal(t, workspace, a.Workspace)
		assert.Equal(t, description, a.Description)
		assert.Equal(t, osType, a.OSType)
	})
}

func TestAgentSQLiteStore_DeleteAgent(t *testing.T) {
	t.Run("success - agent is deleted", func(t *testing.T) {
		// arrange
		c := createCredential(t)
		expectedAgent := createAgent(t, c)

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
		c := createCredential(t)
		expectedAgent := createAgent(t, c)

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

func createAgent(t *testing.T, c *Credential) *Agent {
	a, err := agentStore.CreateAgent(
		context.Background(),
		c.CredentialID,
		fmt.Sprintf("agent%d", time.Now().UnixNano()),
		"localhost",
		fmt.Sprintf("/tmp%d", time.Now().UnixNano()),
		"",
		"unix",
	)
	assert.NoError(t, err)
	return a
}
