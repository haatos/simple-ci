package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/haatos/simple-ci/internal/store"
	"golang.org/x/crypto/ssh"
)

type AgentServicer interface {
	CreateAgent(
		ctx context.Context,
		agentCredentialID int64,
		name, hostname, workspace, description string,
	) (*store.Agent, error)
	GetAgentByID(context.Context, int64) (*store.Agent, error)
	GetAgentAndCredentials(context.Context, int64) (*store.Agent, []*store.Credential, error)
	ListAgents(context.Context) ([]*store.Agent, error)
	ListAgentsAndCredentials(context.Context) ([]*store.Agent, []*store.Credential, error)
	UpdateAgent(
		ctx context.Context,
		agentID int64, agentCredentialID int64,
		name, hostname, workspace, description string,
	) error
	DeleteAgent(context.Context, int64) error

	TestAgentConnection(context.Context, int64) error
}

type AgentService struct {
	agentStore store.AgentStore

	credentialService CredentialServicer
}

func NewAgentService(s store.AgentStore, cs CredentialServicer) *AgentService {
	return &AgentService{agentStore: s, credentialService: cs}
}

func (s *AgentService) CreateAgent(
	ctx context.Context,
	agentCredentialID int64,
	name, hostname, workspace, description string,
) (*store.Agent, error) {
	a, err := s.agentStore.CreateAgent(
		ctx,
		agentCredentialID,
		name,
		hostname,
		workspace,
		description,
	)
	return a, err
}

func (s *AgentService) GetAgentByID(ctx context.Context, agentID int64) (*store.Agent, error) {
	a, err := s.agentStore.ReadAgentByID(ctx, agentID)
	return a, err
}

func (s *AgentService) GetAgentAndCredentials(
	ctx context.Context,
	id int64,
) (*store.Agent, []*store.Credential, error) {
	a, err := s.agentStore.ReadAgentByID(ctx, id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, nil, err
	}
	credentials, err := s.credentialService.ListCredentials(ctx)
	if err != nil {
		return nil, nil, err
	}
	return a, credentials, nil
}

func (s *AgentService) ListAgents(ctx context.Context) ([]*store.Agent, error) {
	return s.agentStore.ListAgents(ctx)
}

func (s *AgentService) ListAgentsAndCredentials(
	ctx context.Context,
) ([]*store.Agent, []*store.Credential, error) {
	agents, err := s.agentStore.ListAgents(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, nil, err
	}

	credentials, err := s.credentialService.ListCredentials(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, nil, err
	}

	return agents, credentials, nil
}

func (s *AgentService) UpdateAgent(
	ctx context.Context,
	agentID int64,
	agentCredentialID int64,
	name, hostname, workspace, description string,
) error {
	err := s.agentStore.UpdateAgent(
		ctx,
		agentID,
		agentCredentialID,
		name,
		hostname,
		workspace,
		description,
	)
	return err
}

func (s *AgentService) DeleteAgent(ctx context.Context, agentID int64) error {
	return s.agentStore.DeleteAgent(ctx, agentID)
}

func (s *AgentService) TestAgentConnection(ctx context.Context, agentID int64) error {
	a, err := s.GetAgentByID(ctx, agentID)
	if err != nil {
		return err
	}

	cred, err := s.credentialService.GetCredentialByID(ctx, a.AgentCredentialID)
	if err != nil {
		return err
	}

	privateKey, err := s.credentialService.DecryptAES(cred.SSHPrivateKeyHash)
	if err != nil {
		return err
	}
	signer, err := ssh.ParsePrivateKey(privateKey)
	if err != nil {
		return err
	}
	auth := ssh.PublicKeys(signer)
	cc := &ssh.ClientConfig{
		User:            cred.Username,
		Auth:            []ssh.AuthMethod{auth},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	hostname := a.Hostname
	split := strings.Split(a.Hostname, ":")
	if len(split) == 1 {
		hostname += ":22"
	}

	client, err := ssh.Dial("tcp", hostname, cc)
	if err != nil {
		return err
	}
	defer client.Close()
	return nil
}
