package store

import (
	"context"
	"database/sql"

	"github.com/georgysavva/scany/v2/sqlscan"
)

type PipelineSQLiteStore struct {
	rdb, rwdb *sql.DB
}

func NewPipelineSQLiteStore(rdb, rwdb *sql.DB) *PipelineSQLiteStore {
	return &PipelineSQLiteStore{rdb, rwdb}
}

func (store *PipelineSQLiteStore) CreatePipeline(
	ctx context.Context,
	agentID int64,
	name, description, repository, scriptPath string,
) (*Pipeline, error) {
	p := &Pipeline{
		PipelineAgentID: agentID,
		Name:            name,
		Description:     description,
		Repository:      repository,
		ScriptPath:      scriptPath,
	}
	query := `insert into pipelines (
		pipeline_agent_id,
		name,
		description,
		repository,
		script_path
	)
	values ($1, $2, $3, $4, $5)
	returning pipeline_id`
	if err := sqlscan.Get(
		ctx, store.rwdb, p, query,
		p.PipelineAgentID,
		p.Name,
		p.Description,
		p.Repository,
		p.ScriptPath,
	); err != nil {
		return nil, err
	}
	return p, nil
}

func (store *PipelineSQLiteStore) ReadPipelineByID(
	ctx context.Context,
	id int64,
) (*Pipeline, error) {
	p := &Pipeline{PipelineID: id}
	query := "select * from pipelines where pipeline_id = $1"
	if err := sqlscan.Get(ctx, store.rdb, p, query, p.PipelineID); err != nil {
		return nil, err
	}
	return p, nil
}

func (store *PipelineSQLiteStore) ReadPipelineRunData(
	ctx context.Context,
	id int64,
) (*PipelineRunData, error) {
	prd := new(PipelineRunData)
	query := `select
		p.pipeline_id,
		p.repository,
		p.script_path,
		a.agent_id,
		a.os_type,
		a.hostname,
		a.workspace,
		c.credential_id,
		c.username,
		c.ssh_private_key_hash
	from pipelines p
	join agents a
	on p.pipeline_agent_id = a.agent_id
	join credentials c
	on a.agent_credential_id = c.credential_id
	where p.pipeline_id = $1`
	err := sqlscan.Get(ctx, store.rdb, prd, query, id)
	if err != nil {
		return nil, err
	}
	return prd, nil
}

func (store *PipelineSQLiteStore) UpdatePipeline(
	ctx context.Context,
	id, agentID int64,
	name, description, repository, scriptPath string,
) error {
	query := `update pipelines
	set pipeline_agent_id = $1,
		name = $2,
		description = $3,
		repository = $4,
		script_path = $5
	where pipeline_id = $6`
	_, err := store.rwdb.ExecContext(
		ctx, query,
		agentID,
		name,
		description,
		repository,
		scriptPath,
		id,
	)
	return err
}

func (store *PipelineSQLiteStore) DeletePipeline(ctx context.Context, id int64) error {
	query := "delete from pipelines where pipeline_id = $1"
	_, err := store.rwdb.ExecContext(ctx, query, id)
	return err
}

func (store *PipelineSQLiteStore) ListPipelines(ctx context.Context) ([]*Pipeline, error) {
	query := "select * from pipelines"
	pipelines := make([]*Pipeline, 0)
	err := sqlscan.Select(ctx, store.rdb, &pipelines, query)
	return pipelines, err
}

func (store *PipelineSQLiteStore) ListScheduledPipelines(ctx context.Context) ([]*Pipeline, error) {
	query := "select * from pipelines where schedule is not null"
	pipelines := make([]*Pipeline, 0)
	err := sqlscan.Select(ctx, store.rdb, &pipelines, query)
	return pipelines, err
}

func (store *PipelineSQLiteStore) UpdatePipelineSchedule(
	ctx context.Context,
	id int64,
	schedule, branch, jobID *string,
) error {
	query := `update pipelines
	set schedule = $1,
		schedule_branch = $2,
		schedule_job_id = $3
	where pipeline_id = $4`
	_, err := store.rwdb.ExecContext(ctx, query, schedule, branch, jobID, id)
	return err
}

func (store *PipelineSQLiteStore) UpdatePipelineScheduleJobID(
	ctx context.Context,
	id int64,
	jobID *string,
) error {
	query := `update pipelines
	set schedule_job_id = $1
	where pipeline_id = $2`
	_, err := store.rwdb.ExecContext(ctx, query, jobID, id)
	return err
}

func (store *PipelineSQLiteStore) BeginTx(ctx context.Context) (*sql.Tx, error) {
	tx, err := store.rwdb.BeginTx(ctx, nil)
	return tx, err
}
