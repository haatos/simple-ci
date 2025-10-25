package testutil

import (
	"context"

	"github.com/haatos/simple-ci/internal/store"
	"github.com/stretchr/testify/mock"
)

type MockAgentService struct {
	mock.Mock
}

func (m *MockAgentService) CreateAgent(
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

func (m *MockAgentService) GetAgentByID(
	ctx context.Context,
	id int64,
) (*store.Agent, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.Agent), args.Error(1)
}

func (m *MockAgentService) GetAgentAndCredentials(
	ctx context.Context,
	id int64,
) (*store.Agent, []*store.Credential, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil || args.Get(1) == nil {
		return nil, nil, args.Error(1)
	}
	a := args.Get(0).(*store.Agent)
	credentials := args.Get(1).([]*store.Credential)
	return a, credentials, nil
}

func (m *MockAgentService) ListAgents(ctx context.Context) ([]*store.Agent, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*store.Agent), args.Error(1)
}

func (m *MockAgentService) ListAgentsAndCredentials(
	ctx context.Context,
) ([]*store.Agent, []*store.Credential, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil || args.Get(1) == nil {
		return nil, nil, args.Error(1)
	}
	agents := args.Get(0).([]*store.Agent)
	credentials := args.Get(1).([]*store.Credential)
	return agents, credentials, nil
}

func (m *MockAgentService) UpdateAgent(
	ctx context.Context,
	id int64,
	credentialID int64,
	name, hostname, workspace, description, osType string,
) error {
	args := m.Called(ctx, id, credentialID, name, hostname, workspace, description, osType)
	return args.Error(0)
}

func (m *MockAgentService) DeleteAgent(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAgentService) TestAgentConnection(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
