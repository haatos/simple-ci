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

type agentSQLiteStoreSuite struct {
	agentStore *AgentSQLiteStore
	db         *sql.DB
	credential *Credential
	suite.Suite
}

func TestAgentSQLiteStore(t *testing.T) {
	suite.Run(t, new(agentSQLiteStoreSuite))
}

func (suite *agentSQLiteStoreSuite) SetupSuite() {
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

	credentialStore := NewCredentialSQLiteStore(db, db)
	c, err := credentialStore.CreateCredential(
		context.Background(),
		"agenttestuser",
		"",
		"agenttestuser",
	)
	if err != nil {
		log.Fatal(err)
	}
	suite.credential = c
	suite.agentStore = NewAgentSQLiteStore(db, db)
}

func (suite *agentSQLiteStoreSuite) TearDownSuite() {
	_ = suite.db.Close()
}

func (suite *agentSQLiteStoreSuite) TestAgentSQLiteStore_CreateAgent() {
	suite.Run("success - agent created", func() {
		// arrange
		name := "testuser agent"
		hostname := "localhost"
		workspace := "/tmp"
		description := "description"
		osType := "unix"

		// act
		a, err := suite.agentStore.CreateAgent(
			context.Background(),
			suite.credential.CredentialID,
			name,
			hostname,
			workspace,
			description,
			osType,
		)

		// assert
		suite.NoError(err)
		suite.NotNil(a)
		suite.NotEqual(0, a.AgentID)
		suite.Equal(name, a.Name)
		suite.Equal(hostname, a.Hostname)
		suite.Equal(workspace, a.Workspace)
		suite.Equal(description, a.Description)
		suite.Equal(osType, a.OSType)
	})
}

func (suite *agentSQLiteStoreSuite) TestAgentSQLiteStore_ReadAgentByID() {
	suite.Run("success - agent found", func() {
		// arrange
		expectedAgent := suite.createAgent()
		// act
		a, err := suite.agentStore.ReadAgentByID(
			context.Background(),
			expectedAgent.AgentID,
		)

		// assert
		suite.NoError(err)
		suite.NotNil(a)
		suite.Equal(expectedAgent.Name, a.Name)
		suite.Equal(expectedAgent.Hostname, a.Hostname)
		suite.Equal(expectedAgent.Workspace, a.Workspace)
		suite.Equal(expectedAgent.Description, a.Description)
	})
	suite.Run("failure - agent not found", func() {
		// arrange
		var id int64 = 43241

		// act
		c, err := suite.agentStore.ReadAgentByID(context.Background(), id)

		// assert
		suite.Error(err)
		suite.True(errors.Is(err, sql.ErrNoRows))
		suite.Nil(c)
	})
}

func (suite *agentSQLiteStoreSuite) TestAgentSQLiteStore_UpdateAgent() {
	suite.Run("success - agent updates", func() {
		// arrange
		agent := suite.createAgent()
		name := "updated name"
		hostname := "updated.host"
		workspace := "/updated/workspace"
		description := "updated description"
		osType := "windows"

		// act
		updateErr := suite.agentStore.UpdateAgent(
			context.Background(),
			agent.AgentID,
			suite.credential.CredentialID,
			name,
			hostname,
			workspace,
			description,
			osType,
		)
		a, readErr := suite.agentStore.ReadAgentByID(context.Background(), agent.AgentID)

		// assert
		suite.NoError(updateErr)
		suite.NoError(readErr)
		suite.Equal(name, a.Name)
		suite.Equal(hostname, a.Hostname)
		suite.Equal(workspace, a.Workspace)
		suite.Equal(description, a.Description)
		suite.Equal(osType, a.OSType)
	})
}

func (suite *agentSQLiteStoreSuite) TestAgentSQLiteStore_DeleteAgent() {
	suite.Run("success - agent is deleted", func() {
		// arrange
		expectedAgent := suite.createAgent()

		// act
		deleteErr := suite.agentStore.DeleteAgent(
			context.Background(),
			expectedAgent.AgentID,
		)
		a, readErr := suite.agentStore.ReadAgentByID(
			context.Background(),
			expectedAgent.AgentID,
		)

		// assert
		suite.NoError(deleteErr)
		suite.Error(readErr)
		suite.Nil(a)
	})
}

func (suite *agentSQLiteStoreSuite) TestAgentSQLiteStore_ListAgents() {
	suite.Run("success - agents found", func() {
		// arrange
		expectedAgent := suite.createAgent()

		// act
		credentials, err := suite.agentStore.ListAgents(context.Background())

		// assert
		suite.NoError(err)
		suite.True(len(credentials) >= 1)
		suite.True(slices.ContainsFunc(credentials, func(c *Agent) bool {
			return c.AgentID == expectedAgent.AgentID
		}))
	})
}

func (suite *agentSQLiteStoreSuite) createAgent() *Agent {
	a, err := suite.agentStore.CreateAgent(
		context.Background(),
		suite.credential.CredentialID,
		fmt.Sprintf("agent%d", time.Now().UnixNano()),
		"localhost",
		fmt.Sprintf("/tmp%d", time.Now().UnixNano()),
		"",
		"unix",
	)
	suite.NoError(err)
	return a
}
