package store

import (
	"context"
	"database/sql"

	"github.com/georgysavva/scany/v2/sqlscan"
)

type AgentSQLiteStore struct {
	rdb, rwdb *sql.DB
}

func NewAgentSQLiteStore(rdb, rwdb *sql.DB) *AgentSQLiteStore {
	return &AgentSQLiteStore{rdb, rwdb}
}

func (store *AgentSQLiteStore) CreateAgent(
	ctx context.Context,
	credentialID int64,
	name, hostname, workspace, description string,
) (*Agent, error) {
	a := &Agent{
		AgentCredentialID: credentialID,
		Name:              name,
		Hostname:          hostname,
		Workspace:         workspace,
		Description:       description,
	}
	query := `insert into agents (
		agent_credential_id,
		name,
		hostname,
		workspace,
		description
	)
	values ($1, $2, $3, $4, $5)
	returning agent_id`
	err := sqlscan.Get(
		ctx, store.rwdb, a, query,
		a.AgentCredentialID,
		a.Name,
		a.Hostname,
		a.Workspace,
		a.Description,
	)
	return a, err
}

func (store *AgentSQLiteStore) ReadAgentByID(ctx context.Context, id int64) (*Agent, error) {
	a := &Agent{AgentID: id}
	query := `select * from agents where agent_id = $1`
	err := sqlscan.Get(ctx, store.rdb, a, query, a.AgentID)
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (store *AgentSQLiteStore) UpdateAgent(
	ctx context.Context,
	agentID int64, credentialID int64,
	name, hostname, workspace, description string,
) error {
	query := `update agents
	set agent_credential_id = $1,
		name = $2,
		hostname = $3,
		workspace = $4,
		description = $5
	where agent_id = $6`
	_, err := store.rwdb.ExecContext(
		ctx, query,
		credentialID,
		name,
		hostname,
		workspace,
		description,
		agentID,
	)
	return err
}

func (store *AgentSQLiteStore) DeleteAgent(ctx context.Context, id int64) error {
	query := "delete from agents where agent_id = $1"
	_, err := store.rwdb.ExecContext(ctx, query, id)
	return err
}

func (store *AgentSQLiteStore) ListAgents(ctx context.Context) ([]*Agent, error) {
	query := `select * from agents`
	agents := make([]*Agent, 0)
	err := sqlscan.Select(ctx, store.rdb, &agents, query)
	return agents, err
}
