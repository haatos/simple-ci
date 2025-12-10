package store

import (
	"context"
	"database/sql"
	"runtime"

	"github.com/georgysavva/scany/v2/sqlscan"
)

type AgentSQLiteStore struct {
	rdb, rwdb *sql.DB
}

func NewAgentSQLiteStore(rdb, rwdb *sql.DB) *AgentSQLiteStore {
	return &AgentSQLiteStore{rdb, rwdb}
}

func (store *AgentSQLiteStore) CreateControllerAgent(
	ctx context.Context,
) (*Agent, error) {
	var osType string
	if runtime.GOOS == "windows" {
		osType = "windows"
	} else {
		osType = "unix"
	}
	a := &Agent{
		Name:        "Localhost",
		Hostname:    "localhost",
		Workspace:   "runs",
		Description: "Agent to run pipelines on the controller machine.",
		OSType:      osType,
	}
	query := `insert into agents (
		name,
		hostname,
		workspace,
		description,
		os_type
	)
	values ($1, $2, $3, $4, $5)
	returning agent_id`
	err := sqlscan.Get(
		ctx, store.rwdb, a, query,
		a.Name,
		a.Hostname,
		a.Workspace,
		a.Description,
		a.OSType,
	)
	return a, err
}

func (store *AgentSQLiteStore) CreateAgent(
	ctx context.Context,
	credentialID int64,
	name, hostname, workspace, description, osType string,
) (*Agent, error) {
	a := &Agent{
		AgentCredentialID: &credentialID,
		Name:              name,
		Hostname:          hostname,
		Workspace:         workspace,
		Description:       description,
		OSType:            osType,
	}
	query := `insert into agents (
		agent_credential_id,
		name,
		hostname,
		workspace,
		description,
		os_type
	)
	values ($1, $2, $3, $4, $5, $6)
	returning agent_id`
	err := sqlscan.Get(
		ctx, store.rwdb, a, query,
		a.AgentCredentialID,
		a.Name,
		a.Hostname,
		a.Workspace,
		a.Description,
		a.OSType,
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
	name, hostname, workspace, description, osType string,
) error {
	query := `update agents
	set agent_credential_id = $1,
		name = $2,
		hostname = $3,
		workspace = $4,
		description = $5,
		os_type = $6
	where agent_id = $7`
	_, err := store.rwdb.ExecContext(
		ctx, query,
		credentialID,
		name,
		hostname,
		workspace,
		description,
		osType,
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
