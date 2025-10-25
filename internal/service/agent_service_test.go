package service

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/haatos/simple-ci/internal/store"
	"github.com/haatos/simple-ci/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockAgentStore struct {
	mock.Mock
}

func (m *MockAgentStore) CreateAgent(
	ctx context.Context,
	credentialID int64,
	name, hostname, workspace, description, osType string,
) (*store.Agent, error) {
	args := m.Called(ctx, name, hostname, workspace, description, osType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.Agent), args.Error(1)
}

func (m *MockAgentStore) ReadAgentByID(
	ctx context.Context,
	id int64,
) (*store.Agent, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.Agent), args.Error(1)
}

func (m *MockAgentStore) UpdateAgent(
	ctx context.Context,
	id int64,
	credentialID int64,
	name, hostname, workspace, description, osType string,
) error {
	args := m.Called(ctx, id, credentialID, name, hostname, workspace, description, osType)
	return args.Error(0)
}

func (m *MockAgentStore) DeleteAgent(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAgentStore) ListAgents(ctx context.Context) ([]*store.Agent, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*store.Agent), args.Error(1)
}

func TestAgentService_CreateAgent(t *testing.T) {
	t.Run("success - agent created", func(t *testing.T) {
		// arrange
		_, _, credential := generateCredential()
		expectedAgent := generateAgent(credential.CredentialID)
		mockStore := new(MockAgentStore)
		mockStore.On(
			"CreateAgent",
			context.Background(),
			expectedAgent.Name,
			expectedAgent.Hostname,
			expectedAgent.Workspace,
			expectedAgent.Description,
			expectedAgent.OSType,
		).Return(expectedAgent, nil)
		agentService := NewAgentService(mockStore, nil)

		// act
		agent, err := agentService.CreateAgent(
			context.Background(),
			expectedAgent.AgentCredentialID,
			expectedAgent.Name,
			expectedAgent.Hostname,
			expectedAgent.Workspace,
			expectedAgent.Description,
			expectedAgent.OSType,
		)

		// assert
		assert.NoError(t, err)
		assert.NotNil(t, agent)
		assert.Equal(t, expectedAgent.AgentCredentialID, agent.AgentCredentialID)
		assert.Equal(t, expectedAgent.Name, agent.Name)
		assert.Equal(t, expectedAgent.Hostname, agent.Hostname)
		assert.Equal(t, expectedAgent.Workspace, agent.Workspace)
		assert.Equal(t, expectedAgent.Description, agent.Description)
		assert.Equal(t, expectedAgent.OSType, agent.OSType)
	})
}

func TestAgentService_GetAgentByID(t *testing.T) {
	t.Run("success - agent is found", func(t *testing.T) {
		// arrange
		_, _, credential := generateCredential()
		expectedAgent := generateAgent(credential.CredentialID)
		mockStore := new(MockAgentStore)
		mockStore.On(
			"ReadAgentByID",
			context.Background(),
			expectedAgent.AgentID,
		).Return(expectedAgent, nil)
		agentService := NewAgentService(mockStore, nil)

		// act
		agent, err := agentService.GetAgentByID(context.Background(), expectedAgent.AgentID)

		// assert
		assert.NoError(t, err)
		assert.NotNil(t, agent)
	})
}

func TestAgentService_GetAgentAndCredentials(t *testing.T) {
	t.Run("success - agent is found", func(t *testing.T) {
		// arrange
		_, _, credential := generateCredential()
		expectedAgent := generateAgent(credential.CredentialID)
		expectedCredentials := []*store.Credential{credential}
		mockStore := new(MockAgentStore)
		mockStore.On(
			"ReadAgentByID",
			context.Background(),
			expectedAgent.AgentID,
		).Return(expectedAgent, nil)
		mockService := new(testutil.MockCredentialService)
		mockService.On(
			"ListCredentials", context.Background(),
		).Return(expectedCredentials, nil)
		agentService := NewAgentService(mockStore, mockService)

		// act
		agent, credentials, err := agentService.GetAgentAndCredentials(
			context.Background(),
			expectedAgent.AgentID,
		)

		// assert
		assert.NoError(t, err)
		assert.NotNil(t, agent)
		assert.NotNil(t, credentials)
		assert.Equal(t, len(expectedCredentials), len(credentials))
	})
}

func TestAgentService_UpdateAgent(t *testing.T) {
	t.Run("success - agent updated", func(t *testing.T) {
		// arrange
		_, _, credential := generateCredential()
		expectedAgent := generateAgent(credential.CredentialID)
		mockStore := new(MockAgentStore)
		mockStore.On(
			"UpdateAgent",
			context.Background(),
			expectedAgent.AgentID,
			expectedAgent.AgentCredentialID,
			expectedAgent.Name,
			expectedAgent.Hostname,
			expectedAgent.Workspace,
			expectedAgent.Description,
			expectedAgent.OSType,
		).Return(nil)
		agentService := NewAgentService(mockStore, nil)

		// act
		err := agentService.UpdateAgent(
			context.Background(),
			expectedAgent.AgentID,
			expectedAgent.AgentCredentialID,
			expectedAgent.Name,
			expectedAgent.Hostname,
			expectedAgent.Workspace,
			expectedAgent.Description,
			expectedAgent.OSType,
		)

		// assert
		assert.NoError(t, err)
	})
}

func TestAgentService_DeleteAgent(t *testing.T) {
	t.Run("success - agent is deleted", func(t *testing.T) {
		// arrange
		var agentID int64 = 1
		mockStore := new(MockAgentStore)
		mockStore.On(
			"DeleteAgent", context.Background(), agentID,
		).Return(nil)
		agentService := NewAgentService(mockStore, nil)

		// act
		err := agentService.DeleteAgent(context.Background(), agentID)

		// assert
		assert.NoError(t, err)
	})
}

func TestAgentService_ListAgents(t *testing.T) {
	t.Run("success - agents found", func(t *testing.T) {
		// arrange
		_, _, credential := generateCredential()
		expectedAgent := generateAgent(credential.CredentialID)
		expectedAgents := []*store.Agent{expectedAgent}
		mockStore := new(MockAgentStore)
		mockStore.On(
			"ListAgents", context.Background(),
		).Return(expectedAgents, nil)
		agentService := NewAgentService(mockStore, nil)

		// act
		agents, err := agentService.ListAgents(context.Background())

		// assert
		assert.NoError(t, err)
		assert.Equal(t, len(expectedAgents), len(agents))
	})
}

func TestAgentService_ListAgentsAndCredentials(t *testing.T) {
	t.Run("success - agents and credentials found", func(t *testing.T) {
		// arrange
		_, _, credential := generateCredential()
		expectedCredentials := []*store.Credential{credential}
		expectedAgent := generateAgent(credential.CredentialID)
		expectedAgents := []*store.Agent{expectedAgent}
		mockStore := new(MockAgentStore)
		mockStore.On(
			"ListAgents", context.Background(),
		).Return(expectedAgents, nil)
		mockService := new(testutil.MockCredentialService)
		mockService.On(
			"ListCredentials", context.Background(),
		).Return(expectedCredentials, nil)
		agentService := NewAgentService(mockStore, mockService)

		// act
		agents, credentials, err := agentService.ListAgentsAndCredentials(
			context.Background(),
		)

		// assert
		assert.NoError(t, err)
		assert.NotNil(t, agents)
		assert.NotNil(t, credentials)
		assert.Equal(t, len(expectedAgents), len(agents))
		assert.Equal(t, len(expectedCredentials), len(credentials))
	})
}

func generateAgent(credentialID int64) *store.Agent {
	if credentialID == 0 {
		credentialID = rand.Int63()
	}
	agent := &store.Agent{
		AgentID:           rand.Int63(),
		AgentCredentialID: credentialID,
		Name:              fmt.Sprintf("agent%d", time.Now().UnixNano()),
		Hostname:          "localhost",
		Workspace:         "/tmp",
		Description:       fmt.Sprintf("description%d", time.Now().UnixNano()),
		OSType:            "unix",
	}
	return agent
}
