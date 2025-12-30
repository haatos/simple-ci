package store

import (
	"context"
	"time"
)

type RunStatus string

const (
	StatusQueued    RunStatus = "queued"
	StatusRunning   RunStatus = "running"
	StatusCancelled RunStatus = "cancelled"
	StatusFailed    RunStatus = "failed"
	StatusPassed    RunStatus = "passed"
)

type Run struct {
	RunID            int64 `param:"run_id"`
	RunPipelineID    int64
	Branch           string
	WorkingDirectory *string
	Output           *string
	Artifacts        *string
	Status           RunStatus
	CreatedOn        time.Time
	StartedOn        *time.Time
	EndedOn          *time.Time
	Archive          bool

	PipelineName string
}

type RunStore interface {
	CreatePipelineRun(context.Context, int64, string) (*Run, error)
	ReadRunByID(context.Context, int64) (*Run, error)
	UpdatePipelineRunStartedOn(context.Context, int64, string, RunStatus, *time.Time) error
	UpdatePipelineRunEndedOn(context.Context, int64, RunStatus, *string, *time.Time) error
	AppendPipelineRunOutput(context.Context, int64, string) error
	DeletePipelineRun(context.Context, int64) error
	ListPipelineRuns(context.Context, int64) ([]Run, error)
	ListLatestPipelineRuns(context.Context, int64, int64) ([]Run, error)
	ListPipelineRunsPaginated(context.Context, int64, int64, int64) ([]Run, error)
	CountPipelineRuns(context.Context, int64) (int64, error)
}
