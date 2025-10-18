package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/georgysavva/scany/v2/sqlscan"
	"github.com/haatos/simple-ci/internal"
)

type RunSQLiteStore struct {
	rdb, rwdb *sql.DB
}

func NewRunSQLiteStore(rdb, rwdb *sql.DB) *RunSQLiteStore {
	return &RunSQLiteStore{rdb, rwdb}
}

func (store *RunSQLiteStore) CreateRun(
	ctx context.Context,
	pipelineID int64,
	branch string,
) (*Run, error) {
	r := &Run{
		RunPipelineID: pipelineID,
		Branch:        branch,
		Status:        StatusQueued,
	}
	query := `insert into runs (
		run_pipeline_id,
		branch,
		status
	)
	values ($1, $2, $3)
	returning run_id, created_on`
	if err := sqlscan.Get(ctx, store.rwdb, r, query, r.RunPipelineID, r.Branch, r.Status); err != nil {
		return nil, err
	}
	return r, nil
}

func (store *RunSQLiteStore) ReadRunByID(ctx context.Context, id int64) (*Run, error) {
	r := &Run{RunID: id}
	query := "select * from runs where run_id = $1"
	if err := sqlscan.Get(ctx, store.rdb, r, query, r.RunID); err != nil {
		return nil, err
	}
	return r, nil
}

func (store *RunSQLiteStore) UpdateRunStartedOn(
	ctx context.Context,
	id int64,
	workingDirectory string,
	status RunStatus,
	startedOn *time.Time,
) error {
	query := `update runs
	set working_directory = $1,
		status = $2,
		started_on = $3
	where run_id = $4`
	_, err := store.rwdb.ExecContext(
		ctx, query,
		workingDirectory,
		status,
		startedOn.Format(internal.DBTimestampLayout),
		id,
	)
	return err
}

func (store *RunSQLiteStore) UpdateRunEndedOn(
	ctx context.Context,
	id int64,
	status RunStatus,
	output, artifacts *string,
	endedOn *time.Time,
) error {
	query := `update runs
	set status = $1,
		output = $2,
		artifacts = $3,
		ended_on = $4
	where run_id = $5`
	_, err := store.rwdb.ExecContext(
		ctx, query,
		status,
		output,
		artifacts,
		endedOn.Format(internal.DBTimestampLayout),
		id,
	)
	return err
}

func (store *RunSQLiteStore) AppendRunOutput(ctx context.Context, id int64, out string) error {
	tx, err := store.rwdb.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	r := &Run{RunID: id}
	readQuery := `select * from runs where run_id = $1`
	err = sqlscan.Get(ctx, tx, r, readQuery, r.RunID)
	if err != nil {
		return err
	}

	var existingOutput string
	if r.Output != nil {
		existingOutput = *r.Output
	}
	updateQuery := `update runs
	set output = $1
	where run_id = $2`
	_, err = tx.ExecContext(ctx, updateQuery, existingOutput+out, r.RunID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (store *RunSQLiteStore) DeleteRun(ctx context.Context, id int64) error {
	query := "delete from runs where run_id = $1"
	_, err := store.rwdb.ExecContext(ctx, query, id)
	return err
}

func (store *RunSQLiteStore) ListPipelineRuns(
	ctx context.Context,
	pipelineID int64,
) ([]Run, error) {
	query := `select * from runs
	where run_pipeline_id = $1`
	runs := make([]Run, 0)
	err := sqlscan.Select(ctx, store.rdb, &runs, query, pipelineID)
	return runs, err
}

func (store *RunSQLiteStore) ListPipelineRunsPaginated(
	ctx context.Context,
	pipelineID, limit, offset int64,
) ([]Run, error) {
	query := `select 
		r.run_id,
		r.run_pipeline_id,
		r.branch,
		r.status,
		r.created_on,
		r.started_on,
		r.ended_on,
		p.name as pipeline_name
	from runs r
	join pipelines p
	on r.run_pipeline_id = p.pipeline_id
	where run_pipeline_id = $1
	order by created_on desc limit $2 offset $3`
	runs := make([]Run, 0)
	err := sqlscan.Select(ctx, store.rdb, &runs, query, pipelineID, limit, offset)
	return runs, err
}

func (store *RunSQLiteStore) ListLatestPipelineRuns(
	ctx context.Context,
	pipelineID, limit int64,
) ([]Run, error) {
	query := `select * from runs
	where run_pipeline_id = $1
	order by created_on desc limit $2`
	runs := make([]Run, 0)
	err := sqlscan.Select(ctx, store.rdb, &runs, query, pipelineID, limit)
	return runs, err
}

func (store *RunSQLiteStore) CountPipelineRuns(
	ctx context.Context,
	pipelineID int64,
) (int64, error) {
	var count int64
	query := `select count(*) from runs where run_pipeline_id = $1`
	err := sqlscan.Get(ctx, store.rdb, &count, query, pipelineID)
	return count, err
}
