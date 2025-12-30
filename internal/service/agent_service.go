package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/haatos/simple-ci/internal/security"
	"github.com/haatos/simple-ci/internal/store"
	"golang.org/x/crypto/ssh"
)

type AgentWriter interface {
	CreateControllerAgent(context.Context) (*store.Agent, error)
	CreateAgent(
		context.Context,
		int64,
		string,
		string,
		string,
		string,
		string,
	) (*store.Agent, error)
	UpdateAgent(context.Context, int64, int64, string, string, string, string, string) error
	DeleteAgent(context.Context, int64) error
}

type AgentReader interface {
	ReadAgentByID(context.Context, int64) (*store.Agent, error)
	ListAgents(context.Context) ([]*store.Agent, error)
}

type AgentStore interface {
	AgentWriter
	AgentReader
}

type AgentService struct {
	agentStore      AgentStore
	credentialStore CredentialStore
	aesEncrypter    *security.AESEncrypter
}

func NewAgentService(
	s AgentStore,
	cs CredentialStore,
	aesEncrypter *security.AESEncrypter,
) *AgentService {
	return &AgentService{agentStore: s, credentialStore: cs, aesEncrypter: aesEncrypter}
}

func (s *AgentService) CreateControllerAgent(
	ctx context.Context,
) (*store.Agent, error) {
	a, err := s.agentStore.CreateControllerAgent(ctx)
	return a, err
}

func (s *AgentService) CreateAgent(
	ctx context.Context,
	agentCredentialID int64,
	name, hostname, workspace, description, osType string,
) (*store.Agent, error) {
	a, err := s.agentStore.CreateAgent(
		ctx,
		agentCredentialID,
		name,
		hostname,
		workspace,
		description,
		osType,
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
	credentials, err := s.credentialStore.ListCredentials(ctx)
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

	credentials, err := s.credentialStore.ListCredentials(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, nil, err
	}

	return agents, credentials, nil
}

func (s *AgentService) UpdateAgent(
	ctx context.Context,
	agentID int64,
	agentCredentialID int64,
	name, hostname, workspace, description, osType string,
) error {
	err := s.agentStore.UpdateAgent(
		ctx,
		agentID,
		agentCredentialID,
		name,
		hostname,
		workspace,
		description,
		osType,
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

	cred, err := s.credentialStore.ReadCredentialByID(ctx, *a.AgentCredentialID)
	if err != nil {
		return err
	}

	privateKey, err := s.aesEncrypter.DecryptAES(cred.SSHPrivateKeyHash)
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
